//go:build windows

package hutils

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	acl "github.com/hectane/go-acl"

	"golang.org/x/sys/windows"
)

func RedirectStderr(path string) error {
	return fmt.Errorf("not supported on windows")
}

func IsAdmin() bool {
	adminSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return false
	}
	token := windows.Token(0)
	isMember, err := token.IsMember(adminSID)
	if err != nil {
		return false
	}
	return isMember
}

var TunAllowed = IsAdmin

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

func chmod(path string, mode os.FileMode) error {
	return acl.Chmod(path, mode)
}
