//go:build windows

package config

import (
	"os"
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

	// Convert args to UTF16Ptr slice
	var argsPtr []*uint16
	for _, arg := range args {
		argPtr, err := syscall.UTF16PtrFromString(arg)
		if err != nil {
			return "", err
		}
		argsPtr = append(argsPtr, argPtr)
	}

	var showCmd int32 = 1 // SW_NORMAL

	err = windows.ShellExecute(0, verbPtr, exePtr, nil, cwdPtr, showCmd)
	if err != nil {
		return "", err
	}
	return "", nil
}
