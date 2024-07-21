package config

import (
	"encoding/base64"
	"fmt"
	"net/netip"
	"os"

	"github.com/bepass-org/warp-plus/warp"
	C "github.com/sagernet/sing-box/constant"

	// "github.com/bepass-org/wireguard-go/warp"
	"log/slog"

	"github.com/sagernet/sing-box/option"
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
	clientID, _ := base64.StdEncoding.DecodeString(wgConfig.ClientID)
	if len(clientID) < 2 {
		clientID = []byte{0, 0, 0}
	}
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
			// Reserved: []uint8{0, 0, 0},
			MTU: 1330,
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

func generateWarp(license string, host string, port uint16, fakePackets string, fakePacketsSize string, fakePacketsDelay string, fakePacketsMode string) (*T.Outbound, error) {

	_, _, wgConfig, err := GenerateWarpInfo(license, "", "")
	if err != nil {
		return nil, err
	}
	if wgConfig == nil {
		return nil, fmt.Errorf("invalid warp config")
	}

	return GenerateWarpSingbox(*wgConfig, host, port, fakePackets, fakePacketsSize, fakePacketsDelay, fakePacketsMode)
}

func GenerateWarpSingbox(wgConfig WarpWireguardConfig, host string, port uint16, fakePackets string, fakePacketsSize string, fakePacketsDelay string, fakePacketMode string) (*T.Outbound, error) {
	if host == "" {
		host = "auto4"
	}

	if (host == "auto" || host == "auto4" || host == "auto6") && fakePackets == "" {
		fakePackets = "1-3"
	}
	if fakePackets != "" && fakePacketsSize == "" {
		fakePacketsSize = "10-30"
	}
	if fakePackets != "" && fakePacketsDelay == "" {
		fakePacketsDelay = "10-30"
	}
	singboxConfig, err := wireGuardToSingbox(wgConfig, host, port)
	if err != nil {
		fmt.Printf("%v %v", singboxConfig, err)
		return nil, err
	}

	singboxConfig.WireGuardOptions.FakePackets = fakePackets
	singboxConfig.WireGuardOptions.FakePacketsSize = fakePacketsSize
	singboxConfig.WireGuardOptions.FakePacketsDelay = fakePacketsDelay
	singboxConfig.WireGuardOptions.FakePacketsMode = fakePacketMode

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

func patchWarp(base *option.Outbound, configOpt *ConfigOptions, final bool, staticIpsDns map[string][]string) error {
	if base.Type == C.TypeCustom {
		if warp, ok := base.CustomOptions["warp"].(map[string]interface{}); ok {
			key, _ := warp["key"].(string)
			host, _ := warp["host"].(string)
			port, _ := warp["port"].(uint16)
			detour, _ := warp["detour"].(string)
			fakePackets, _ := warp["fake_packets"].(string)
			fakePacketsSize, _ := warp["fake_packets_size"].(string)
			fakePacketsDelay, _ := warp["fake_packets_delay"].(string)
			fakePacketsMode, _ := warp["fake_packets_mode"].(string)
			var warpConfig *T.Outbound
			var err error

			if configOpt == nil && (key == "p1" || key == "p2") {
				warpConfig = base
				return nil
			} else if key == "p1" {
				warpConfig, err = GenerateWarpSingbox(configOpt.Warp.WireguardConfig, host, port, fakePackets, fakePacketsSize, fakePacketsDelay, fakePacketsMode)
			} else if key == "p2" {
				warpConfig, err = GenerateWarpSingbox(configOpt.Warp2.WireguardConfig, host, port, fakePackets, fakePacketsSize, fakePacketsDelay, fakePacketsMode)
			} else {
				warpConfig, err = generateWarp(key, host, uint16(port), fakePackets, fakePacketsSize, fakePacketsDelay, fakePacketsMode)
			}
			if err != nil {
				fmt.Printf("Error generating warp config: %v", err)
				return err
			}
			warpConfig.WireGuardOptions.Detour = detour

			base.Type = C.TypeWireGuard

			base.WireGuardOptions = warpConfig.WireGuardOptions

		}

	}

	if final && base.Type == C.TypeWireGuard {
		host := base.WireGuardOptions.Server

		if host == "default" || host == "random" || host == "auto" || host == "auto4" || host == "auto6" || isBlockedDomain(host) {
			if base.WireGuardOptions.Detour != "" {
				base.WireGuardOptions.Server = "162.159.192.1"
			} else {
				rndDomain := generateRandomString(20)
				staticIpsDns[rndDomain] = []string{}
				if host != "auto4" {
					randomIpPort, _ := warp.RandomWarpEndpoint(false, true)
					staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], randomIpPort.Addr().String())
				}
				if host != "auto6" {
					randomIpPort, _ := warp.RandomWarpEndpoint(true, false)
					staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], randomIpPort.Addr().String())
				}
				base.WireGuardOptions.Server = rndDomain
			}

		}
		if base.WireGuardOptions.ServerPort == 0 {
			port := warp.RandomWarpPort()
			base.WireGuardOptions.ServerPort = port
		}

		if base.WireGuardOptions.Detour != "" {
			if base.WireGuardOptions.MTU > 1000 {
				base.WireGuardOptions.MTU -= 50
			}
			base.WireGuardOptions.FakePackets = ""
			base.WireGuardOptions.FakePacketsDelay = ""
			base.WireGuardOptions.FakePacketsSize = ""
		}
		// if base.WireGuardOptions.Detour == "" {
		// 	base.WireGuardOptions.GSO = runtime.GOOS != "windows"
		// }
	}

	return nil
}
