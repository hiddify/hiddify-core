package hcore

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"google.golang.org/grpc"
)

// stateMu orders every change of the core state with its publication, so listeners see the
// states in the order they were set. The ping watcher changes the state without static.lock,
// hence the separate lock.
var stateMu sync.Mutex

// lastCoreInfo is the last state published, with its message: why the core stopped, when it
// failed. A listener that comes after it is told it as well.
var lastCoreInfo atomic.Pointer[CoreInfoResponse]

func SetCoreStatus(state CoreStates, msgType MessageType, message string) *CoreInfoResponse {
	stateMu.Lock()
	defer stateMu.Unlock()
	return setCoreStatusLocked(state, msgType, message)
}

// setCoreStatusIf changes the state only while it is one of from and is not state already, and
// reports whether it did.
func setCoreStatusIf(from []CoreStates, state CoreStates, msgType MessageType, message string) (*CoreInfoResponse, bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	if !slices.Contains(from, currentCoreState()) {
		return nil, false
	}
	if currentCoreState() == state && msgType == MessageType_EMPTY {
		// nothing changed: listeners are told of changes only
		return nil, false
	}
	return setCoreStatusLocked(state, msgType, message), true
}

func setCoreStatusLocked(state CoreStates, msgType MessageType, message string) *CoreInfoResponse {
	msg := fmt.Sprintf("%s: %s %s", state.String(), msgType.String(), message)
	if msgType == MessageType_EMPTY {
		msg = fmt.Sprintf("%s: %s", state.String(), message)
	}
	Log(LogLevel_INFO, LogType_CORE, msg)
	setCoreState(state)
	if state == CoreStates_STOPPED {
		// the engine and its inbounds are closed: they undid what they changed in the host
		markEngineLeftovers(nil)
	}
	info := CoreInfoResponse{
		CoreState:   state,
		MessageType: msgType,
		Message:     message,
	}
	lastCoreInfo.Store(&info)
	static.coreInfoObserver.Publish(&info)

	return &info
}

// currentCoreInfo is the state a new listener is told first: the last one published, with why the
// core stopped, while the core is still in it. A start answers at once and fails a moment later,
// often before the app listens; told only "stopped", it waited for the start for good.
func currentCoreInfo() *CoreInfoResponse {
	state := currentCoreState()
	if last := lastCoreInfo.Load(); last != nil && last.CoreState == state {
		return last
	}
	return &CoreInfoResponse{CoreState: state, MessageType: MessageType_EMPTY}
}

// setCoreState changes the state without publishing it; stateMu must be held.
func setCoreState(state CoreStates) {
	static.CoreState = state
	coreState.Store(int32(state))
}

func (s *CoreService) CoreInfoListener(req *hcommon.Empty, stream grpc.ServerStreamingServer[CoreInfoResponse]) error {
	// Start and Stop answer at once and report through here: the buffer holds a whole burst of
	// states (STOPPING, STOPPED, STARTING, ...) so none is dropped before it is sent
	coreSub := static.coreInfoObserver.Subscribe(32)
	defer static.coreInfoObserver.Unsubscribe(coreSub)
	stream.Send(currentCoreInfo())
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case info := <-coreSub:
			stream.Send(info)
			// case <-time.After(500 * time.Millisecond):
			// 	// 	info := SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "")
			// 	stream.Send(&CoreInfoResponse{
			// 		CoreState:   static.CoreState,
			// 		MessageType: MessageType_EMPTY,
			// 		Message:     "",
			// 	})
		}
	}
}
