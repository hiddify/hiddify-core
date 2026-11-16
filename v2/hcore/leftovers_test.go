package hcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/stretchr/testify/require"
)

func systemProxyConfig(port uint16) *option.Options {
	return &option.Options{Inbounds: []option.Inbound{{
		Type:    C.TypeMixed,
		Options: &option.HTTPMixedInboundOptions{ListenOptions: option.ListenOptions{ListenPort: port}, SetSystemProxy: true},
	}}}
}

// The marker names what the running engine changed: a refused hot reload keeps the running
// config's address (it was the new config's), and the marker goes only once the engine is closed,
// not when the state says STOPPED (StopAndAlert says it before it closes the engine).
func TestEngineMarkerFollowsWhatTheEngineRuns(t *testing.T) {
	previousPath := sWorkingPath
	t.Cleanup(func() {
		engineClosed()
		sWorkingPath = previousPath
	})
	sWorkingPath = t.TempDir()
	marker := filepath.Join(sWorkingPath, "data", "system-proxy.on")
	addresses := func() []string {
		content, err := os.ReadFile(marker)
		if os.IsNotExist(err) {
			return nil
		}
		require.NoError(t, err)
		return strings.Fields(string(content))
	}

	engineApplying(systemProxyConfig(2334))
	require.Equal(t, []string{"127.0.0.1:2334"}, addresses(), "marked before the engine applies it")
	engineApplied(true)

	engineApplying(systemProxyConfig(2335))
	require.ElementsMatch(t, []string{"127.0.0.1:2334", "127.0.0.1:2335"}, addresses(), "both while a reload applies")
	engineApplied(false)
	require.Equal(t, []string{"127.0.0.1:2334"}, addresses(), "a refused reload keeps the running config's")

	SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "")
	require.Equal(t, []string{"127.0.0.1:2334"}, addresses(), "STOPPED alone does not clear it")

	engineClosed()
	require.Empty(t, addresses())
}
