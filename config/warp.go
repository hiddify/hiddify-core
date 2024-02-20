package config

import (
	"fmt"
	"math/rand"
	"net/netip"

	"github.com/bepass-org/wireguard-go/warp"
	T "github.com/sagernet/sing-box/option"
)

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

func wireGuardToSingbox(wgConfig warp.WireguardConfig, server string, port uint16) (*T.Outbound, error) {
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

			PrivateKey:    wgConfig.PrivateKey,
			PeerPublicKey: wgConfig.PeerPublicKey,
			Reserved:      []uint8{0, 0, 0},
			MTU:           1280,
		},
	}

	ips := []string{wgConfig.LocalAddressIPv4 + "/24", wgConfig.LocalAddressIPv6 + "/128"}
	for _, addr := range ips {
		if addr == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			return nil, err // Handle the error appropriately
		}
		out.WireGuardOptions.LocalAddress = append(out.WireGuardOptions.LocalAddress, prefix)
	}
	return &out, nil
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

func generateWarp(license string, host string, port uint16, fakePackets string, fakePacketsSize string, fakePacketsDelay string) (*T.Outbound, error) {

	_, _, wgConfig, err := warp.LoadOrCreateIdentityHiddify(license, nil)
	if err != nil {
		return nil, err
	}
	if wgConfig == nil {
		return nil, fmt.Errorf("invalid warp config")
	}
	fmt.Printf("%v", wgConfig)

	return generateWarpSingbox(*wgConfig, host, port, fakePackets, fakePacketsSize, fakePacketsDelay)
}

func generateWarpSingbox(wgConfig warp.WireguardConfig, host string, port uint16, fakePackets string, fakePacketsSize string, fakePacketsDelay string) (*T.Outbound, error) {
	if host == "" || isBlockedDomain(host) {
		host = "auto"
	}

	if host == "auto" && fakePackets == "" {
		fakePackets = "8-15"
	}
	if fakePackets != "" && fakePacketsSize == "" {
		fakePacketsSize = "40-100"
	}
	if fakePackets != "" && fakePacketsDelay == "" {
		fakePacketsDelay = "20-250"
	}
	singboxConfig, err := wireGuardToSingbox(wgConfig, host, port)
	if err != nil {
		fmt.Printf("%v %v", singboxConfig, err)
		return nil, err
	}

	singboxConfig.WireGuardOptions.FakePackets = fakePackets
	singboxConfig.WireGuardOptions.FakePacketsSize = fakePacketsSize
	singboxConfig.WireGuardOptions.FakePacketsDelay = fakePacketsDelay

	return singboxConfig, nil
}

func GenerateWarpInfo(license string, oldAccountId string, oldAccessToken string) (*warp.AccountData, string, *warp.WireguardConfig, error) {
	if oldAccountId != "" && oldAccessToken != "" {
		accountData := warp.AccountData{
			AccountID:   oldAccountId,
			AccessToken: oldAccessToken,
		}
		err := warp.RemoveDevice(accountData)
		if err != nil {
			fmt.Printf("Error in removing old device: %v\n", err)
		} else {
			fmt.Printf("Old Device Removed")
		}
	}

	return warp.LoadOrCreateIdentityHiddify(license, nil)

}
