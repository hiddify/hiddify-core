//go:build (!linux && !windows) || android

package hutils

import "os"

func TunAllowed() bool {
	return false
}

func ExecuteCmd(executablePath string, background bool, args ...string) (string, error) {
	return "", nil
}

func IsAdmin() bool {
	return os.Getuid() == 0
}
