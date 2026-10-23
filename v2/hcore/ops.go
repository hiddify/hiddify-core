package hcore

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
)

// Start, stop, restart and hot reload run one at a time, in the order they were called, on a
// background queue: a start never runs before an earlier stop has completed. The gRPC calls
// answer at once (STARTING, STOPPING) and the outcome is reported through CoreInfoListener; the
// Go functions (StartService, Stop, ...) wait for their turn and return the outcome.

type operationKind int

const (
	operationStart operationKind = iota // start, restart, hot reload: the core ends up started
	operationStop
)

type operationResult struct {
	response *CoreInfoResponse
	err      error
}

var queue struct {
	sync.Mutex
	last     <-chan struct{} // closed once the latest queued operation is done
	pending  int             // operations queued or running
	lastKind operationKind
	stop     chan struct{} // closed by a stop, for the operations queued before it
}

// coreState mirrors static.CoreState for reads outside static.lock.
var coreState atomic.Int32

func init() {
	done := make(chan struct{})
	close(done)
	queue.last = done
	queue.stop = make(chan struct{})
}

func currentCoreState() CoreStates {
	return CoreStates(coreState.Load())
}

// enqueueLocked runs run with static.lock held, after every operation queued before it; queue
// must be locked. The result channel receives the outcome.
func enqueueLocked(kind operationKind, name string, run func() (*CoreInfoResponse, error)) <-chan operationResult {
	previous := queue.last
	done := make(chan struct{})
	queue.last = done
	queue.pending++
	queue.lastKind = kind
	result := make(chan operationResult, 1)
	go func() {
		defer close(done)
		<-previous
		response, err := runOperation(name, run)
		queue.Lock()
		queue.pending--
		queue.Unlock()
		result <- operationResult{response, err}
	}()
	return result
}

func runOperation(name string, run func() (*CoreInfoResponse, error)) (response *CoreInfoResponse, err error) {
	static.lock.Lock()
	defer static.lock.Unlock()
	defer config.DeferPanicToError(name, func(recovered error) {
		response, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered)
	})
	return run()
}

func wait(result <-chan operationResult) (*CoreInfoResponse, error) {
	outcome := <-result
	return outcome.response, outcome.err
}

// signalStopLocked makes every operation queued before it give up at its next check, so a stop
// does not wait for a start to complete only to undo it; queue must be locked.
func signalStopLocked() {
	close(queue.stop)
	queue.stop = make(chan struct{})
}

// startingLocked reports whether the core is started or will be once the queue is done; queue
// must be locked.
func startingLocked() bool {
	if queue.pending > 0 {
		return queue.lastKind == operationStart
	}
	switch currentCoreState() {
	case CoreStates_STARTING, CoreStates_STARTED, CoreStates_HOT_RELOADING:
		return true
	}
	return false
}

type stopSignalKey struct{}

// withStopSignal makes ctx give up once the current stop signal is (see contextErr); queue must
// be locked.
func withStopSignalLocked(ctx context.Context) context.Context {
	return context.WithValue(ctx, stopSignalKey{}, (<-chan struct{})(queue.stop))
}

// serviceContext is ctx without the stop signal, for what outlives the operation: the core.
func serviceContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, stopSignalKey{}, nil)
}

var errStopped = errors.New("stop requested")

// contextErr reports why the operation running with ctx should give up; nil while it goes on.
func contextErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if stop, _ := ctx.Value(stopSignalKey{}).(<-chan struct{}); stop != nil {
		select {
		case <-stop:
			return errStopped
		default:
		}
	}
	return nil
}

// sleepContext waits for d, returning early once the operation running with ctx should give up.
func sleepContext(ctx context.Context, d time.Duration) {
	stop, _ := ctx.Value(stopSignalKey{}).(<-chan struct{})
	select {
	case <-ctx.Done():
	case <-stop:
	case <-time.After(d):
	}
}
