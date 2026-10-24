//go:build !unix

package tunnelservice

import "os"

func processSignalAlive(process *os.Process) bool {
	return process != nil
}
