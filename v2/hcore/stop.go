package hcore

import (
	"context"
	"fmt"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
)

// stoppable are the states a stop moves to STOPPING.
var stoppable = append([]CoreStates{CoreStates_STARTING, CoreStates_HOT_RELOADING}, running...)

// Stop queues a stop of the core and answers at once with STOPPING (or ALREADY_STOPPED when the
// core is stopped or will be); the outcome is reported through CoreInfoListener.
func (s *CoreService) Stop(ctx context.Context, empty *hcommon.Empty) (*CoreInfoResponse, error) {
	queue.Lock()
	defer queue.Unlock()
	if !startingLocked() {
		return &CoreInfoResponse{CoreState: currentCoreState(), MessageType: MessageType_ALREADY_STOPPED}, nil
	}
	queueStopLocked()
	// disconnecting shows at once, not when the queue reaches the stop
	setCoreStatusIf(stoppable, CoreStates_STOPPING, MessageType_EMPTY, "")
	return &CoreInfoResponse{CoreState: CoreStates_STOPPING, MessageType: MessageType_EMPTY}, nil
}

// Stop stops the core after the operations queued before it, and returns the outcome. A start
// queued before it gives up instead of completing to be stopped.
func Stop() (*CoreInfoResponse, error) {
	queue.Lock()
	result := queueStopLocked()
	// disconnecting shows at once, not when the queue reaches the stop
	setCoreStatusIf(stoppable, CoreStates_STOPPING, MessageType_EMPTY, "")
	queue.Unlock()
	return wait(result)
}

// queueStopLocked queues a stop; queue must be locked.
func queueStopLocked() <-chan operationResult {
	signalStopLocked()
	return enqueueLocked(operationStop, "stop", stopLocked)
}

// stopLocked stops the core; static.lock must be held.
func stopLocked() (*CoreInfoResponse, error) {
	watch := newStopwatch("stop")
	stopConnectivityWatcher()
	watch.step("ending the connectivity watcher")
	// already reported when the stop was called, unless a restart runs it
	setCoreStatusIf(stoppable, CoreStates_STOPPING, MessageType_EMPTY, "")
	ss := static.StartedService
	if ss == nil {
		return SetCoreStatus(CoreStates_STOPPED, MessageType_ALREADY_STOPPED, ""), nil
	}

	err := closeService(ss)
	watch.step("closing the service")
	if err != nil {
		static.StartedService = nil
		dumpGoroutinesToFile(fmt.Sprint(sWorkingPath, "/data/goroutine-stop.log"))
		return errorWrapper(MessageType_UNEXPECTED_ERROR, err)
	}
	static.StartedService = nil
	watch.done("core is stopped")

	return SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, ""), nil
}
