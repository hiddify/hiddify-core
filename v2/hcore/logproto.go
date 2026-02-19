package hcore

import (
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/sagernet/sing-box/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func logLevel(level LogLevel, msg string) {
	switch level {
	case LogLevel_FATAL:
		log.Error(msg)
	case LogLevel_TRACE:
		log.Trace(msg)
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
	if level < static.logLevel {
		return
	}
	// if static.debug {
	msg := fmt.Sprintf("H %v %v", typ, fmt.Sprint(message...))
	logLevel(level, msg)
	// fmt.Printf("%v %v %v\n", level, typ, fmt.Sprint(message...))
	// os.Stderr.WriteString(fmt.Sprintf("%v %v %v\n", level, typ, fmt.Sprint(message...)))
	// }

	static.logObserver.Publish(&LogMessage{
		Level:   level,
		Type:    typ,
		Time:    timestamppb.New(time.Now()),
		Message: fmt.Sprint(message...),
	})
}

func (s *CoreService) LogListener(req *LogRequest, stream grpc.ServerStreamingServer[LogMessage]) error {
	logSub := static.logObserver.Subscribe(1)
	defer static.logObserver.Unsubscribe(logSub)

	for {
		select {
		case <-stream.Context().Done():
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

func dumpGoroutinesToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return pprof.Lookup("goroutine").WriteTo(f, 2)
}
