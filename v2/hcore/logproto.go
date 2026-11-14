package hcore

import (
	"context"
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
	if level < LogLevel(static.logLevel.Load()) {
		return
	}
	// if static.debug {
	msg := fmt.Sprintf("H %v %v", typ, fmt.Sprint(message...))
	logLevel(level, msg)
	// fmt.Printf("%v %v %v\n", level, typ, fmt.Sprint(message...))
	// os.Stderr.WriteString(fmt.Sprintf("%v %v %v\n", level, typ, fmt.Sprint(message...)))
	// }

	publishLog(level, typ, fmt.Sprint(message...))
}

// publishLog sends a message to log listeners only, without writing it to the
// sing-box logger. Used for messages that already come from sing-box, which
// would otherwise loop back through the platform writer.
func publishLog(level LogLevel, typ LogType, message string) {
	if level < LogLevel(static.logLevel.Load()) {
		return
	}
	static.logObserver.Publish(&LogMessage{
		Level:   level,
		Type:    typ,
		Time:    timestamppb.New(time.Now()),
		Message: message,
	})
}

// logListenerBuffer is how many log lines wait for a listener that is busy sending. With room
// for one, as it was, every burst lost all but its first lines: the app never saw them.
const logListenerBuffer = 1024

// logSendEvery is how long log lines wait to be sent to a listener together, and logSendCount how
// many of them wait at most. A line sent on its own is a write of its own, a TLS record and a
// wake-up of the app, which pays three isolate messages for each; with the debug or trace level
// the core logs hundreds of lines a second, and the app spent more on receiving them than on
// everything else. A warning or an error is not held back.
const (
	logSendEvery = 200 * time.Millisecond
	logSendCount = 256
)

func (s *CoreService) LogListener(req *LogRequest, stream grpc.ServerStreamingServer[LogMessage]) error {
	logSub := static.logObserver.Subscribe(logListenerBuffer)
	defer static.logObserver.Unsubscribe(logSub)
	return sendLogs(stream.Context(), logSub, req.GetLevel(), stream.Send)
}

// sendLogs hands the lines of logs at level or above to send, a few times a second (logSendEvery)
// rather than one by one, until ctx is done, logs is closed, or a send fails.
func sendLogs(ctx context.Context, logs <-chan *LogMessage, level LogLevel, send func(*LogMessage) error) error {
	var pending []*LogMessage
	// armed while lines wait; never running otherwise, so an idle listener wakes nothing
	timer := time.NewTimer(logSendEvery)
	timer.Stop()
	defer timer.Stop()
	flush := func() error {
		for i, info := range pending {
			pending[i] = nil
			if err := send(info); err != nil {
				return err
			}
		}
		pending = pending[:0]
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case info, ok := <-logs:
			if !ok {
				return flush()
			}
			if info == nil || info.Level < level {
				continue
			}
			pending = append(pending, info)
			if info.Level >= LogLevel_WARNING || len(pending) >= logSendCount {
				timer.Stop()
				if err := flush(); err != nil {
					return err
				}
			} else if len(pending) == 1 {
				timer.Reset(logSendEvery)
			}
		case <-timer.C:
			if err := flush(); err != nil {
				return err
			}
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
