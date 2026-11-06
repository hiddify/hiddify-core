package hprocess

import "golang.org/x/sys/unix"

// RaisePriority takes the process out of the background band. A process started by an app, or a
// launchd job without a ProcessType, can otherwise run with background priority and throttled I/O,
// which drains its sockets so slowly that macOS shrinks the TCP receive window (hiddify-app#2343).
func RaisePriority() {
	_ = unix.Setpriority(unix.PRIO_DARWIN_PROCESS, 0, 0)
}
