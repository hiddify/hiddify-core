//go:build !unix

package hprocess

import "os"

func processSignalAlive(process *os.Process) bool {
	return process != nil
}
