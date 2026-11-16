package hcore

import (
	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/log"
)

func errorWrapper(state MessageType, err error) (*CoreInfoResponse, error) {
	Log(LogLevel_FATAL, LogType_CORE, err.Error())
	StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	return SetCoreStatus(CoreStates_STOPPED, state, err.Error()), err
}

// StopAndAlert stops the core and tells why. It changes the core's state and closes its service,
// so it runs only inside an operation (static.lock held): from anywhere else it would race a
// start that is about to store its service, which then runs with the state STOPPED.
func StopAndAlert(msgType MessageType, message string) {
	stopConnectivityWatcher()
	SetCoreStatus(CoreStates_STOPPED, msgType, message)

	if ss := static.StartedService; ss != nil {
		closeService(ss)
		static.StartedService = nil
	}
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
