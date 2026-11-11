//go:build windows

package hutils

import (
	"os"
	"strings"
	"syscall"

	acl "github.com/hectane/go-acl"
	"github.com/sagernet/sing-box/experimental/libbox"

	"golang.org/x/sys/windows"
)

// RedirectStderr sends a Go crash (panic, fatal error) to path as well, as on the other systems.
// The desktop core's stderr goes to NUL once it is set up (hprocess.DetachStdio), so without it a
// crash left no trace and the app could only show the exit code (hiddify-app#2517).
func RedirectStderr(path string) error {
	return libbox.RedirectStderr(path)
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

// ExecuteCmd shows one UAC prompt and runs the already-installed HiddifyCli. It does not copy
// binaries into ProgramData or %TEMP% (both patterns raise antivirus false positives).
func ExecuteCmd(exe string, background bool, args ...string) (string, error) {
	verb := "runas"
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(strings.Join(args, " "))

	var showCmd int32 = 0 // SW_HIDE / SW_NORMAL: 0 keeps the UAC dialog only

	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		return "", err
	}
	return "", nil
}

func chmod(path string, mode os.FileMode) error {
	return acl.Chmod(path, mode)
}
