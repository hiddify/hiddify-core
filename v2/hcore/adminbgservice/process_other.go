//go:build !unix

package adminbgservice

import "os"

func processSignalAlive(process *os.Process) bool {
	return process != nil
}
