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

func Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}
