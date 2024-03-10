package v2

import (
	"time"

	pb "github.com/hiddify/libcore/hiddifyrpc"
	"github.com/sagernet/sing/common/observable"
)

var coreInfoObserver = observable.Observer[pb.CoreInfoResponse]{}
var CoreState = pb.CoreState_STOPPED

func SetCoreStatus(state pb.CoreState, msgType pb.MessageType, message string) pb.CoreInfoResponse {
	CoreState = state
	info := pb.CoreInfoResponse{
		CoreState:   state,
		MessageType: msgType,
		Message:     message,
	}
	coreInfoObserver.Emit(info)
	return info

}

func (s *server) CoreInfoListener(stream pb.Hiddify_CoreInfoListenerServer) error {
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
			break
		case <-stopch:
			break
		case info := <-coreSub:
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
