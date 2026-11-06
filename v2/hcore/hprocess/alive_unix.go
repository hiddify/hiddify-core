//go:build unix

package hprocess

import (
	"os"
	"syscall"
)

func processSignalAlive(process *os.Process) bool {
	err := process.Signal(syscall.Signal(0))
	return err == nil
}
