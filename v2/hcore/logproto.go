package hcore

import (
	"fmt"
	"time"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/observable"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewObserver[T any](listenerBufferSize int) *observable.Observer[T] {
	return observable.NewObserver(observable.NewSubscriber[T](listenerBufferSize), listenerBufferSize)
}
func logLevel(level LogLevel, msg string) {
	switch level {
	case LogLevel_DEBUG:
		log.Debug(msg)
	case LogLevel_INFO:
		log.Info(msg)
	case LogLevel_WARNING:
		log.Warn(msg)
	case LogLevel_ERROR:
		log.Error(msg)
	default:
		log.Debug(msg)
	}
}
func Log(level LogLevel, typ LogType, message ...any) {
	if static.debug || level != LogLevel_DEBUG {
		msg := fmt.Sprintf("H %v %v", typ, fmt.Sprint(message...))
		logLevel(level, msg)
		// fmt.Printf("%v %v %v\n", level, typ, fmt.Sprint(message...))
		// os.Stderr.WriteString(fmt.Sprintf("%v %v %v\n", level, typ, fmt.Sprint(message...)))
	}

	static.logObserver.Emit(&LogMessage{
		Level:   level,
		Type:    typ,
		Time:    timestamppb.New(time.Now()),
		Message: fmt.Sprint(message...),
	})
}

func (s *CoreService) LogListener(req *LogRequest, stream grpc.ServerStreamingServer[LogMessage]) error {
	logSub, stopch, _ := static.logObserver.Subscribe()
	defer static.logObserver.UnSubscribe(logSub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-stopch:
			return nil
		case info := <-logSub:
			if info.Level < req.Level {
				continue
			}
			stream.Send(info)
			// case <-time.After(500 * time.Millisecond):
		}
	}
}
