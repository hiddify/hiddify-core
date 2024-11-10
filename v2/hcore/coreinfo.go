package hcore

import (
	"fmt"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"google.golang.org/grpc"
)

func SetCoreStatus(state CoreStates, msgType MessageType, message string) *CoreInfoResponse {
	msg := fmt.Sprintf("%s: %s %s", state.String(), msgType.String(), message)
	if msgType == MessageType_EMPTY {
		msg = fmt.Sprintf("%s: %s", state.String(), message)
	}
	Log(LogLevel_INFO, LogType_CORE, msg)
	static.CoreState = state
	info := CoreInfoResponse{
		CoreState:   state,
		MessageType: msgType,
		Message:     message,
	}
	static.coreInfoObserver.Emit(&info)

	return &info
}

func (s *CoreService) CoreInfoListener(req *hcommon.Empty, stream grpc.ServerStreamingServer[CoreInfoResponse]) error {
	coreSub, done, err := static.coreInfoObserver.Subscribe()
	if err != nil {
		return err
	}
	defer static.coreInfoObserver.UnSubscribe(coreSub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-done:
			return nil
		case info := <-coreSub:
			stream.Send(info)
			// case <-time.After(500 * time.Millisecond):
			// 	info := SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "")
			// 	stream.Send(info)
		}
	}
}
