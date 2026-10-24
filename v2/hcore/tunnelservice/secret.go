package tunnelservice

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/v2/hcore/tunnel"
)

const controlSecretFileName = "control.secret"

// macOSTunnelServiceRoot is the root-owned install location of the unsigned VPN service.
const macOSTunnelServiceRoot = "/Library/Application Support/Hiddify/TunnelService"

func controlSecretPath() string {
	if dir := os.Getenv("HIDDIFY_TUNNEL_DATA"); dir != "" {
		return filepath.Join(dir, controlSecretFileName)
	}
	// The app (user) and the daemon (root copy) both read the secret next to the installed service.
	if _, err := os.Stat(macOSTunnelServiceRoot); err == nil {
		return filepath.Join(macOSTunnelServiceRoot, "data", controlSecretFileName)
	}
	return filepath.Join(serviceDataDirectory(), controlSecretFileName)
}

// EnsureControlSecret writes a random secret the installing user can read and the
// root daemon can enforce on every gRPC call. Safe to call again when the file exists.
func EnsureControlSecret(owner string) (string, error) {
	path := controlSecretPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if secret, err := os.ReadFile(path); err == nil && len(secret) >= tunnel.MinSecretLength {
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
		return "", err
	}
	if len(secret) < tunnel.MinSecretLength {
		return "", fmt.Errorf("tunnel control secret is too short")
	}
	return string(secret), nil
}

func restrictSecretFile(path, owner string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	if owner == "" {
		return nil
	}
	return chownSecret(path, owner)
}
