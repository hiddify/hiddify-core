package v2

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hiddify/libcore/bridge"
	pb "github.com/hiddify/libcore/hiddifyrpc"
)

var coreInfoObserver = NewObserver[pb.CoreInfoResponse](10)
var CoreState = pb.CoreState_STOPPED

func SetCoreStatus(state pb.CoreState, msgType pb.MessageType, message string) pb.CoreInfoResponse {
	msg := fmt.Sprintf("%s: %s %s", state.String(), msgType.String(), message)
	if msgType == pb.MessageType_EMPTY {
		msg = fmt.Sprintf("%s: %s", state.String(), message)
	}
	Log(pb.LogLevel_INFO, pb.LogType_CORE, msg)
	CoreState = state
	info := pb.CoreInfoResponse{
		CoreState:   state,
		MessageType: msgType,
		Message:     message,
	}
	coreInfoObserver.Emit(info)
	if useFlutterBridge {
		msg, _ := json.Marshal(StatusMessage{Status: convert2OldState(CoreState)})
		bridge.SendStringToPort(statusPropagationPort, string(msg))
	}
	return info

}

func (s *CoreService) CoreInfoListener(stream pb.Core_CoreInfoListenerServer) error {
	coreSub, _, _ := coreInfoObserver.Subscribe()
	defer coreInfoObserver.UnSubscribe(coreSub)
	stopch := make(chan int)
	go func() {
		stream.Recv()
		close(stopch)
	}()
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-stopch:
			return nil
		case info := <-coreSub:
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
