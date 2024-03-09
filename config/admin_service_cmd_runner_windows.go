//go:build windows

package config

import (
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func ExecuteCmd(exe string, background bool, args ...string) (string, error) {
	verb := "runas"
	cwd, err := os.Getwd() // Error handling added
	if err != nil {
		return "", err
	}

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(strings.Join(args, " "))

	var showCmd int32 = 0 // SW_NORMAL

	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		return "", err
	}
	return "", nil
}
