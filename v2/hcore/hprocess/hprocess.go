// Package hprocess holds what the processes hiddify-core runs on its own share: the root VPN
// service (admin_bgservice) and the desktop app's core (`core`). Both stop when the app that
// started them is gone, and both must keep full speed while the app is in the background.
package hprocess

import (
	"os"
	"time"
)

// Alive reports whether the process with the id is still running.
func Alive(pid int32) bool {
	process, err := os.FindProcess(int(pid))
	if err != nil {
		return false
	}
	return processSignalAlive(process)
}

// WaitOwnerGone returns true once the process pid has been gone for two checks in a row, or false
// when done closes first (or there is no process to watch).
func WaitOwnerGone(pid int32, done <-chan struct{}) bool {
	if pid <= 0 {
		return false
	}
	failures := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return false
		case <-ticker.C:
		}
		if Alive(pid) {
			failures = 0
			continue
		}
		if failures++; failures >= 2 {
			return true
		}
	}
}
