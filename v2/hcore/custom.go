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

func StopAndAlert(msgType MessageType, message string) {
	SetCoreStatus(CoreStates_STOPPED, msgType, message)

	if static.StartedService != nil {
		static.StartedService.CloseService()
		static.StartedService = nil
	}
}

func Close(mode SetupMode) error {
	defer config.DeferPanicToError("close", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	})
	log.Debug("[Service] Closing")

	_, err := Stop()
	CloseGrpcServer(mode)

	return err
}

// func (s *CoreService) Status(ctx context.Context, empty *hcommon.Empty) (*CoreInfoResponse, error) {
// 	return Status()
// }
