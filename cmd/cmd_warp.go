package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"

	"os"
	"strings"

	"github.com/hiddify/hiddify-core/config"
	T "github.com/sagernet/sing-box/option"
	"github.com/spf13/cobra"
)

var warpKey string

var commandWarp = &cobra.Command{
	Use:   "warp",
	Short: "warp configuration",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := generateWarp()
		fmt.Printf("out=%v Error! %v", out, err)
		if err != nil {
			fmt.Printf("Error! %v", err)
		}
	},
}

func init() {
	// commandWarp.Flags().StringVarP(&warpKey, "key", "k", "", "warp key")
	mainCommand.AddCommand(commandWarp)
}

type WireGuardConfig struct {
	Interface InterfaceConfig `json:"Interface"`
	Peer      PeerConfig      `json:"Peer"`
}

type InterfaceConfig struct {
	PrivateKey string   `json:"PrivateKey"`
	DNS        string   `json:"DNS"`
	Address    []string `json:"Address"`
}

type PeerConfig struct {
	PublicKey  string   `json:"PublicKey"`
	AllowedIPs []string `json:"AllowedIPs"`
	Endpoint   string   `json:"Endpoint"`
}

type SingboxConfig struct {
	Type          string   `json:"type"`
	Tag           string   `json:"tag"`
	Server        string   `json:"server"`
	ServerPort    int      `json:"server_port"`
	LocalAddress  []string `json:"local_address"`
	PrivateKey    string   `json:"private_key"`
	PeerPublicKey string   `json:"peer_public_key"`
	Reserved      []int    `json:"reserved"`
	MTU           int      `json:"mtu"`
}

func readWireGuardConfig(filePath string) (WireGuardConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return WireGuardConfig{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var wgConfig WireGuardConfig
	var currentSection string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		if currentSection == "Interface" {
			parseInterfaceConfig(&wgConfig.Interface, line)
		} else if currentSection == "Peer" {
			parsePeerConfig(&wgConfig.Peer, line)
		}
	}

	return wgConfig, nil
}

func parseInterfaceConfig(interfaceConfig *InterfaceConfig, line string) {
	if strings.HasPrefix(line, "PrivateKey") {
		interfaceConfig.PrivateKey = strings.TrimSpace(strings.SplitN(line, "=", 2)[1])
	} else if strings.HasPrefix(line, "DNS") {
		interfaceConfig.DNS = strings.TrimSpace(strings.SplitN(line, "=", 2)[1])
	} else if strings.HasPrefix(line, "Address") {
		interfaceConfig.Address = append(interfaceConfig.Address, strings.TrimSpace(strings.SplitN(line, "=", 2)[1]))
	}
}

func parsePeerConfig(peerConfig *PeerConfig, line string) {
	if strings.HasPrefix(line, "PublicKey") {
		peerConfig.PublicKey = strings.TrimSpace(strings.SplitN(line, "=", 2)[1])
	} else if strings.HasPrefix(line, "AllowedIPs") {
		peerConfig.AllowedIPs = append(peerConfig.AllowedIPs, strings.TrimSpace(strings.SplitN(line, "=", 2)[1]))
	} else if strings.HasPrefix(line, "Endpoint") {
		peerConfig.Endpoint = strings.TrimSpace(strings.SplitN(line, "=", 2)[1])
	}
}
func generateWarp() (*T.Outbound, error) {
	_, _, wg, err := config.GenerateWarpInfo("", "", "")

	// fmt.Printf("%v", wgConfig)
	singboxConfig, err := config.GenerateWarpSingbox(*wg, "", 0, "", "", "", "")
	singboxJSON, err := json.MarshalIndent(singboxConfig, "", "    ")
	if err != nil {
		fmt.Println("Error marshaling Singbox configuration:", err)
		return nil, err
	}
	fmt.Println(string(singboxJSON))
	return nil, nil
}
