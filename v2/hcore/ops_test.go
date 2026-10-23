package hcore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupOperationTest(t *testing.T) (int, <-chan *CoreInfoResponse) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	mixedPort := freeTCPPort(t)
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: hotReloadSettings(t, mixedPort, "warn")}, false)
	require.NoError(t, err)
	states := static.coreInfoObserver.Subscribe(64)
	t.Cleanup(func() {
		Stop()
		static.coreInfoObserver.Unsubscribe(states)
	})
	return mixedPort, states
}

// waitStates reads published states until want has been seen in order, and returns them all.
func waitStates(t *testing.T, states <-chan *CoreInfoResponse, want ...CoreStates) []CoreStates {
	t.Helper()
	var seen []CoreStates
	timeout := time.After(10 * time.Second)
	for len(want) > 0 {
		select {
		case state := <-states:
			seen = append(seen, state.CoreState)
			if state.CoreState == want[0] {
				want = want[1:]
			}
		case <-timeout:
			t.Fatalf("timeout waiting for %v, seen %v", want, seen)
		}
	}
	return seen
}

func startedService() bool {
	static.lock.Lock()
	defer static.lock.Unlock()
	return static.StartedService != nil
}

// gRPC Start and Stop answer at once and report the outcome through CoreInfoListener; a second
// Start while one is pending is ALREADY_STARTED.
func TestGRPCStartStopAnswerAtOnce(t *testing.T) {
	mixedPort, states := setupOperationTest(t)
	service := &CoreService{}
	request := &StartRequest{ConfigContent: profile(socks("proxy-a", 1080)), DelayStart: true}

	begin := time.Now()
	response, err := service.StartService(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, CoreStates_STARTING, response.CoreState)
	require.Less(t, time.Since(begin), 200*time.Millisecond)

	response, err = service.StartService(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, MessageType_ALREADY_STARTED, response.MessageType)

	waitStates(t, states, CoreStates_STARTING, CoreStates_STARTED)
	require.True(t, listening(mixedPort))

	begin = time.Now()
	response, err = service.Stop(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, CoreStates_STOPPING, response.CoreState)
	require.Less(t, time.Since(begin), 200*time.Millisecond)

	response, err = service.Stop(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, MessageType_ALREADY_STOPPED, response.MessageType)

	waitStates(t, states, CoreStates_STOPPING, CoreStates_STOPPED)
	require.False(t, startedService())
	require.False(t, listening(mixedPort))
}

// A Start right after a Stop waits for the stop to complete, then starts.
func TestGRPCStartAfterStopRunsInOrder(t *testing.T) {
	mixedPort, states := setupOperationTest(t)
	service := &CoreService{}
	request := &StartRequest{ConfigContent: profile(socks("proxy-a", 1080))}
	service.StartService(context.Background(), request)
	waitStates(t, states, CoreStates_STARTED)

	response, _ := service.Stop(context.Background(), nil)
	require.Equal(t, CoreStates_STOPPING, response.CoreState)
	response, _ = service.StartService(context.Background(), request)
	require.Equal(t, CoreStates_STARTING, response.CoreState, "start after a pending stop must be queued")

	seen := waitStates(t, states, CoreStates_STOPPING, CoreStates_STOPPED, CoreStates_STARTING, CoreStates_STARTED)
	require.Equal(t, []CoreStates{CoreStates_STOPPING, CoreStates_STOPPED, CoreStates_STARTING, CoreStates_STARTED}, seen)
	require.True(t, startedService())
	require.True(t, listening(mixedPort))
}

// A Stop during a start cancels the start instead of waiting for it to complete and undoing it.
func TestGRPCStopCancelsStartInProgress(t *testing.T) {
	mixedPort, states := setupOperationTest(t)
	service := &CoreService{}
	// DelayStart waits a second before the service is created
	service.StartService(context.Background(), &StartRequest{ConfigContent: profile(socks("proxy-a", 1080)), DelayStart: true})
	waitStates(t, states, CoreStates_STARTING)
	time.Sleep(100 * time.Millisecond)

	begin := time.Now()
	service.Stop(context.Background(), nil)
	seen := waitStates(t, states, CoreStates_STOPPED)
	require.Less(t, time.Since(begin), 900*time.Millisecond, "the stop waited for the start delay")
	require.NotContains(t, seen, CoreStates_STARTED)
	require.False(t, startedService())
	require.False(t, listening(mixedPort))
}

// The Go functions wait for the outcome; a start queued before Stop gives up.
func TestStopCancelsQueuedStart(t *testing.T) {
	setupOperationTest(t)
	static.lock.Lock() // hold the queue on an operation in progress
	started := make(chan error, 1)
	go func() {
		_, err := StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080))})
		started <- err
	}()
	stopped := make(chan error, 1)
	time.Sleep(100 * time.Millisecond) // the start is queued
	go func() {
		_, err := Stop()
		stopped <- err
	}()
	time.Sleep(100 * time.Millisecond) // the stop is queued
	static.lock.Unlock()

	require.ErrorIs(t, <-started, errStopped)
	require.NoError(t, <-stopped)
	require.Equal(t, CoreStates_STOPPED, currentCoreState())
	require.False(t, startedService())

	response, err := StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080))})
	require.NoError(t, err)
	require.Equal(t, CoreStates_STARTED, response.CoreState)
}
