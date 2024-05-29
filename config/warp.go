package config

import (
	"encoding/base64"
	"fmt"
	"net/netip"
	"os"

	"github.com/bepass-org/warp-plus/warp"

	// "github.com/bepass-org/wireguard-go/warp"
	"log/slog"

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

func wireGuardToSingbox(wgConfig WarpWireguardConfig, server string, port uint16) (*T.Outbound, error) {
	// splt := strings.Split(wgConfig.Peer.Endpoint, ":")
	// port, err := strconv.Atoi(splt[1])
	// if err != nil {
	// 	fmt.Printf("%v", err)
	// 	return nil
	// }
	clientID, _ := base64.StdEncoding.DecodeString(wgConfig.ClientID)

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
			Reserved:      []uint8{clientID[0], clientID[1], clientID[2]},
			MTU:           1330,
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

func getRandomIP() string {

	ipPort, err := warp.RandomWarpEndpoint(true, true)
	if err == nil {
		return ipPort.Addr().String()
	}
	return "engage.cloudflareclient.com"
}

func generateWarp(license string, host string, port uint16, fakePackets string, fakePacketsSize string, fakePacketsDelay string) (*T.Outbound, error) {

	_, _, wgConfig, err := GenerateWarpInfo(license, "", "")
	if err != nil {
		return nil, err
	}
	if wgConfig == nil {
		return nil, fmt.Errorf("invalid warp config")
	}
	fmt.Printf("%v", wgConfig)

	return GenerateWarpSingbox(*wgConfig, host, port, fakePackets, fakePacketsSize, fakePacketsDelay)
}

func GenerateWarpSingbox(wgConfig WarpWireguardConfig, host string, port uint16, fakePackets string, fakePacketsSize string, fakePacketsDelay string) (*T.Outbound, error) {
	if host == "" {
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

func GenerateWarpInfo(license string, oldAccountId string, oldAccessToken string) (*warp.Identity, string, *WarpWireguardConfig, error) {
	if oldAccountId != "" && oldAccessToken != "" {
		err := warp.DeleteDevice(oldAccessToken, oldAccountId)
		if err != nil {
			fmt.Printf("Error in removing old device: %v\n", err)
		} else {
			fmt.Printf("Old Device Removed")
		}
	}
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	identity, err := warp.CreateIdentityOnly(l, license)
	res := "Error!"
	var warpcfg WarpWireguardConfig
	if err == nil {
		res = "Success"
		res = fmt.Sprintf("Warp+ enabled: %t\n", identity.Account.WarpPlus)
		res += fmt.Sprintf("\nAccount type: %s\n", identity.Account.AccountType)
		warpcfg = WarpWireguardConfig{
			PrivateKey:       identity.PrivateKey,
			PeerPublicKey:    identity.Config.Peers[0].PublicKey,
			LocalAddressIPv4: identity.Config.Interface.Addresses.V4,
			LocalAddressIPv6: identity.Config.Interface.Addresses.V6,
			ClientID:         identity.Config.ClientID,
		}
	}

	return &identity, res, &warpcfg, err

}
