//go:build (!linux && !windows) || android

package hcore

import "fmt"

func elevateAdminBgInstall(cliPath, owner string) error {
	return fmt.Errorf("admin_bgservice install is not supported on this platform")
}
