//go:build !windows

package hcore

// systemInterfaces has nothing to add outside Windows: there the monitor follows the routing
// table, which the default interface already shows.
func systemInterfaces() any {
	return nil
}
