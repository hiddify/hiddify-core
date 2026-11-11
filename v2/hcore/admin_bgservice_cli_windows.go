//go:build windows

package hcore

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

// prepareCliCommand runs HiddifyCli.exe as the invoker for the admin_bgservice calls that need
// no administrator rights (start, connect: they only talk to the already elevated service).
//
// HiddifyCli.exe has an asInvoker manifest, but a "Run this program as an administrator"
// compatibility layer on its path (set by hand, or left by an older install) makes
// CreateProcess refuse it with ERROR_ELEVATION_REQUIRED (740), and VPN mode then never
// connects (hiddify-app#2501). RunAsInvoker overrides that layer. Only install, through
// HiddifyAdmin.exe, elevates.
func prepareCliCommand(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "__COMPAT_LAYER=RunAsInvoker")
}

// explainCliError turns ERROR_ELEVATION_REQUIRED into a message that says what to change.
func explainCliError(cliPath string, err error) error {
	if errors.Is(err, windows.ERROR_ELEVATION_REQUIRED) {
		return fmt.Errorf("Windows asks for administrator rights to run %s: turn off \"Run this program as an administrator\" in its Properties > Compatibility, then try again: %w", cliPath, err)
	}
	return err
}
