//go:build !darwin && !windows

package hprocess

// RaisePriority does nothing: Linux does not lower a background process on its own.
func RaisePriority() {}
