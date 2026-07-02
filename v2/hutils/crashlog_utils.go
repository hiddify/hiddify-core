//go:build darwin || linux || android

package hutils

import (
	"os"
	"runtime/debug"
)

// RedirectStderr writes Go crash reports into path so they survive with no
// terminal attached. Reimplemented locally after libbox.RedirectStderr became
// private upstream. Windows has its own no-op variant.
func RedirectStderr(path string) error {
	if stats, err := os.Stat(path); err == nil && stats.Size() > 0 {
		_ = os.Rename(path, path+".old")
	}
	outputFile, err := os.Create(path)
	if err != nil {
		return err
	}
	if err = debug.SetCrashOutput(outputFile, debug.CrashOptions{}); err != nil {
		outputFile.Close()
		os.Remove(outputFile.Name())
		return err
	}
	return nil
}
