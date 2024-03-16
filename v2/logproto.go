package v2

import (
	"fmt"
	"time"

	pb "github.com/hiddify/libcore/hiddifyrpc"
	"github.com/sagernet/sing/common/observable"
)

func NewObserver[T any](listenerBufferSize int) *observable.Observer[T] {
	return observable.NewObserver[T](&observable.Subscriber[T]{}, listenerBufferSize)
}

var logObserver = NewObserver[pb.LogMessage](10)

func Log(level pb.LogLevel, typ pb.LogType, message string) {
	fmt.Printf("%s %s %s\n", level, typ, message)
	logObserver.Emit(pb.LogMessage{
		Level:   level,
		Type:    typ,
		Message: message,
	})

}

func (s *CoreService) LogListener(stream pb.Core_LogListenerServer) error {
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
			return nil
		case info := <-logSub:
			stream.Send(&info)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
