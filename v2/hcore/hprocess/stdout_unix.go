//go:build unix

package hprocess

import (
	"os"

	"golang.org/x/sys/unix"
)

// DetachStdio points stdout and stderr at /dev/null when they are pipes. A process whose output is
// a pipe to the app must stop writing to it once the app may be gone: a write to a pipe nobody reads
// raises SIGPIPE, which ends a Go process at once, before it can stop cleanly (and restore the system
// proxy). Output already sent to a file is left alone.
func DetachStdio() {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer null.Close()
	for _, fd := range []int{int(os.Stdout.Fd()), int(os.Stderr.Fd())} {
		var stat unix.Stat_t
		if unix.Fstat(fd, &stat) == nil && stat.Mode&unix.S_IFMT == unix.S_IFIFO {
			_ = unix.Dup2(int(null.Fd()), fd)
		}
	}
}
