package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/netip"
	"os"
	"strings"

	"github.com/bepass-org/wireguard-go/warp"
	T "github.com/sagernet/sing-box/option"
)

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

func wireGuardToSingbox(wgConfig WireGuardConfig, server string, port uint16) (*T.Outbound, error) {
	// splt := strings.Split(wgConfig.Peer.Endpoint, ":")
	// port, err := strconv.Atoi(splt[1])
	// if err != nil {
	// 	fmt.Printf("%v", err)
	// 	return nil
	// }
	out := T.Outbound{
		Type: "wireguard",
		Tag:  "WARP",
		WireGuardOptions: T.WireGuardOutboundOptions{
			ServerOptions: T.ServerOptions{
				Server:     server,
				ServerPort: port,
			},

			PrivateKey:    wgConfig.Interface.PrivateKey,
			PeerPublicKey: wgConfig.Peer.PublicKey,
			Reserved:      []uint8{0, 0, 0},
			MTU:           1280,
		},
	}

	for _, addr := range wgConfig.Interface.Address {
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			return nil, err // Handle the error appropriately
		}
		out.WireGuardOptions.LocalAddress = append(out.WireGuardOptions.LocalAddress, prefix)

	}

	return &out, nil
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

var warpIPList = []string{
	"162.159.192.0/24",
	"162.159.193.0/24",
	"162.159.195.0/24",
	"162.159.204.0/24",
	"188.114.96.0/24",
	"188.114.97.0/24",
	"188.114.98.0/24",
	"188.114.99.0/24",
}
var warpPorts = []uint16{500, 854, 859, 864, 878, 880, 890, 891, 894, 903, 908, 928, 934, 939, 942,
	943, 945, 946, 955, 968, 987, 988, 1002, 1010, 1014, 1018, 1070, 1074, 1180, 1387, 1701,
	1843, 2371, 2408, 2506, 3138, 3476, 3581, 3854, 4177, 4198, 4233, 4500, 5279,
	5956, 7103, 7152, 7156, 7281, 7559, 8319, 8742, 8854, 8886}

func getRandomIP() string {
	randomRange := warpIPList[rand.Intn(len(warpIPList))]

	ip, err := warp.RandomIPFromRange(randomRange)
	if err == nil {
		return ip.String()
	}
	return "engage.cloudflareclient.com"
}

func generateRandomPort() uint16 {
	return warpPorts[rand.Intn(len(warpPorts))]
}

func generateWarp(license string, host string, port uint16, fakePackets string) (*T.Outbound, error) {
	if host == "" || isBlockedDomain(host) {
		host = "auto"
	}

	if host == "auto" && fakePackets == "" {
		fakePackets = "8-15"
	}
	// warp.UpdatePath("./secondary")
	if _, err := os.Stat("./wgcf-identity.json"); err == nil {
		os.Remove("./wgcf-identity.json")
	}

	if !warp.CheckProfileExists(license) {
		fmt.Printf("profile s not exit! ---%s---", license)
		err := warp.LoadOrCreateIdentity(license)
		if err != nil {
			return nil, err
		}
	}

	wgConfig, err := readWireGuardConfig("wgcf-profile.ini")
	if err != nil {
		fmt.Println("Error reading WireGuard configuration:", err)
		return nil, err
	}
	// fmt.Printf("%v", wgConfig)
	singboxConfig, err := wireGuardToSingbox(wgConfig, host, port)

	singboxConfig.WireGuardOptions.FakePackets = fakePackets
	singboxJSON, err := json.MarshalIndent(singboxConfig, "", "    ")
	if err != nil {
		fmt.Println("Error marshaling Singbox configuration:", err)
		return nil, err
	}

	fmt.Println(string(singboxJSON))
	return singboxConfig, nil
}
