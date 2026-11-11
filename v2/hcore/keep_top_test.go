package hcore

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	"github.com/stretchr/testify/require"
)

// keepTopCore starts a core on a profile with the keep top settings given (JSON fields), in a
// working directory of its own, and stops it with the test.
func keepTopCore(t *testing.T, settings string, outbounds ...string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{
		HiddifySettingsJson: strings.TrimSuffix(hotReloadSettings(t, freeTCPPort(t), "warn"), "}") + "," + settings + "}",
	}, false)
	require.NoError(t, err)
	t.Cleanup(func() { Stop() })
	path := filepath.Join(dir, "profile.json")
	require.NoError(t, os.WriteFile(path, []byte(profile(outbounds...)), 0o600))
	_, err = StartService(static.BaseContext, &StartRequest{ConfigPath: path})
	require.NoError(t, err)
	return path
}

// keep-top answering (-2) is below 0, and the first check took it for "run them all": the core
// never picked, and the proxies that did not answer kept running.
func TestKeepTopAnsweringPicks(t *testing.T) {
	keepTopCore(t, fmt.Sprintf(`"keep-top":%d`, config.KeepTopAnswering), socks("proxy-a", 1080), socks("proxy-b", 1081), socks("proxy-c", 1082))
	require.True(t, monitoring.Get(static.Context()).KeepTopSearching(), "the core looks for the ones that answer")
	for tag, outbound := range profileProxies() {
		_, isLazy := outbound.(adapter.LazyOutbound)
		require.True(t, isLazy, tag)
	}
}

// keep-top-countries keeps top on whatever the count: with "all", every proxy of those countries.
func TestKeepTopCountriesPicks(t *testing.T) {
	keepTopCore(t, `"keep-top":-1,"keep-top-countries":["de"]`, socks("proxy-a", 1080), socks("proxy-b", 1081))
	require.True(t, monitoring.Get(static.Context()).KeepTopSearching(), "the core looks for the ones of those countries")
}

// A profile with no more proxies than keep-top keeps uses them all: none rests.
func TestKeepTopSmallProfileUsesAll(t *testing.T) {
	keepTopCore(t, `"keep-top":3`, socks("proxy-a", 1080), socks("proxy-b", 1081))
	require.False(t, monitoring.Get(static.Context()).KeepTopSearching())
	used, _ := usedProxies()
	require.Equal(t, []string{"proxy-a", "proxy-b"}, used)
}

// Changing keep-top in the settings changes how many proxies the running core uses, and nothing
// else: the core is not restarted, no outbound is replaced, and no proxy is tested again to keep
// fewer.
func TestKeepTopCountChangesInPlace(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a core through a pick")
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	t.Cleanup(target.Close)
	outbounds := []string{socks("proxy-0", freeTCPPort(t))} // does not answer
	for i, delay := range []time.Duration{120 * time.Millisecond, 20 * time.Millisecond, 60 * time.Millisecond} {
		outbounds = append(outbounds, socks(fmt.Sprint("proxy-", i+1), socksServer(t, delay)))
	}
	settings := func(count int) string {
		return fmt.Sprintf(`"keep-top":%d,"connection-test-url":%q`, count, target.URL)
	}
	path := keepTopCore(t, settings(3), outbounds...)
	usedIs := func(want ...string) func() bool {
		return func() bool {
			used, _ := usedProxies()
			return !monitoring.Get(static.Context()).KeepTopSearching() && slices.Equal(used, want)
		}
	}
	require.Eventually(t, usedIs("proxy-1", "proxy-2", "proxy-3"), 60*time.Second, 20*time.Millisecond)
	instance, atStart := static.StartedService.Instance(), profileProxies()
	base := strings.TrimSuffix(hotReloadSettings(t, int(static.HiddifyOptions.MixedPort), "warn"), "}") + ","
	change := func(count int) {
		t.Helper()
		_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: base + settings(count) + "}"}, false)
		require.NoError(t, err)
		response, err := HotReload(static.BaseContext, &StartRequest{ConfigPath: path})
		require.NoError(t, err)
		require.Equal(t, MessageType_EMPTY, response.MessageType)
		require.Same(t, instance, static.StartedService.Instance(), "the core was not restarted")
		for tag, outbound := range profileProxies() {
			require.Same(t, atStart[tag], outbound, "%s was not replaced", tag)
		}
	}

	change(2)
	require.True(t, usedIs("proxy-2", "proxy-3")(), "at once: the fastest two of the three it had, by the pings it has")
	change(1)
	require.True(t, usedIs("proxy-2")())
	change(3) // more: a new pick finds the others, and the one in use stays in use meanwhile
	used, _ := usedProxies()
	require.Contains(t, used, "proxy-2")
	require.Eventually(t, usedIs("proxy-1", "proxy-2", "proxy-3"), 60*time.Second, 20*time.Millisecond)
	change(10) // more than there are: none rests
	require.True(t, usedIs("proxy-0", "proxy-1", "proxy-2", "proxy-3")())

	// the app is told which rest, and the one the user keeps loaded is used whatever its ping
	change(1) // from none resting: a pick again
	require.Eventually(t, usedIs("proxy-2"), 60*time.Second, 20*time.Millisecond)
	resting := func() []string {
		var tags []string
		for _, group := range static.GetAllProxiesInfo(monitoring.Get(static.Context()).OutboundsHistory(""), false).Items {
			require.False(t, group.KeepTopPicking)
			if group.Tag != config.OutboundSelectTag {
				continue
			}
			for _, item := range group.Items {
				if item.Resting {
					tags = append(tags, item.Tag)
				}
			}
		}
		slices.Sort(tags)
		return tags
	}
	require.Equal(t, []string{"proxy-0", "proxy-1", "proxy-3"}, resting())
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: base + settings(1) + `,"keep-top-pinned":["proxy-0"]}`}, false)
	require.NoError(t, err)
	_, err = HotReload(static.BaseContext, &StartRequest{ConfigPath: path})
	require.NoError(t, err)
	require.Same(t, instance, static.StartedService.Instance())
	require.True(t, usedIs("proxy-0", "proxy-2")(), "kept loaded beside the fastest, though it does not answer")
	require.Equal(t, []string{"proxy-1", "proxy-3"}, resting())
}

func socksServer(t *testing.T, delay time.Duration) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				buffer := make([]byte, 262)
				if _, err := io.ReadFull(conn, buffer[:2]); err != nil {
					return
				}
				if _, err := io.ReadFull(conn, buffer[:buffer[1]]); err != nil {
					return
				}
				conn.Write([]byte{5, 0})
				if _, err := io.ReadFull(conn, buffer[:4]); err != nil {
					return
				}
				var host string
				switch buffer[3] {
				case 1:
					io.ReadFull(conn, buffer[:4])
					host = net.IP(buffer[:4]).String()
				case 3:
					io.ReadFull(conn, buffer[:1])
					n := int(buffer[0])
					io.ReadFull(conn, buffer[:n])
					host = string(buffer[:n])
				case 4:
					io.ReadFull(conn, buffer[:16])
					host = net.IP(buffer[:16]).String()
				}
				if _, err := io.ReadFull(conn, buffer[:2]); err != nil {
					return
				}
				port := int(buffer[0])<<8 | int(buffer[1])
				target, err := net.Dial("tcp", net.JoinHostPort(host, fmt.Sprint(port)))
				if err != nil {
					conn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
					return
				}
				defer target.Close()
				conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
				go io.Copy(target, conn)
				// every answer is late by delay: that is the proxy's ping
				for {
					n, err := target.Read(buffer)
					if n > 0 {
						time.Sleep(delay)
						if _, err := conn.Write(buffer[:n]); err != nil {
							return
						}
					}
					if err != nil {
						return
					}
				}
			}()
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port
}

// profileProxies are the profile's proxies in the core, by tag.
func profileProxies() map[string]adapter.Outbound {
	proxies := make(map[string]adapter.Outbound)
	for _, outbound := range static.Box().Outbound().Outbounds() {
		if strings.HasPrefix(outbound.Tag(), "proxy-") {
			proxies[outbound.Tag()] = outbound
		}
	}
	return proxies
}

// usedProxies are the profile's proxies the core uses (the others rest), and the ones of the
// profile whose outbound exists now.
func usedProxies() (used, created []string) {
	monitor := monitoring.Get(static.Context())
	for tag, outbound := range profileProxies() {
		if !monitor.Parked(tag) {
			used = append(used, tag)
		}
		if lazy, isLazy := outbound.(adapter.LazyOutbound); !isLazy || lazy.Loaded() {
			created = append(created, tag)
		}
	}
	slices.Sort(used)
	slices.Sort(created)
	return used, created
}

// The core goes through a profile a few proxies at a time and ends with its fastest. The config
// has them all from the start and never changes: the ones not in use are not created. At no time
// are more in use, or created, than the kept ones and a batch.
func TestKeepTopLoadsInStages(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a core through several stages")
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	t.Cleanup(target.Close)
	// eight proxies: three answer, the fastest two of them last in the profile
	delays := map[int]time.Duration{2: 400 * time.Millisecond, 5: 150 * time.Millisecond, 7: 20 * time.Millisecond}
	var outbounds []string
	for i := range 8 {
		port := freeTCPPort(t) // nothing listens: the proxy does not answer
		if delay, alive := delays[i]; alive {
			port = socksServer(t, delay)
		}
		outbounds = append(outbounds, socks(fmt.Sprint("proxy-", i), port))
	}
	path := keepTopCore(t, fmt.Sprintf(`"keep-top":2,"keep-top-batch":3,"connection-test-url":%q`, target.URL), outbounds...)
	monitor := monitoring.Get(static.Context())
	atStart := profileProxies()
	require.Len(t, atStart, 8, "the config has the whole profile")
	mostUsed, mostCreated := 0, 0
	deadline := time.Now().Add(90 * time.Second)
	for {
		used, created := usedProxies()
		mostUsed, mostCreated = max(mostUsed, len(used)), max(mostCreated, len(created))
		if !monitor.KeepTopSearching() && len(used) == 2 {
			break
		}
		require.True(t, time.Now().Before(deadline), "the pick ends; using %v", used)
		time.Sleep(20 * time.Millisecond)
	}
	require.LessOrEqual(t, mostUsed, 5, "never more in use than the two kept and a batch of three")
	require.LessOrEqual(t, mostCreated, 5, "never more created than that")
	used, _ := usedProxies()
	require.Equal(t, []string{"proxy-5", "proxy-7"}, used, "the two fastest stay")
	require.Eventually(t, func() bool { return currentCoreState() == CoreStates_STARTED }, 10*time.Second, 50*time.Millisecond)
	for tag, outbound := range profileProxies() {
		require.Same(t, atStart[tag], outbound, "nothing was reloaded: %s is the outbound of the start", tag)
	}
	require.Eventually(t, func() bool {
		_, created := usedProxies()
		for _, tag := range created {
			if tag != "proxy-5" && tag != "proxy-7" {
				return false
			}
		}
		return true
	}, 10*time.Second, 50*time.Millisecond, "the ones at rest are closed")

	// the user selects one that rests: it is used, beside the kept ones
	selector, _ := static.Box().Outbound().Outbound(config.OutboundSelectTag)
	require.True(t, selector.(interface{ SelectOutbound(string) bool }).SelectOutbound("proxy-2"))
	require.Eventually(t, func() bool {
		used, _ := usedProxies()
		return slices.Equal(used, []string{"proxy-2", "proxy-5", "proxy-7"})
	}, 20*time.Second, 50*time.Millisecond)

	// another run ends on the same, with the selection the selector remembers
	_, err := Stop()
	require.NoError(t, err)
	_, err = StartService(static.BaseContext, &StartRequest{ConfigPath: path})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		used, _ := usedProxies()
		return !monitoring.Get(static.Context()).KeepTopSearching() && slices.Equal(used, []string{"proxy-2", "proxy-5", "proxy-7"})
	}, 60*time.Second, 50*time.Millisecond)
}
