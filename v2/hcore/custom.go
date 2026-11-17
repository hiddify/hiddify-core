package hcore

import (
	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/log"
)

// errorWrapper stops the core for err and reports it once, as state. It reported it twice: first
// as UNEXPECTED_ERROR (StopAndAlert), then as state, and a listener took the first, so every
// failed start read as an unexpected error, whatever its kind.
func errorWrapper(state MessageType, err error) (*CoreInfoResponse, error) {
	Log(LogLevel_FATAL, LogType_CORE, err.Error())
	return StopAndAlert(state, err.Error()), err
}

// StopAndAlert stops the core and tells why. It changes the core's state and closes its service,
// so it runs only inside an operation (static.lock held): from anywhere else it would race a
// start that is about to store its service, which then runs with the state STOPPED. STOPPED is
// told once the service is closed: told before, the app could start again while it closed.
func StopAndAlert(msgType MessageType, message string) *CoreInfoResponse {
	stopConnectivityWatcher()
	if ss := static.startedService.Swap(nil); ss != nil {
		if err := closeService(ss); err == nil {
			engineClosed()
		}
	}
	return SetCoreStatus(CoreStates_STOPPED, msgType, message)
}

// Close stops the core and its gRPC server. The stop runs in the operation queue, which recovers
// a panic in it; a panic here comes after it, so it is only returned.
func Close(mode SetupMode) (err error) {
	defer config.DeferPanicToError("close", func(recovered error) {
		Log(LogLevel_FATAL, LogType_CORE, recovered.Error())
		err = recovered
	})
	log.Debug("[Service] Closing")

	_, err = Stop()
	// a system proxy this core set for a root core's engine (SetSystemProxyEnabled) goes with it
	if proxyErr := DisableUserSystemProxy(); proxyErr != nil {
		Log(LogLevel_WARNING, LogType_CORE, "restoring the system proxy: ", proxyErr.Error())
	}
	CloseGrpcServer(mode)

	return err
}

// func (s *CoreService) Status(ctx context.Context, empty *hcommon.Empty) (*CoreInfoResponse, error) {
// 	return Status()
// }
