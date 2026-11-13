package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/bepass-org/warp-plus/warp"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/hiddify/warpscan"
	"github.com/sagernet/sing/common/json/badoption"
	M "github.com/sagernet/sing/common/metadata"
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
		Options: &T.WARPEndpointOptions{
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
		if opts, ok := base.Options.(*option.WARPEndpointOptions); ok {
			opts.ServerOptions.Server = ""
			opts.ServerOptions.ServerPort = 0
			opts.Profile.Detour = dialDetour(OutboundWARPConfigDetour)
			return nil
		}
	}

	if final && base.Type == C.TypeWireGuard {
		if opts, ok := base.Options.(*option.WireGuardEndpointOptions); ok {
			if family, auto := wireGuardAutoFamily(*base, opts); auto {
				base.Type = C.TypeWARP
				base.Options = wireGuardToWarpEndpoint(opts, family)
			} else if opts.Detour != "" {
				opts.Noise = hiddify.NoiseOptions{} // the detour carries the packets, not the network
			}
		}
	}
	return nil
}

// wireGuardAutoFamily reports a WireGuard endpoint that leaves the WARP address to the app: its
// peer is auto, auto4, auto6, default or random (or there is no peer), or it is a WARP peer
// given by a domain or without a port. family is "4" for auto4, "6" for auto6, otherwise "". A
// server of the user's own keeps its address and port, a domain included (hiddify-app#2533).
func wireGuardAutoFamily(endpoint option.Endpoint, opts *option.WireGuardEndpointOptions) (family string, auto bool) {
	if len(opts.Peers) == 0 {
		return "", true
	}
	peer := opts.Peers[0]
	switch strings.ToLower(peer.Address) {
	case "", "auto", "default", "random":
		return "", true
	case "auto4":
		return "4", true
	case "auto6":
		return "6", true
	}
	if isWarp(endpoint) && (!M.ParseAddr(peer.Address).IsValid() || peer.Port == 0) {
		return "", true
	}
	return "", false
}

// wireGuardToWarpEndpoint makes a warp endpoint of a WireGuard endpoint to WARP whose address is
// left to the app, so that the WARP scanner finds an address and port that answer on the network,
// as for a warp:// link. Its key and tunnel addresses are kept as the endpoint's profile; without
// a key, the endpoint registers a device. family "4" or "6" limits the scanner, and the address used
// when it finds none, to IPv4 or IPv6 WARP addresses. A port of the config is the only one
// scanned. The MTU is found by MTU discovery, from the configured MTU but at least 1280.
func wireGuardToWarpEndpoint(opts *option.WireGuardEndpointOptions, family string) *option.WARPEndpointOptions {
	var port uint16
	if len(opts.Peers) > 0 {
		port = opts.Peers[0].Port
	}
	scanner := &option.WARPScannerOptions{Enabled: true, SearchNoise: true}
	if port != 0 {
		scanner.Ports = badoption.Listable[uint16]{port}
	}
	if family != "" {
		for _, prefix := range warpscan.WireGuardCIDRs {
			if prefix.Addr().Is4() == (family == "4") {
				scanner.CIDRs = append(scanner.CIDRs, prefix)
			}
		}
	}
	warpOpts := &option.WARPEndpointOptions{
		System:          opts.System,
		Name:            opts.Name,
		ListenPort:      opts.ListenPort,
		UDPTimeout:      opts.UDPTimeout,
		Workers:         opts.Workers,
		DialerOptions:   opts.DialerOptions,
		Noise:           opts.Noise,
		AWG:             opts.AWG,
		Scanner:         scanner,
		MTUDiscovery:    opts.MTUDiscovery,
		FallbackAddress: opts.FallbackAddress,
		WARPConfig:      warpInlineConfig(opts, family, port),
	}
	if opts.MTU != 0 {
		warpOpts.MTU = max(opts.MTU, minWarpMTU)
	}
	if opts.Detour != "" {
		warpOpts.Noise = hiddify.NoiseOptions{} // the detour carries the packets, not the network
	}
	return warpOpts
}

// minWarpMTU is the lowest MTU a WARP tunnel is configured with: IPv6's minimum. MTU discovery
// lowers it further only when the path needs it.
const minWarpMTU = 1280

// warpInlineConfig is the WARP profile of a WireGuard endpoint's key and tunnel addresses, nil
// without a key. Its peer is WARP's, on port, or 2408 when the config has none; the scanner
// replaces that address and port with ones that answer.
func warpInlineConfig(opts *option.WireGuardEndpointOptions, family string, port uint16) *C.WARPConfig {
	if opts.PrivateKey == "" {
		return nil
	}
	config := &C.WARPConfig{PrivateKey: opts.PrivateKey}
	for _, prefix := range opts.Address {
		address := prefix.Addr().String()
		if prefix.Addr().Is4() && config.Interface.Addresses.V4 == "" {
			config.Interface.Addresses.V4 = address
		} else if prefix.Addr().Is6() && config.Interface.Addresses.V6 == "" {
			config.Interface.Addresses.V6 = address
		}
	}
	if port == 0 {
		port = warpDefaultPort
	}
	publicKey := warpPublicKey
	if len(opts.Peers) > 0 && opts.Peers[0].PublicKey != "" {
		publicKey = opts.Peers[0].PublicKey
	}
	// the peers' element type is unnamed: it is made from JSON
	if err := json.Unmarshal([]byte(`{"peers":[{}]}`), config); err != nil {
		return nil
	}
	peer := &config.Peers[0]
	peer.PublicKey = publicKey
	peer.Endpoint.Host = net.JoinHostPort("engage.cloudflareclient.com", strconv.Itoa(int(port)))
	if family != "6" {
		peer.Endpoint.V4 = netip.AddrPortFrom(warpDefaultIPv4, port).String()
	}
	if family != "4" {
		peer.Endpoint.V6 = netip.AddrPortFrom(warpDefaultIPv6, port).String()
	}
	return config
}

// WARP's well-known endpoint, used when the scanner finds none or is off (behind a detour).
const warpDefaultPort = 2408

var (
	warpDefaultIPv4 = netip.MustParseAddr("162.159.192.1")
	warpDefaultIPv6 = netip.MustParseAddr("2606:4700:d0::a29f:c001")
)
