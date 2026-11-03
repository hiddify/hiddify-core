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
