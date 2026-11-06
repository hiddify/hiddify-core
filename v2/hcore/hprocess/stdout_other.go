//go:build !unix && !windows

package hprocess

// DetachStdio does nothing on other systems.
func DetachStdio() {}
