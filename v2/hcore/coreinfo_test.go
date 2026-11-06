package hcore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A start answers STARTING at once and fails a moment later, often before the app listens: told
// only "stopped", the app waited for the start for good. A new listener is told why it stopped.
func TestCoreInfoTellsWhyTheCoreStopped(t *testing.T) {
	t.Cleanup(func() {
		lastCoreInfo.Store(nil)
		SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "")
	})
	SetCoreStatus(CoreStates_STARTING, MessageType_EMPTY, "")
	StopAndAlert(MessageType_UNEXPECTED_ERROR, "start inbound/tun[tun-in]: unknown stack")

	info := currentCoreInfo()
	require.Equal(t, CoreStates_STOPPED, info.CoreState)
	require.Equal(t, MessageType_UNEXPECTED_ERROR, info.MessageType)
	require.Equal(t, "start inbound/tun[tun-in]: unknown stack", info.Message)

	// once the state moves on, the old reason is not told any more
	stateMu.Lock()
	setCoreState(CoreStates_STARTING)
	stateMu.Unlock()
	info = currentCoreInfo()
	require.Equal(t, CoreStates_STARTING, info.CoreState)
	require.Equal(t, MessageType_EMPTY, info.MessageType)
	require.Empty(t, info.Message)
}
