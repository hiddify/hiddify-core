package adminbgservice

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hiddify/hiddify-core/v2/hcore/adminbg"
)

const controlSecretFileName = "control.secret"

// controlDataDir is a fixed, OS-standard location shared by the elevated service and the
// unelevated app. It must not live next to a debug build path (those move) and must not be a
// copy of the app binary tree (that triggers antivirus heuristics on Windows).
func controlDataDir() string {
	if dir := os.Getenv("HIDDIFY_ADMIN_BG_DATA"); dir != "" {
		return dir
	}
	if dir := os.Getenv("HIDDIFY_TUNNEL_DATA"); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/Hiddify/AdminBgService/data"
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "Hiddify", "AdminBgService")
	default:
		return "/var/lib/hiddify-admin-bgservice"
	}
}

// legacyControlDataDir is the pre-rename path; used only to find an existing secret/stamp.
func legacyControlDataDir() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/Hiddify/TunnelService/data"
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "Hiddify", "TunnelService")
	default:
		return "/var/lib/hiddify-tunnel"
	}
}

func controlSecretPath() string {
	return filepath.Join(controlDataDir(), controlSecretFileName)
}

// EnsureControlSecret writes a random secret the installing user can read and the
// privileged service can enforce on every gRPC call. Safe to call again when the file exists.
func EnsureControlSecret(owner string) (string, error) {
	path := controlSecretPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if secret, err := os.ReadFile(path); err == nil && len(secret) >= adminbg.MinSecretLength {
		if err := restrictSecretFile(path, owner); err != nil {
			return "", err
		}
		return string(secret), nil
	}
	raw := make([]byte, 48)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	secret := base64.RawStdEncoding.EncodeToString(raw)
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		return "", err
	}
	if err := restrictSecretFile(path, owner); err != nil {
		return "", err
	}
	return secret, nil
}

func LoadControlSecret() (string, error) {
	secret, err := os.ReadFile(controlSecretPath())
	if err != nil {
		legacy := filepath.Join(legacyControlDataDir(), controlSecretFileName)
		secret, err = os.ReadFile(legacy)
		if err != nil {
			return "", err
		}
	}
	if len(secret) < adminbg.MinSecretLength {
		return "", fmt.Errorf("admin_bgservice control secret is too short")
	}
	return string(secret), nil
}

func restrictSecretFile(path, owner string) error {
	if err := os.Chmod(path, 0o600); err != nil && runtime.GOOS != "windows" {
		return err
	}
	if owner == "" {
		return nil
	}
	return chownSecret(path, owner)
}
