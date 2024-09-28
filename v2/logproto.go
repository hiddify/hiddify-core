package v2

import (
	"fmt"
	"time"

	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/sagernet/sing/common/observable"
	"google.golang.org/grpc"
)

func NewObserver[T any](listenerBufferSize int) *observable.Observer[T] {
	return observable.NewObserver(observable.NewSubscriber[T](listenerBufferSize), listenerBufferSize)
}

var logObserver = NewObserver[pb.LogMessage](10)

func Log(level pb.LogLevel, typ pb.LogType, message string) {
	if level != pb.LogLevel_DEBUG {
		fmt.Printf("%s %s %s\n", level, typ, message)
	}
	logObserver.Emit(pb.LogMessage{
		Level:   level,
		Type:    typ,
		Message: message,
	})
}

func (s *CoreService) LogListener(req *pb.Empty, stream grpc.ServerStreamingServer[pb.LogMessage]) error {
	logSub, stopch, _ := logObserver.Subscribe()
	defer logObserver.UnSubscribe(logSub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-stopch:
			return nil
		case info := <-logSub:
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
