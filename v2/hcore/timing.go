package hcore

import (
	"fmt"
	"log"
	"os"
	"time"
)

// stopwatch logs how long each step of an operation (start, stop, ...) takes, to find what makes
// it slow. It writes to the core log at info level and also through the standard logger, so the
// steps show in the console (and in logcat on Android, where gomobile forwards it) even when the
// core log is only kept in a file.
type stopwatch struct {
	operation   string
	begin, last time.Time
}

func newStopwatch(operation string) *stopwatch {
	now := time.Now()
	return &stopwatch{operation: operation, begin: now, last: now}
}

// step reports the time since the previous step and since the beginning.
func (s *stopwatch) step(name string) {
	now := time.Now()
	s.report(fmt.Sprintf("%s: %s took %dms (total %dms)", s.operation, name, now.Sub(s.last).Milliseconds(), now.Sub(s.begin).Milliseconds()))
	s.last = now
}

// done reports the total.
func (s *stopwatch) done(outcome string) {
	s.report(fmt.Sprintf("%s: %s after %dms", s.operation, outcome, time.Since(s.begin).Milliseconds()))
}

func (s *stopwatch) report(message string) {
	reportTiming(message)
}

func reportTiming(message string) {
	log.Printf("[timing] %s", message)
	fmt.Fprintf(os.Stderr, "%s [timing] %s\n", time.Now().Format("15:04:05.000"), message)
	Log(LogLevel_INFO, LogType_CORE, "[timing] ", message)
}
