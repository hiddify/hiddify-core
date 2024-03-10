package v2

import (
	"time"

	pb "github.com/hiddify/libcore/hiddifyrpc"
	"github.com/sagernet/sing/common/observable"
)

var logObserver = observable.Observer[pb.LogMessage]{}

func Log(level pb.LogLevel, typ pb.LogType, message string) {
	logObserver.Emit(pb.LogMessage{
		Level:   level,
		Type:    typ,
		Message: message,
	})

}

func (s *server) LogListener(stream pb.Hiddify_LogListenerServer) error {
	logSub, _, _ := logObserver.Subscribe()
	defer logObserver.UnSubscribe(logSub)

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
		case info := <-logSub:
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
