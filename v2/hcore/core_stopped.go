package hcore

import "sync/atomic"

// coreStoppedHandler is told when the core stops by itself, with the reason: StopAndAlert (a
// start, stop or reload that failed or panicked). The platform's service (Android's VPN service,
// the iOS packet tunnel) sets it: the core runs inside it, and only the app, through its status
// stream, saw such a stop, so with the app closed the service stayed up with no core behind it.
var coreStoppedHandler atomic.Pointer[func(MessageType, string)]

// SetCoreStoppedHandler sets the handler told when the core stops by itself; nil removes it. It
// is called on a goroutine of its own: the stop runs in an operation, which holds the core's lock
// that the handler's own stop of the core needs.
func SetCoreStoppedHandler(handler func(msgType MessageType, message string)) {
	if handler == nil {
		coreStoppedHandler.Store(nil)
		return
	}
	coreStoppedHandler.Store(&handler)
}

func tellCoreStopped(msgType MessageType, message string) {
	if handler := coreStoppedHandler.Load(); handler != nil {
		go (*handler)(msgType, message)
	}
}
