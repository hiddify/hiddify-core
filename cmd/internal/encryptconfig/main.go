// encryptconfig encrypts the Psiphon config for embedding at build time.
//
// It reads HIDDIFY_SECRET_KEY (base64 AES-256 key) and PSIPHON_CONFIG (the Psiphon config, raw JSON
// or base64) from the environment and prints the "henc:..." value for -ldflags -X. Used by the Makefile.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sagernet/sing-box/hiddify/secret"
)

func main() {
	encrypted, err := encrypt(os.Getenv("HIDDIFY_SECRET_KEY"), os.Getenv("PSIPHON_CONFIG"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "encryptconfig:", err)
		os.Exit(1)
	}
	fmt.Print(encrypted)
}

func encrypt(key, config string) (string, error) {
	plaintext := []byte(strings.TrimSpace(config))
	if !json.Valid(plaintext) {
		decoded, err := base64.StdEncoding.DecodeString(string(plaintext))
		if err != nil {
			return "", fmt.Errorf("PSIPHON_CONFIG is neither JSON nor base64: %w", err)
		}
		plaintext = decoded
	}
	if !json.Valid(plaintext) {
		return "", fmt.Errorf("PSIPHON_CONFIG is not a valid JSON config")
	}
	return secret.Encrypt(key, plaintext)
}
