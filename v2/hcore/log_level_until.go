package hcore

import (
	"sync"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/log"
)

// A log level the app turns on for a while (trace, from Report a problem) ends in the core too, at
// HiddifyOptions.LogLevelUntil: the app's own timer runs only while its UI does, and a VPN that
// outlived the UI (Android, iOS, a desktop core in its own process) stayed on trace until the app
// was opened again.
var logLevelEnd struct {
	sync.Mutex
	timer *time.Timer
}

func coreLogLevel(name string) LogLevel {
	switch name {
	case "trace":
		return LogLevel_TRACE
	case "debug":
		return LogLevel_DEBUG
	case "info":
		return LogLevel_INFO
	case "warn":
		return LogLevel_WARNING
	case "error":
		return LogLevel_ERROR
	case "fatal":
		return LogLevel_FATAL
	default:
		return LogLevel_INFO
	}
}

// applyLogLevel sets the level of the core's own logs and of the running box's, which keeps the
// level of its config otherwise until the next start.
func applyLogLevel(name string) {
	level := coreLogLevel(name)
	static.logLevel.Store(int32(level))
	static.debug = static.setupDebug || level <= LogLevel_DEBUG
	if box := static.Box(); box != nil {
		if boxLevel, err := log.ParseLevel(name); err == nil {
			box.LogFactory().SetLevel(boxLevel)
		}
	}
}

// scheduleLogLevelEnd ends the level of opt, the settings just set, at its LogLevelUntil, or now
// when that has passed (settings read back from the database at a start after it).
func scheduleLogLevelEnd(opt *config.HiddifyOptions) {
	logLevelEnd.Lock()
	defer logLevelEnd.Unlock()
	if logLevelEnd.timer != nil {
		logLevelEnd.timer.Stop()
		logLevelEnd.timer = nil
	}
	if opt.LogLevelUntil == "" || opt.LogLevelAfter == "" {
		return
	}
	until, err := time.Parse(time.RFC3339Nano, opt.LogLevelUntil)
	if err != nil {
		Log(LogLevel_WARNING, LogType_CORE, "log level end ", opt.LogLevelUntil, " unreadable, ending it now: ", err.Error())
		endLogLevelLocked(opt)
		return
	}
	wait := time.Until(until)
	if wait <= 0 {
		endLogLevelLocked(opt)
		return
	}
	Log(LogLevel_INFO, LogType_CORE, "log level ", opt.LogLevel, " until ", until.Local().Format(time.RFC3339), ", then ", opt.LogLevelAfter)
	logLevelEnd.timer = time.AfterFunc(wait, func() {
		logLevelEnd.Lock()
		defer logLevelEnd.Unlock()
		// settings set since then have their own end, or none
		if static.HiddifyOptions != opt {
			return
		}
		logLevelEnd.timer = nil
		endLogLevelLocked(opt)
	})
}

func endLogLevelLocked(opt *config.HiddifyOptions) {
	after := opt.LogLevelAfter
	Log(LogLevel_INFO, LogType_CORE, "log level ", opt.LogLevel, " ended, back to ", after)
	opt.LogLevel = after
	opt.LogLevelUntil, opt.LogLevelAfter = "", ""
	applyLogLevel(after)
}
