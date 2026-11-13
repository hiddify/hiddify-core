//go:build windows

package hcore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// elevateAdminBgInstall shows one UAC prompt and waits for install to finish.
//
// It uses ShellExecuteExW with the "runas" verb directly. The previous
// implementation spawned `powershell -Command "Start-Process ... -Verb RunAs"`,
// which makes a GUI app launch PowerShell with an elevation one-liner: that is
// exactly the pattern antivirus heuristics and Sigma rules ("non-interactive
// PowerShell", "privilege escalation via Start-Process") are written for.
func elevateAdminBgInstall(cliPath, owner string) error {
	args := "admin_bgservice install " + quoteWindowsArg(owner)
	// HiddifyAdmin.exe (GUI subsystem, requireAdministrator manifest, app icon) is the binary
	// shown in the UAC dialog and the one the service is registered with; it forwards to the
	// same parseCli(). Every package ships it beside HiddifyCli.exe: no fallback to HiddifyCli.exe.
	exe := filepath.Join(filepath.Dir(cliPath), "HiddifyAdmin.exe")
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("HiddifyAdmin.exe could not be found in %s; reinstall Hiddify: %w", filepath.Dir(cliPath), err)
	}
	exitCode, err := shellExecuteRunAsWait(exe, args)
	if err != nil {
		return fmt.Errorf("could not install the VPN service; allow the administrator prompt and try again: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("the VPN service installer exited with code %d; allow the administrator prompt and try again", exitCode)
	}
	return nil
}

// quoteWindowsArg quotes one argument for a CreateProcess-style command line.
func quoteWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\"") {
		return arg
	}
	return `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
}

var (
	modshell32          = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW = modshell32.NewProc("ShellExecuteExW")
)

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoAsync        = 0x00000100
	seeMaskFlagNoUI       = 0x00000400
	seeMaskNoZoneChecks   = 0x00800000
)

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         windows.Handle
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     windows.Handle
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    windows.Handle
	dwHotKey     uint32
	hIcon        windows.Handle
	hProcess     windows.Handle
}

// shellExecuteRunAsWait launches exe elevated (UAC prompt), waits for it to exit and
// returns its exit code. ERROR_CANCELLED is returned when the user declines the prompt.
func shellExecuteRunAsWait(exe, params string) (uint32, error) {
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	parameters, _ := syscall.UTF16PtrFromString(params)
	// NoZoneChecks: HiddifyAdmin.exe from a downloaded portable zip or a network share carries the
	// mark of the web. ShellExecuteEx then wants to show the "Open File - Security Warning" dialog,
	// which FlagNoUI suppresses, and returns ERROR_CANCELLED as if the UAC prompt was refused
	// ("Administrator permission was not given"). The app beside it is already running, so the
	// check adds nothing; the UAC prompt itself still shows.
	info := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess | seeMaskNoAsync | seeMaskFlagNoUI | seeMaskNoZoneChecks,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: parameters,
		nShow:        windows.SW_HIDE,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))
	ok, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return 0, callErr
	}
	if info.hProcess == 0 {
		return 0, fmt.Errorf("elevated process handle not returned")
	}
	defer windows.CloseHandle(info.hProcess)
	if _, err := windows.WaitForSingleObject(info.hProcess, windows.INFINITE); err != nil {
		return 0, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &code); err != nil {
		return 0, err
	}
	return code, nil
}
