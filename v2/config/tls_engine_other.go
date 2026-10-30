//go:build !(darwin && cgo) && !windows

package config

// systemTLSEngine is none here (Android, Linux): uTLS is used.
func systemTLSEngine() string {
	return ""
}
