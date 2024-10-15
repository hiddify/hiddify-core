package hcore

import (
	"encoding/json"
	"fmt"

	"github.com/hiddify/hiddify-core/bridge"
	common "github.com/hiddify/hiddify-core/v2/common"
	"google.golang.org/grpc"
)

var (
	coreInfoObserver = NewObserver[*CoreInfoResponse](1)
	CoreState        = CoreStates_STOPPED
)

func SetCoreStatus(state CoreStates, msgType MessageType, message string) *CoreInfoResponse {
	msg := fmt.Sprintf("%s: %s %s", state.String(), msgType.String(), message)
	if msgType == MessageType_EMPTY {
		msg = fmt.Sprintf("%s: %s", state.String(), message)
	}
	Log(LogLevel_INFO, LogType_CORE, msg)
	CoreState = state
	info := CoreInfoResponse{
		CoreState:   state,
		MessageType: msgType,
		Message:     message,
	}
	coreInfoObserver.Emit(&info)
	if useFlutterBridge {
		msg, _ := json.Marshal(StatusMessage{Status: convert2OldState(CoreState)})
		bridge.SendStringToPort(statusPropagationPort, string(msg))
	}
	return &info
}

func (s *CoreService) CoreInfoListener(req *common.Empty, stream grpc.ServerStreamingServer[CoreInfoResponse]) error {
	coreSub, done, err := coreInfoObserver.Subscribe()
	if err != nil {
		return err
	}
	defer coreInfoObserver.UnSubscribe(coreSub)

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
