package v2

import (
	"encoding/json"
	"fmt"

	"github.com/hiddify/hiddify-core/bridge"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"google.golang.org/grpc"
)

var (
	coreInfoObserver = *NewObserver[*pb.CoreInfoResponse](1)
	CoreState        = pb.CoreState_STOPPED
)

func SetCoreStatus(state pb.CoreState, msgType pb.MessageType, message string) *pb.CoreInfoResponse {
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
	coreInfoObserver.Emit(&info)
	if useFlutterBridge {
		msg, _ := json.Marshal(StatusMessage{Status: convert2OldState(CoreState)})
		bridge.SendStringToPort(statusPropagationPort, string(msg))
	}
	return &info
}

func (s *CoreService) CoreInfoListener(req *pb.Empty, stream grpc.ServerStreamingServer[pb.CoreInfoResponse]) error {
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
			// 	info := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_EMPTY, "")
			// 	stream.Send(info)
		}
	}
}
