package hcore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	N "github.com/sagernet/sing/common/network"
	"github.com/stretchr/testify/require"
)

func selectedTag(t *testing.T) string {
	t.Helper()
	outbound, loaded := static.Box().Outbound().Outbound(config.OutboundSelectTag)
	require.True(t, loaded)
	return outbound.(adapter.OutboundGroup).Selected(N.NetworkTCP).Tag()
}

func reportedSelected(t *testing.T) string {
	t.Helper()
	info := static.GetAllProxiesInfo(monitoring.Get(static.Context()).OutboundsHistory(""), true)
	for _, group := range info.Items {
		if group.Tag == config.OutboundSelectTag {
			return group.Selected
		}
	}
	return ""
}

// after a hot reload, selecting a proxy changes the selector and what the app is told
func TestHotReloadThenSelect(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: hotReloadSettings(t, freeTCPPort(t), "warn")}, false)
	require.NoError(t, err)
	t.Cleanup(func() { Stop() })

	_, err = StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 1081))})
	require.NoError(t, err)
	_, err = HotReload(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 2081), socks("proxy-c", 1082))})
	require.NoError(t, err)

	// the app selects in the first group of the proxy list (its main group): replaced groups must
	// keep their place, otherwise another group (e.g. the balancer) comes first after a hot reload
	groups := static.GetAllProxiesInfo(monitoring.Get(static.Context()).OutboundsHistory(""), false)
	require.NotEmpty(t, groups.Items)
	mainGroup := groups.Items[0].Tag
	require.Equal(t, config.OutboundSelectTag, mainGroup, "the main group stays first after a hot reload")

	target := runningOutbound(t, "proxy-c").Tag()
	events, err := monitoring.Get(static.Context()).SubscribeGroup("")
	require.NoError(t, err)
	_, err = static.SelectOutbound(&SelectOutboundRequest{GroupTag: mainGroup, OutboundTag: target})
	require.NoError(t, err)
	require.Equal(t, target, selectedTag(t), "the selector uses the selection")
	select {
	case <-events:
	case <-time.After(5 * time.Second):
		t.Fatal("no group event after selecting: the app's proxy list is not refreshed")
	}
	require.Equal(t, target, reportedSelected(t), "the app is told the new selection")
}
