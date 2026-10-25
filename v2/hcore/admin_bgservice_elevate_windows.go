//go:build windows

package hcore

import (
	"fmt"
	"os/exec"
	"strings"
)

// elevateAdminBgInstall shows one UAC prompt and waits for install to finish.
func elevateAdminBgInstall(cliPath, owner string) error {
	cli := strings.ReplaceAll(cliPath, "'", "''")
	ownerArg := strings.ReplaceAll(owner, "'", "''")
	ps := fmt.Sprintf(
		"Start-Process -FilePath '%s' -ArgumentList 'admin_bgservice','install','%s' -Verb RunAs -Wait",
		cli, ownerArg,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("could not install the VPN service; allow the administrator prompt and try again: %w", err)
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
