//go:build !windows && !(linux && !android) && !(darwin && !ios)

package hcore

// restoreSystemProxy: mobile platforms have no system proxy for a core to leave behind.
func restoreSystemProxy(string) error {
	return nil
}
