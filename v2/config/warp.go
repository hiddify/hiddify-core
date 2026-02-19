package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/netip"
	"os"

	"github.com/bepass-org/warp-plus/warp"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/wireguard-go/hiddify"

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
	PeerPublicKey string   `json:"peer_public_key"`
	Reserved      []int    `json:"reserved"`
	MTU           int      `json:"mtu"`
}

func wireGuardToSingbox(wgConfig WarpWireguardConfig, server string, port uint16) (*T.Endpoint, error) {
	clientID, _ := base64.StdEncoding.DecodeString(wgConfig.ClientID)
	if len(clientID) < 2 {
		clientID = []byte{0, 0, 0}
	}

	ips := []string{wgConfig.LocalAddressIPv4 + "/24", wgConfig.LocalAddressIPv6 + "/128"}
	localsaddrs := make([]netip.Prefix, 0)
	for _, addr := range ips {
		if addr == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			return nil, err // Handle the error appropriately
		}
		localsaddrs = append(localsaddrs, prefix)
	}
	out := T.Endpoint{
		Type: C.TypeWireGuard,
		Tag:  "WARP",
		Options: &T.WireGuardEndpointOptions{
			Peers: []T.WireGuardPeer{
				{
					AllowedIPs: []netip.Prefix{
						netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0"),
					},
					Address:   server,
					Port:      port,
					PublicKey: wgConfig.PeerPublicKey,
					Reserved:  []uint8{clientID[0], clientID[1], clientID[2]},
				},
			},
			Address:    localsaddrs,
			PrivateKey: wgConfig.PrivateKey,
			MTU:        1330,
		},
	}

	return &out, nil
}

func getRandomWarpIP() string {
	ipPort, err := warp.RandomWarpEndpoint(true, true)
	if err == nil {
		return ipPort.Addr().String()
	}
	return "engage.cloudflareclient.com"
}

func generateWarp(license string, host string, port uint16, noise *hiddify.NoiseOptions) (*T.Endpoint, error) {
	_, _, wgConfig, err := GenerateWarpInfo(license, "", "")
	if err != nil {
		return nil, err
	}
	if wgConfig == nil {
		return nil, fmt.Errorf("invalid warp config")
	}

	return GenerateWarpSingbox(*wgConfig, host, port, noise)
}

func GenerateWarpSingbox(wgConfig WarpWireguardConfig, host string, port uint16, noise *hiddify.NoiseOptions) (*T.Endpoint, error) {
	if host == "" {
		host = "auto4"
	}

	// if (host == "auto" || host == "auto4" || host == "auto6") && noise.FakePacket.Count.To == 0 {
	// 	noise.FakePackets = "1-3"
	// }
	// if noise.FakePackets != "" && noise.FakePacketsSize == "" {
	// 	noise.FakePacketsSize = "10-30"
	// }
	// if noise.FakePackets != "" && noise.FakePacketsDelay == "" {
	// 	noise.FakePacketsDelay = "10-30"
	// }
	singboxConfig, err := wireGuardToSingbox(wgConfig, host, port)
	if err != nil {
		fmt.Printf("%v %v", singboxConfig, err)
		return nil, err
	}
	if opts, ok := singboxConfig.Options.(*T.WireGuardEndpointOptions); ok {

		opts.Noise = *noise
	}
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

func getOrGenerateWarpLocallyIfNeeded(warpOptions *WarpOptions) WarpWireguardConfig {
	if warpOptions.WireguardConfig.PrivateKey != "" {
		return warpOptions.WireguardConfig
	}
	table := db.GetTable[WarpOptions]()
	dbWarpOptions, err := table.Get(warpOptions.Id)
	if err == nil && dbWarpOptions.WireguardConfig.PrivateKey != "" {
		return dbWarpOptions.WireguardConfig
	}
	license := ""
	if len(warpOptions.Id) == 26 { // warp key is 26 characters long
		license = warpOptions.Id
	} else if len(warpOptions.Id) > 28 && warpOptions.Id[2] == '_' { // warp key is 26 characters long
		license = warpOptions.Id[3:]
	}

	accountidentity, _, wireguardConfig, err := GenerateWarpInfo(license, warpOptions.Account.AccountID, warpOptions.Account.AccessToken)
	if err != nil {
		return WarpWireguardConfig{}
	}
	warpOptions.Account = WarpAccount{
		AccountID:   accountidentity.ID,
		AccessToken: accountidentity.Token,
	}
	warpOptions.WireguardConfig = *wireguardConfig
	table.UpdateInsert(warpOptions)

	return *wireguardConfig
}

func GenerateWarpSingboxNew(uniqueIdentifier string, noise *hiddify.NoiseOptions) (*T.Endpoint, error) {
	// if host=="auto4" || host=="auto6" || host=="auto"{
	// }
	// host=""
	// port=0

	out := T.Endpoint{
		Type: C.TypeWARP,
		Tag:  "WARP",
		Options: &T.WireGuardWARPEndpointOptions{
			// ServerOptions: T.ServerOptions{
			// 	Server:     server,
			// 	ServerPort: port,
			// },
			UniqueIdentifier: uniqueIdentifier,
			Profile: T.WARPProfile{
				Detour: OutboundSelectTag,
			},
			Noise: *noise,
			MTU:   1280,
		},
	}
	return &out, nil
}

func patchWarp(base *option.Endpoint, configOpt *HiddifyOptions, final bool, staticIpsDns map[string][]string) error {
	if base.Type == C.TypeWARP {
		if opts, ok := base.Options.(*option.WireGuardWARPEndpointOptions); ok {
			opts.ServerOptions.Server = ""
			opts.ServerOptions.ServerPort = 0
			opts.Profile.Detour = OutboundWARPConfigDetour
			return nil
			is_saved_key := len(opts.UniqueIdentifier) > 1 && opts.UniqueIdentifier[0] == 'p'

			if (configOpt == nil || !final) && is_saved_key {
				return nil
			}
			var wireguardConfig WarpWireguardConfig
			if is_saved_key {
				var warpOpt *WarpOptions
				if opts.UniqueIdentifier == "p1" {
					warpOpt = &configOpt.Warp
				} else if opts.UniqueIdentifier == "p2" {
					warpOpt = &configOpt.Warp2
				} else {
					warpOpt = &WarpOptions{
						Id: opts.UniqueIdentifier,
					}
				}
				warpOpt.Id = opts.UniqueIdentifier

				wireguardConfig = getOrGenerateWarpLocallyIfNeeded(warpOpt)
			} else {
				_, _, wgConfig, err := GenerateWarpInfo(opts.UniqueIdentifier, "", "")
				if err != nil {
					return err
				}
				wireguardConfig = *wgConfig
			}
			warpOutbound, err := GenerateWarpSingbox(wireguardConfig, opts.Server, opts.ServerPort, &opts.Noise)
			if err != nil {
				fmt.Printf("Error generating warp config: %v", err)
				return err
			}
			base.Type = C.TypeWireGuard
			if wopts, ok := warpOutbound.Options.(*option.WireGuardEndpointOptions); ok {
				wopts.Detour = opts.Detour
				base.Options = wopts
			}

		}
	}

	if final && base.Type == C.TypeWireGuard {
		if opts, ok := base.Options.(*option.WireGuardEndpointOptions); ok {
			host := "auto"
			if len(opts.Peers) == 0 {
				opts.Peers = append(opts.Peers, T.WireGuardPeer{Address: "auto"})
			}
			if opts.Peers[0].Address != "" {
				host = opts.Peers[0].Address
			}

			if host == "default" || host == "random" || host == "auto" || host == "auto4" || host == "auto6" || isBlockedDomain(host) {
				// if base.WireGuardOptions.Detour != "" {
				// 	base.WireGuardOptions.Server = "162.159.192.1"
				// } else {
				// rndDomain := strings.ToLower(generateRandomString(20))
				// staticIpsDns[rndDomain] = []string{}
				if host != "auto4" {
					if host == "auto6" { //|| common.u.CanConnectIPv6() {
						randomIpPort, _ := warp.RandomWarpEndpoint(false, true)
						// staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], randomIpPort.Addr().String())
						host = randomIpPort.Addr().String()
					}
				}
				if host != "auto6" {
					randomIpPort, _ := warp.RandomWarpEndpoint(true, false)
					// staticIpsDns[rndDomain] = append(staticIpsDns[rndDomain], randomIpPort.Addr().String())
					host = randomIpPort.Addr().String()
				}
				opts.Peers[0].Address = host
				// }
			}
			if opts.Peers[0].Port == 0 {
				port := warp.RandomWarpPort()
				opts.Peers[0].Port = port
			}

			if opts.Detour != "" {
				if opts.MTU < 100 {
					opts.MTU = 1280
				}
				opts.Noise = hiddify.NoiseOptions{}

			}
			// if base.WireGuardOptions.Detour == "" {
			// 	base.WireGuardOptions.GSO = runtime.GOOS != "windows"
			// }
		}
	}
	return nil
}
