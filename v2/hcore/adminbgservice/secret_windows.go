//go:build windows

package adminbgservice

import (
	"fmt"
	"os/exec"
	"os/user"
)

// Restrict the secret to the installing user (read), SYSTEM and Administrators (full).
// icacls is the supported Windows way to set a small ACL without extra native code; antivirus
// tools treat this as normal installer behaviour.
func chownSecret(path, owner string) error {
	account := owner
	if _, err := user.Lookup(owner); err != nil {
		if current, lookupErr := user.Current(); lookupErr == nil {
			account = current.Username
		}
	}
	cmd := exec.Command("icacls", path, "/inheritance:r",
		"/grant:r", account+":(R)",
		"/grant:r", `NT AUTHORITY\SYSTEM:(F)`,
		"/grant:r", `BUILTIN\Administrators:(F)`,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restrict tunnel secret ACL for %s: %w (%s)", account, err, string(out))
	}
	return nil
}
