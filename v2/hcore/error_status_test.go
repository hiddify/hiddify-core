package hcore

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A failed start is told once, with its own kind: it was told twice, as UNEXPECTED_ERROR first,
// and the app took that one.
func TestFailedStartIsToldOnceWithItsKind(t *testing.T) {
	t.Cleanup(func() { SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "") })
	SetCoreStatus(CoreStates_STARTING, MessageType_EMPTY, "")
	events := static.coreInfoObserver.Subscribe(8)
	defer static.coreInfoObserver.Unsubscribe(events)

	errorWrapper(MessageType_ERROR_BUILDING_CONFIG, errors.New("bad config"))

	var told []*CoreInfoResponse
	for {
		select {
		case info := <-events:
			told = append(told, info)
			continue
		case <-time.After(200 * time.Millisecond):
		}
		break
	}
	require.Len(t, told, 1)
	require.Equal(t, CoreStates_STOPPED, told[0].CoreState)
	require.Equal(t, MessageType_ERROR_BUILDING_CONFIG, told[0].MessageType)
}
