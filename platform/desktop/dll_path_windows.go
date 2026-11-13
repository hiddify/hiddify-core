//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// moduleAnchor is a variable of this DLL: its address names the module that holds this code.
var moduleAnchor byte

// init puts the folder of hiddify-core.dll on this process's PATH when libcronet.dll sits there.
//
// cronet-go loads libcronet.dll lazily and looks for it only beside the executable and on PATH.
// The core ran in core\Hiddify.exe, a subfolder of the app, while hiddify-core.dll and
// libcronet.dll stayed in the app folder, so Naive outbounds failed with "cronet: library not
// found. Place libcronet.dll in the executable directory or PATH". HiddifyCoreEngine.exe and the app
// sit beside the DLL now, which returns early below; a launcher elsewhere still needs it. The launcher's
// SetDllDirectoryW does not help: cronet-go checks file paths itself. Prepending to PATH keeps
// the load lazy (profiles without Naive never map the library).
func init() {
	dir := moduleDir()
	if dir == "" {
		return
	}
	if _, err := os.Stat(filepath.Join(dir, "libcronet.dll")); err != nil {
		return
	}
	if exe, err := os.Executable(); err == nil && strings.EqualFold(filepath.Dir(exe), dir) {
		return // found beside the executable already
	}
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// moduleDir returns the folder of the module (DLL or EXE) this code is linked into.
func moduleDir() string {
	var module windows.Handle
	flags := uint32(windows.GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | windows.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT)
	if err := windows.GetModuleHandleEx(flags, (*uint16)(unsafe.Pointer(&moduleAnchor)), &module); err != nil {
		return ""
	}
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n, err := windows.GetModuleFileName(module, &buf[0], uint32(len(buf)))
	if err != nil || n == 0 {
		return ""
	}
	return filepath.Dir(windows.UTF16ToString(buf[:n]))
}
