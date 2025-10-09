//go:build warp_disabled

package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strings"

	"github.com/bepass-org/warp-plus/warp"
	"github.com/hiddify/hiddify-core/v2/common"
	C "github.com/sagernet/sing-box/constant"

	// "github.com/bepass-org/wireguard-go/warp"
	"github.com/hiddify/hiddify-core/v2/db"

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
	Reserved      []int    `json:"reserved"`
	MTU           int      `json:"mtu"`
}

func wireGuardToSingbox(wgConfig WarpWireguardConfig, server string, port uint16) (*T.Outbound, error) {
    clientID, _ := base64.StdEncoding.DecodeString(wgConfig.ClientID)
    if len(clientID) < 2 {
        clientID = []byte{0, 0, 0}
    }
    opt := T.LegacyWireGuardOutboundOptions{
        ServerOptions: T.ServerOptions{
            Server:     server,
            ServerPort: port,
        },
        PrivateKey:    wgConfig.PrivateKey,
        PeerPublicKey: wgConfig.PeerPublicKey,
        Reserved:      []uint8{clientID[0], clientID[1], clientID[2]},
        MTU:           1330,
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
        opt.LocalAddress = append(opt.LocalAddress, prefix)
    }
    out := T.Outbound{Type: C.TypeWireGuard, Tag: "WARP", Options: opt}
    return &out, nil
}

func getRandomIP() string {
	ipPort, err := warp.RandomWarpEndpoint(true, true)
	if err == nil {
{{ ... }
	if err != nil {
		fmt.Printf("%v %v", singboxConfig, err)
		return nil, err
	}

	return singboxConfig, nil
}

func GenerateWarpInfo(license string, oldAccountId string, oldAccessToken string) (*warp.Identity, string, *WarpWireguardConfig, error) {
	if oldAccountId != "" && oldAccessToken != "" {
		err := warp.DeleteDevice(oldAccessToken, oldAccountId)
{{ ... }
				if err != nil {
					return err
				}
				wireguardConfig = *wgConfig
			}
			warpOutbound, err = GenerateWarpSingbox(wireguardConfig, host, port, fakePackets, fakePacketsSize, fakePacketsDelay, fakePacketsMode)
			if err != nil {
				fmt.Printf("Error generating warp config: %v", err)
				return err
			}
			if lw, ok := warpOutbound.Options.(T.LegacyWireGuardOutboundOptions); ok {
				lw.DialerOptions.Detour = detour
				base.Type = C.TypeWireGuard
				base.Options = lw
			} else {
				base.Type = C.TypeWireGuard
				base.Options = warpOutbound.Options
			}
		}
	}

	if final && base.Type == C.TypeWireGuard {
		lw, ok := base.Options.(T.LegacyWireGuardOutboundOptions)
		if !ok {
			return nil
		}
		host := lw.ServerOptions.Server

		if host == "default" || host == "random" || host == "auto" || host == "auto4" || host == "auto6" || isBlockedDomain(host) {
			// if base.WireGuardOptions.Detour != "" {
			//  base.WireGuardOptions.Server = "162.159.192.1"
			// } else {
			rndDomain := strings.ToLower(generateRandomString(20))
			staticIpsDns[rndDomain] = []string{}
			if host != "auto4" {
				if host == "auto6" || common.CanConnectIPv6() {
					randomIpPort, _ := warp.RandomWarpEndpoint(false, true)
					staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], randomIpPort.Addr().String())
				}
			}
			if host != "auto6" {
				randomIpPort, _ := warp.RandomWarpEndpoint(true, false)
				staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], randomIpPort.Addr().String())
			}
			lw.ServerOptions.Server = rndDomain
			// }
		}
		if lw.ServerOptions.ServerPort == 0 {
			port := warp.RandomWarpPort()
			lw.ServerOptions.ServerPort = port
		}

		if lw.DialerOptions.Detour != "" {
			if lw.MTU < 100 {
				lw.MTU = 1280
			}
		}
		base.Options = lw
		// if base.WireGuardOptions.Detour == "" {
		//  base.WireGuardOptions.GSO = runtime.GOOS != "windows"
		// }
	}

	return nil
}
