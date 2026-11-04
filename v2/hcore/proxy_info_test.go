package hcore

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/common/monitoring"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/stretchr/testify/require"
)

// fakeGroup is an outbound group with fixed members and no selection.
type fakeGroup struct {
	tag string
	all []string
}

func (g *fakeGroup) Type() string                               { return "fake" }
func (g *fakeGroup) Tag() string                                { return g.tag }
func (g *fakeGroup) Network() []string                          { return []string{N.NetworkTCP} }
func (g *fakeGroup) Dependencies() []string                     { return g.all }
func (g *fakeGroup) Status() adapter.Status                     { return adapter.StatusReady }
func (g *fakeGroup) All() []string                              { return g.all }
func (g *fakeGroup) Selected(network string) adapter.Outbound   { return nil }
func (g *fakeGroup) AttachConnection(io.Closer) (detach func()) { return func() {} }
func (g *fakeGroup) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, errors.New("fake")
}
func (g *fakeGroup) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("fake")
}

func TestIsChainTag(t *testing.T) {
	require.True(t, isChainTag(config.ChainHopTag(config.ChainStatusExtraSecurity, config.ChainModePsiphon)))
	require.True(t, isChainTag(config.ChainHopTag(config.ChainStatusUnblocker, config.ChainModeWarp)))
	require.False(t, isChainTag("proxy-a"))
	require.False(t, isChainTag(config.OutboundSelectTag))
}

func TestTrimTagName(t *testing.T) {
	require.Equal(t, "grpc vmess tl CDN IP4", TrimTagName("grpc vmess tl CDN IP4  § 212"))
	require.Equal(t, "plain", TrimTagName("plain"))
	require.Equal(t, "hop", TrimTagName(" hop §hide§"))
}

// The best member is the one with the lowest real delay: no result and a failed test do not count.
func TestBestGroupMember(t *testing.T) {
	group := &fakeGroup{tag: "lowest", all: []string{"a", "b", "c", "d"}}
	history := urltest.NewHistoryStorage()
	history.StoreURLTestHistory("a", &adapter.URLTestHistory{Delay: 300})
	history.StoreURLTestHistory("b", &adapter.URLTestHistory{Delay: 120})
	history.StoreURLTestHistory("c", &adapter.URLTestHistory{Delay: unreachableDelay})
	require.Equal(t, "b", bestGroupMember(group, history))

	require.Equal(t, "", bestGroupMember(group, nil), "no history, no best member")
	empty := urltest.NewHistoryStorage()
	empty.StoreURLTestHistory("c", &adapter.URLTestHistory{Delay: unreachableDelay})
	require.Equal(t, "", bestGroupMember(group, empty), "a failed test is not a result")
}

// A balancer spreading connections (round robin) uses each member in turn: its row speaks for its
// fastest member, by ping, not for the one it happened to pick last.
func TestBalancerRowSpeaksForItsFastestProxy(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	plain := hotReloadSettings(t, freeTCPPort(t), "warn")
	settings := strings.Replace(plain, "{", `{"balancer-strategy":"round-robin",`, 1)
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: settings}, false)
	require.NoError(t, err)
	// the next tests get the balancer they expect
	t.Cleanup(func() {
		Stop()
		_, _ = ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: plain}, false)
	})
	_, err = StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 1081), socks("proxy-c", 1082))})
	require.NoError(t, err)

	a, b, c := runningOutbound(t, "proxy-a").Tag(), runningOutbound(t, "proxy-b").Tag(), runningOutbound(t, "proxy-c").Tag()
	waitForMonitorQuiet(t, a, b, c) // its own results of the refused proxies would overwrite the test's
	delays := map[string]uint16{a: 300, b: 120, c: 200}
	history := static.UrlTestHistory()
	hismap := map[string]*adapter.URLTestHistory{}
	for tag, delay := range delays {
		history.StoreURLTestHistory(tag, &adapter.URLTestHistory{Delay: delay})
		hismap[tag] = &adapter.URLTestHistory{Delay: delay}
	}
	// what the balancer itself reports: the member it picked last
	hismap[config.OutboundRoundRobinTag] = &adapter.URLTestHistory{Delay: 300}

	balance, loaded := static.Box().Outbound().Outbound(config.OutboundRoundRobinTag)
	require.True(t, loaded)
	require.Equal(t, b, static.speedTestTagOf(balance), "its speeds are its fastest member's")

	info := static.GetAllProxiesInfo(hismap, false)
	var row *OutboundInfo
	for _, group := range info.Items {
		for _, item := range group.Items {
			if item.Tag == config.OutboundRoundRobinTag {
				row = item
			}
		}
	}
	require.NotNil(t, row, "the balancer has a row")
	require.Equal(t, int32(120), row.UrlTestDelay, "its ping is its fastest member's")
}

// The home screen reads the main list, where each group shows the member it uses. A round robin
// balancer has no one member in use: what it picks next turns with every connection, so a member
// whose ping failed came up there in turn, and the home screen said the selected proxy was not
// responding while the balancer, by its ping, worked. It shows its fastest member instead.
func TestBalancerMainItemIsItsFastestProxy(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	plain := hotReloadSettings(t, freeTCPPort(t), "warn")
	settings := strings.Replace(plain, "{", `{"balancer-strategy":"round-robin",`, 1)
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: settings}, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		Stop()
		_, _ = ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: plain}, false)
	})
	_, err = StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 1081), socks("proxy-c", 1082))})
	require.NoError(t, err)

	balance, loaded := static.Box().Outbound().Outbound(config.OutboundRoundRobinTag)
	require.True(t, loaded)
	group := balance.(adapter.OutboundGroup)
	// the member round robin picks next failed its ping; of the other two, the fastest is c
	next := groupNow(group)
	require.NotEmpty(t, next)
	var others []string
	for _, name := range []string{"proxy-a", "proxy-b", "proxy-c"} {
		if tag := runningOutbound(t, name).Tag(); tag != next {
			others = append(others, tag)
		}
	}
	require.Len(t, others, 2)
	c := others[1]
	delays := map[string]uint16{next: unreachableDelay, others[0]: 300, c: 120}
	history := static.UrlTestHistory()
	hismap := map[string]*adapter.URLTestHistory{}
	for tag, delay := range delays {
		history.StoreURLTestHistory(tag, &adapter.URLTestHistory{Delay: delay})
		hismap[tag] = &adapter.URLTestHistory{Delay: delay}
	}
	require.Equal(t, next, groupNow(group), "round robin picks the dead member next")

	info := static.GetAllProxiesInfo(hismap, true)
	var items []*OutboundInfo
	for _, g := range info.Items {
		if g.Tag == config.OutboundRoundRobinTag {
			items = g.Items
		}
	}
	require.Len(t, items, 1, "the main list shows one member of the balancer")
	require.Equal(t, c, items[0].Tag, "the member it shows is its fastest")
	require.True(t, items[0].IsSelected)
}

// fakeFailing is an outbound that is not ready and says why.
type fakeFailing struct {
	fakeGroup
	err error
}

func (f *fakeFailing) Status() adapter.Status { return adapter.StatusFailed }
func (f *fakeFailing) StatusError() error     { return f.err }

// The app shows why a proxy failed: the error of its last URL test, or why it is not ready. A
// result from the cache has no error to show, as it has no ping.
func TestProxyInfoCarriesTheError(t *testing.T) {
	h := &HiddifyInstance{}
	ready := &fakeGroup{tag: "proxy"}
	info := h.GetProxyInfo(&adapter.URLTestHistory{Delay: unreachableDelay, Error: "tls: handshake failure"}, ready)
	require.Equal(t, "tls: handshake failure", info.Error)

	info = h.GetProxyInfo(&adapter.URLTestHistory{Delay: unreachableDelay, Error: "old", IsFromCache: true}, ready)
	require.Empty(t, info.Error, "a cached result is from an earlier run")

	info = h.GetProxyInfo(&adapter.URLTestHistory{Delay: 120}, ready)
	require.Empty(t, info.Error)

	failing := &fakeFailing{fakeGroup: fakeGroup{tag: "warp"}, err: errors.New("register WARP MASQUE device: 403")}
	info = h.GetProxyInfo(&adapter.URLTestHistory{Delay: unreachableDelay, Error: "dial: endpoint not initialized"}, failing)
	require.Equal(t, "register WARP MASQUE device: 403", info.Error, "why it is not ready comes first")
}

// A spreading balancer's row keeps the member it shows while another is faster by no more than
// the balancers' tolerance (the larger of 20 ms and 0.3 of its delay): taking the fastest at every result
// moved the row between members a few milliseconds apart, and the ping it showed jumped up and
// down. A member clearly faster, or the shown one failing, moves it.
func TestSpreadMemberKeepsItsMemberWithinTheTolerance(t *testing.T) {
	group := &fakeGroup{tag: "spread-test", all: []string{"a", "b", "c"}}
	history := urltest.NewHistoryStorage()
	set := func(delays map[string]uint16) {
		for tag, delay := range delays {
			storePing(history, tag, delay, false)
		}
	}
	set(map[string]uint16{"a": 29, "b": 44, "c": 33})
	require.Equal(t, "a", spreadMember(group, history), "the fastest at first")
	set(map[string]uint16{"a": 44, "b": 33, "c": 40})
	require.Equal(t, "a", spreadMember(group, history), "11 ms faster is within the 20 ms floor")
	set(map[string]uint16{"a": 600, "b": 33, "c": 40})
	require.Equal(t, "b", spreadMember(group, history), "far faster")
	set(map[string]uint16{"a": 1020, "b": 1000, "c": 940})
	require.Equal(t, "b", spreadMember(group, history), "60 ms faster is within 0.3 of 1000 ms")
	set(map[string]uint16{"a": 30, "b": unreachableDelay, "c": 940})
	require.Equal(t, "a", spreadMember(group, history), "the shown one failed: the fastest")
}

// A real ping comes before every cached one in the row, and a cached one before a failed one; the
// tolerance keeps only a real ping.
func TestSpreadMemberRanksRealPingsBeforeCached(t *testing.T) {
	group := &fakeGroup{tag: "rank-test", all: []string{"a", "b", "c"}}
	history := urltest.NewHistoryStorage()
	set := func(tag string, delay uint16, cached bool) { storePing(history, tag, delay, cached) }
	set("a", 15, true)
	set("b", 40, true)
	require.Equal(t, "a", spreadMember(group, history), "only cached pings: the best of them")
	set("b", 10, true)
	require.Equal(t, "b", spreadMember(group, history), "a cached one is left for any better")
	set("c", 300, false)
	require.Equal(t, "c", spreadMember(group, history), "a real ping before every cached one")
	set("a", 260, false)
	require.Equal(t, "c", spreadMember(group, history), "between real pings, the tolerance")
	set("c", unreachableDelay, false)
	require.Equal(t, "a", spreadMember(group, history), "a failed one is left for a real ping that works")
	set("a", unreachableDelay, false)
	require.Equal(t, "b", spreadMember(group, history), "with no real ping that works, a cached one")
}

// storePing records a result for tag with the flag given: the storage's own store keeps an
// existing entry's flags and updates its delay only.
func storePing(history adapter.URLTestHistoryStorage, tag string, delay uint16, cached bool) {
	if his := history.LoadURLTestHistory(tag); his != nil {
		his.Delay, his.IsFromCache = delay, cached
		return
	}
	history.StoreURLTestHistory(tag, &adapter.URLTestHistory{Delay: delay, IsFromCache: cached})
}

// waitForMonitorQuiet waits until the running monitor has a result for each tag and has written
// none for a while: the first cycle and the selection's staged test are over.
func waitForMonitorQuiet(t *testing.T, tags ...string) {
	t.Helper()
	monitor := monitoring.Get(static.Context())
	require.NotNil(t, monitor)
	last := map[string]time.Time{}
	quietSince := time.Now()
	require.Eventually(t, func() bool {
		histories := monitor.OutboundsHistory("")
		for _, tag := range tags {
			his := histories[tag]
			if his == nil || his.Delay == 0 {
				quietSince = time.Now()
				return false
			}
			if !his.Time.Equal(last[tag]) {
				last[tag] = his.Time
				quietSince = time.Now()
			}
		}
		return time.Since(quietSince) > 1500*time.Millisecond
	}, 20*time.Second, 50*time.Millisecond, "the monitor did not go quiet")
}
