//go:build !linux

package hprocess

// SetName does nothing: on Windows and macOS the name and icon come from the launcher's file and
// bundle.
func SetName(name string, args []string) {}
