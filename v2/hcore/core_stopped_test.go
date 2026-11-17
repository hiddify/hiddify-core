package hcore

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The platform's service is told when the core stops by itself, with the reason: only the app
// saw it, and with the app closed the service stayed up with no core.
func TestCoreStoppedHandlerIsTold(t *testing.T) {
	type stop struct {
		msgType MessageType
		message string
	}
	told := make(chan stop, 1)
	SetCoreStoppedHandler(func(msgType MessageType, message string) { told <- stop{msgType, message} })
	t.Cleanup(func() {
		SetCoreStoppedHandler(nil)
		SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "")
	})
	SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, "")

	errorWrapper(MessageType_START_SERVICE, errors.New("tun failed"))
	select {
	case got := <-told:
		require.Equal(t, stop{MessageType_START_SERVICE, "tun failed"}, got)
	case <-time.After(2 * time.Second):
		t.Fatal("the handler was not told")
	}
}
