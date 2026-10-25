//go:build linux && !android

package hcore

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// elevateAdminBgInstall uses pkexec (PolicyKit) and waits for install to finish.
func elevateAdminBgInstall(cliPath, owner string) error {
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		return fmt.Errorf("VPN mode in the AppImage needs root; quit and run: sudo %s", appimage)
	}
	cmd := exec.Command("pkexec", cliPath, "admin_bgservice", "install", owner)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if owner != "" {
		cmd.Env = append(os.Environ(), "HIDDIFY_TUNNEL_OWNER="+owner)
	}
	if err := cmd.Run(); err != nil {
		msg := err.Error()
		if strings.TrimSpace(msg) == "" {
			return fmt.Errorf("could not install the VPN service; allow the administrator prompt and try again")
		}
		return err
	}
	return nil
}
