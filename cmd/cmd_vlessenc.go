package cmd

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var vlessEncJSON bool

var commandVlessEnc = &cobra.Command{
	Use:   "vlessenc",
	Short: "Generate VLESS encryption/decryption pair (X25519)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runVlessEnc(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	commandVlessEnc.Flags().BoolVar(&vlessEncJSON, "json", false, "output JSON")
	mainCommand.AddCommand(commandVlessEnc)
}

type vlessEncPair struct {
	Decryption string `json:"decryption"`
	Encryption string `json:"encryption"`
}

func runVlessEnc() error {
	privateKey, publicKey, err := genVlessEncCurve25519(nil)
	if err != nil {
		return err
	}
	serverKey := base64.RawURLEncoding.EncodeToString(privateKey)
	clientKey := base64.RawURLEncoding.EncodeToString(publicKey)
	pair := vlessEncPair{
		Decryption: generateVlessDotConfig("mlkem768x25519plus", "native", "600s", serverKey),
		Encryption: generateVlessDotConfig("mlkem768x25519plus", "native", "0rtt", clientKey),
	}
	if vlessEncJSON {
		return json.NewEncoder(os.Stdout).Encode(pair)
	}
	fmt.Printf("\"decryption\": \"%s\"\n\"encryption\": \"%s\"\n", pair.Decryption, pair.Encryption)
	return nil
}

func generateVlessDotConfig(fields ...string) string {
	return strings.Join(fields, ".")
}

func genVlessEncCurve25519(inputPrivateKey []byte) (privateKey []byte, publicKey []byte, err error) {
	if len(inputPrivateKey) > 0 {
		privateKey = inputPrivateKey
	}
	if privateKey == nil {
		privateKey = make([]byte, 32)
		if _, err = rand.Read(privateKey); err != nil {
			return nil, nil, err
		}
	}

	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	key, err := ecdh.X25519().NewPrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, key.PublicKey().Bytes(), nil
}
