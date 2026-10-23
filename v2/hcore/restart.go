package hcore

import (
	"context"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
)

// Restart queues a restart of the core and answers at once with STOPPING; the outcome
// (STOPPING, STOPPED, STARTING, STARTED) is reported through CoreInfoListener.
func (s *CoreService) Restart(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	queue.Lock()
	defer queue.Unlock()
	queueRestartLocked(static.BaseContext, in)
	return &CoreInfoResponse{CoreState: CoreStates_STOPPING, MessageType: MessageType_EMPTY}, nil
}

// Restart stops then starts the core after the operations queued before it, and returns the
// outcome.
func Restart(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	queue.Lock()
	result := queueRestartLocked(ctx, in)
	queue.Unlock()
	return wait(result)
}

// queueRestartLocked queues a restart: it cancels the starts queued before it, and a stop
// called after it cancels its start; queue must be locked.
func queueRestartLocked(ctx context.Context, in *StartRequest) <-chan operationResult {
	signalStopLocked()
	ctx = withStopSignalLocked(ctx)
	return enqueueLocked(operationStart, "restart", func() (*CoreInfoResponse, error) {
		log.Debug("[Service] Restarting")
		if response, err := stopLocked(); err != nil {
			return response, err
		}
		if C.IsAndroid && static.HiddifyOptions.EnableTun {
			sleepContext(ctx, time.Second)
		}
		if err := contextErr(ctx); err != nil {
			return cancelledStart(err)
		}
		return startServiceLocked(ctx, in)
	})
}
