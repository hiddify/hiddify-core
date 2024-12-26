package hcore

import (
	"fmt"
	"os"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/observable"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewObserver[T any](listenerBufferSize int) *observable.Observer[T] {
	return observable.NewObserver(observable.NewSubscriber[T](listenerBufferSize), listenerBufferSize)
}

func Log(level LogLevel, typ LogType, message ...any) {
	if true || level != LogLevel_DEBUG {
		log.Debug(level, typ, fmt.Sprint(message...))
		fmt.Printf("%v %v %v\n", level, typ, fmt.Sprint(message...))
		os.Stderr.WriteString(fmt.Sprintf("%v %v %v\n", level, typ, fmt.Sprint(message...)))
	}

	static.logObserver.Emit(&LogMessage{
		Level:   level,
		Type:    typ,
		Time:    timestamppb.New(time.Now()),
		Message: fmt.Sprint(message...),
	})
}

func (s *CoreService) LogListener(req *hcommon.Empty, stream grpc.ServerStreamingServer[LogMessage]) error {
	logSub, stopch, _ := static.logObserver.Subscribe()
	defer static.logObserver.UnSubscribe(logSub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-stopch:
			return nil
		case info := <-logSub:
			stream.Send(info)
			// case <-time.After(500 * time.Millisecond):
		}
	}
}
