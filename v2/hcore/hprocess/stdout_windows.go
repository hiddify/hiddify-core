package hprocess

import (
	"os"

	"golang.org/x/sys/windows"
)

// DetachStdio points stdout and stderr at NUL when they are pipes, so nothing is written to the
// app's pipes once the app may be gone (see the Unix version).
func DetachStdio() {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	for _, std := range []struct {
		id   uint32
		file **os.File
	}{{windows.STD_OUTPUT_HANDLE, &os.Stdout}, {windows.STD_ERROR_HANDLE, &os.Stderr}} {
		handle, err := windows.GetStdHandle(std.id)
		if err != nil {
			continue
		}
		if kind, err := windows.GetFileType(handle); err != nil || kind != windows.FILE_TYPE_PIPE {
			continue
		}
		_ = windows.SetStdHandle(std.id, windows.Handle(null.Fd()))
		*std.file = null
	}
}
