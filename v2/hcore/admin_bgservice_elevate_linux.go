//go:build linux && !android

package hcore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if owner != "" {
		cmd.Env = append(os.Environ(), "HIDDIFY_TUNNEL_OWNER="+owner)
	}
	err := cmd.Run()
	if err == nil {
		return nil
	}
	// pkexec exits 126 when the prompt was dismissed, and 127 both when authorization failed and when
	// the program could not run (a missing library, hiddify-app#2461): its stderr tells them apart.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 126 {
		return fmt.Errorf("could not install the VPN service; allow the administrator prompt and try again")
	}
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return fmt.Errorf("could not install the VPN service (%w): %s", err, detail)
	}
	return fmt.Errorf("could not install the VPN service (%w)", err)
}
