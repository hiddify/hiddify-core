package hcore

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/stretchr/testify/require"
)

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

var clashPort int

var otherPorts []int

func hotReloadSettings(t *testing.T, mixedPort int, logLevel string) string {
	if clashPort == 0 {
		clashPort = freeTCPPort(t)
		otherPorts = []int{freeTCPPort(t), freeTCPPort(t), freeTCPPort(t)}
	}
	return fmt.Sprintf(`{"mixed-port":%d,"tproxy-port":%d,"redirect-port":%d,"direct-port":%d,"clash-api-port":%d,"log-level":%q}`,
		mixedPort, otherPorts[0], otherPorts[1], otherPorts[2], clashPort, logLevel)
}

func listening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprint("127.0.0.1:", port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func profile(outbounds ...string) string {
	return `{"outbounds":[` + strings.Join(outbounds, ",") + `]}`
}

func socks(tag string, port int) string {
	return fmt.Sprintf(`{"type":"socks","tag":%q,"server":"127.0.0.1","server_port":%d}`, tag, port)
}

func runningOutbound(t *testing.T, name string) adapter.Outbound {
	t.Helper()
	for _, outbound := range static.StartedService.Instance().Box().Outbound().Outbounds() {
		if strings.HasPrefix(outbound.Tag(), name) {
			return outbound
		}
	}
	return nil
}

func TestHotReload(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	mixedPort := freeTCPPort(t)
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: hotReloadSettings(t, mixedPort, "warn")}, false)
	require.NoError(t, err)
	t.Cleanup(func() { Stop() })

	// hot reload while stopped starts normally
	response, err := HotReload(static.BaseContext, &StartRequest{
		ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 1081)),
	})
	require.NoError(t, err)
	require.Equal(t, CoreStates_STARTED, response.CoreState)
	instance := static.StartedService.Instance()
	proxyA := runningOutbound(t, "proxy-a")
	proxyB := runningOutbound(t, "proxy-b")
	require.NotNil(t, proxyA)
	require.NotNil(t, proxyB)
	require.Nil(t, runningOutbound(t, "proxy-c"))

	// only outbounds change: applied to the running core, reported as HOT_RELOADING meanwhile
	states := static.coreInfoObserver.Subscribe(16)
	defer static.coreInfoObserver.Unsubscribe(states)
	response, err = HotReload(static.BaseContext, &StartRequest{
		ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 2081), socks("proxy-c", 1082)),
	})
	require.NoError(t, err)
	require.Equal(t, CoreStates_STARTED, response.CoreState)
	require.Equal(t, MessageType_EMPTY, response.MessageType)
	require.Same(t, instance, static.StartedService.Instance(), "the core was not restarted")
	require.Same(t, proxyA, runningOutbound(t, "proxy-a"), "an unchanged outbound keeps running")
	require.NotSame(t, proxyB, runningOutbound(t, "proxy-b"), "a changed outbound is replaced")
	require.NotNil(t, runningOutbound(t, "proxy-c"), "a new outbound is added")
	var seen []CoreStates
	for len(seen) == 0 || seen[len(seen)-1] != CoreStates_STARTED {
		select {
		case info := <-states:
			seen = append(seen, info.CoreState)
		case <-time.After(5 * time.Second):
			t.Fatalf("core states seen: %v", seen)
		}
	}
	require.Equal(t, CoreStates_HOT_RELOADING, seen[0], "states: %v", seen)

	// a normal start while started is still refused
	response, err = StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080))})
	require.NoError(t, err)
	require.Equal(t, MessageType_ALREADY_STARTED, response.MessageType)

	// an inbound changes (the mixed port setting): applied too
	newMixedPort := freeTCPPort(t)
	_, err = ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: hotReloadSettings(t, newMixedPort, "warn")}, false)
	require.NoError(t, err)
	require.True(t, listening(mixedPort))
	response, err = HotReload(static.BaseContext, &StartRequest{
		ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 2081), socks("proxy-c", 1082)),
	})
	require.NoError(t, err)
	require.Equal(t, MessageType_EMPTY, response.MessageType)
	require.Same(t, instance, static.StartedService.Instance())
	require.Same(t, proxyA, runningOutbound(t, "proxy-a"))
	require.True(t, listening(newMixedPort), "the inbound moved to the new port")
	require.False(t, listening(mixedPort))

	// something that cannot be hot reloaded changes (the log level): refused, core unchanged
	_, err = ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: hotReloadSettings(t, newMixedPort, "error")}, false)
	require.NoError(t, err)
	proxyC := runningOutbound(t, "proxy-c")
	response, err = HotReload(static.BaseContext, &StartRequest{
		ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 2081)),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "hot reload cannot change: log")
	require.Equal(t, MessageType_HOT_RELOAD_FAILED, response.MessageType)
	require.Equal(t, CoreStates_STARTED, response.CoreState)
	require.Equal(t, CoreStates_STARTED, static.CoreState)
	require.Same(t, instance, static.StartedService.Instance())
	require.Same(t, proxyC, runningOutbound(t, "proxy-c"), "nothing was applied")

	// the TUN inbound changes (here: TUN is turned on): refused so the app restarts instead
	_, err = ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: strings.Replace(hotReloadSettings(t, newMixedPort, "warn"), "{", `{"enable-tun":true,`, 1)}, false)
	require.NoError(t, err)
	response, err = HotReload(static.BaseContext, &StartRequest{
		ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 2081), socks("proxy-c", 1082)),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "tun inbound has changed, hot reload can not be applied")
	require.Equal(t, MessageType_HOT_RELOAD_FAILED, response.MessageType)
	require.Equal(t, CoreStates_STARTED, static.CoreState)
	require.Same(t, instance, static.StartedService.Instance())

	response, err = Stop()
	require.NoError(t, err)
	require.Equal(t, CoreStates_STOPPED, response.CoreState)
}
