package config

import (
	"net/netip"

	"github.com/bepass-org/warp-plus/warp"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// A WireGuard (or AmneziaWG) endpoint whose detour is another WireGuard endpoint (WARP in WARP:
// the exit hop dials through the first hop) sends its UDP inside that tunnel. Each of its packets
// is its MTU plus 60 bytes of IPv4, UDP and WireGuard headers (80 over IPv6), and over IPv6 a packet
// that does not fit cannot be fragmented on the way: with the usual 1280 on both hops, the exit hop
// never works. So:
//   - the exit hop reaches a WARP peer given by an IPv6 address at an IPv4 WARP address instead, on
//     the same port (every WARP address serves every account), which needs 20 bytes less. This is
//     done when the first hop is WARP, and for the extra security hop, which dials through the
//     main profile. A WireGuard exit keeps the IPv6 address as its fallback, used when the IPv4
//     one gets no handshake;
//   - the first hop's MTU is raised to the most its own packets allow on a 1500-byte network, so
//     the exit hop keeps its MTU;
//   - if the exit hop's packets still do not fit, its MTU is lowered to fit.

const (
	wireGuardDefaultMTU   = 1408
	wireGuardIPv4Overhead = 60 // IPv4 20, UDP 8, WireGuard 32
	wireGuardIPv6Overhead = 80 // IPv6 40, UDP 8, WireGuard 32
	networkMTU            = 1500
)

// wireGuardOverhead is what WireGuard adds to each packet sent to address.
func wireGuardOverhead(address string) uint32 {
	if addr, err := netip.ParseAddr(address); err == nil && addr.Unmap().Is4() {
		return wireGuardIPv4Overhead
	}
	return wireGuardIPv6Overhead
}

// warpPublicKey is the public key of Cloudflare WARP's servers.
const warpPublicKey = "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo="

var warpIPv6Prefixes = []netip.Prefix{
	netip.MustParsePrefix("2606:4700:d0::/48"),
	netip.MustParsePrefix("2606:4700:d1::/48"),
}

func isWarpIPv6(addr netip.Addr) bool {
	for _, prefix := range warpIPv6Prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// isWarp reports an endpoint that connects to Cloudflare WARP.
func isWarp(endpoint option.Endpoint) bool {
	switch opts := endpoint.Options.(type) {
	case *option.WireGuardEndpointOptions:
		return len(opts.Peers) > 0 && opts.Peers[0].PublicKey == warpPublicKey
	case *option.AwgEndpointOptions:
		return len(opts.Peers) > 0 && opts.Peers[0].PublicKey == warpPublicKey
	}
	return endpoint.Type == C.TypeWARP
}

// tunnel is the parts of a WireGuard or AmneziaWG endpoint that the rules above read and change.
type tunnel struct {
	tag     string
	detour  string
	mtu     *uint32
	address []*string
	warp    bool
	// fallback is the WireGuard endpoint's fallback addresses (nil for AmneziaWG, which has none)
	fallback *badoption.Listable[netip.Addr]
}

func tunnelOf(endpoint option.Endpoint) (*tunnel, bool) {
	t := &tunnel{tag: endpoint.Tag, warp: isWarp(endpoint)}
	switch opts := endpoint.Options.(type) {
	case *option.WireGuardEndpointOptions:
		t.detour, t.mtu, t.fallback = opts.Detour, &opts.MTU, &opts.FallbackAddress
		for i := range opts.Peers {
			t.address = append(t.address, &opts.Peers[i].Address)
		}
	case *option.AwgEndpointOptions:
		t.detour, t.mtu = opts.Detour, &opts.MTU
		for i := range opts.Peers {
			t.address = append(t.address, &opts.Peers[i].Address)
		}
	default:
		return nil, false
	}
	if *t.mtu == 0 {
		*t.mtu = wireGuardDefaultMTU
	}
	return t, true
}

// overhead is the largest overhead of the tunnel's packets, by its peers' addresses.
func (t *tunnel) overhead() uint32 {
	overhead := uint32(wireGuardIPv4Overhead)
	for _, address := range t.address {
		overhead = max(overhead, wireGuardOverhead(*address))
	}
	return overhead
}

// warpBehindDetour sets up each WARP tunnel that dials through a detour (the extra security hop,
// WARP in WARP), where searching for a way in only measures the detour, and is slow:
//   - the scanner is off, for warp endpoints and MASQUE outbounds: its probes go through the detour;
//   - a warp endpoint tries two addresses and no other port: the profile's IPv4 address, then an
//     IPv6 WARP address (every WARP address serves every account) when that gets no handshake.
//
// Options a config sets itself are kept.
func warpBehindDetour(outbounds []option.Outbound, endpoints []option.Endpoint) {
	for _, outbound := range outbounds {
		if options, isMASQUE := outbound.Options.(*option.MASQUEOutboundOptions); isMASQUE && options.Detour != "" && options.Scanner != nil {
			options.Scanner.Enabled = false
		}
	}
	for _, endpoint := range endpoints {
		options, isWarp := endpoint.Options.(*option.WARPEndpointOptions)
		if !isWarp || endpoint.Type != C.TypeWARP || options.Detour == "" {
			continue
		}
		if options.Scanner != nil {
			options.Scanner.Enabled = false
			options.Scanner.SearchNoise = false
		}
		if options.RotatePorts == nil {
			rotatePorts := false
			options.RotatePorts = &rotatePorts
		}
		if len(options.FallbackAddress) == 0 {
			if ipv6, err := warp.RandomWarpEndpoint(false, true); err == nil {
				options.FallbackAddress = badoption.Listable[netip.Addr]{ipv6.Addr()}
			}
		}
	}
}

// fitWireGuardInWireGuard applies the rules above.
func fitWireGuardInWireGuard(endpoints []option.Endpoint) {
	tunnels := map[string]*tunnel{}
	var list []*tunnel
	for _, endpoint := range endpoints {
		if t, ok := tunnelOf(endpoint); ok {
			tunnels[t.tag] = t
			list = append(list, t)
		}
	}
	for _, exit := range list {
		if exit.detour == "" {
			continue
		}
		first := tunnels[exit.detour]
		if (first != nil && first.warp) || IsChainExtraSecurityTag(exit.tag) {
			for _, address := range exit.address {
				if addr, err := netip.ParseAddr(*address); err == nil && isWarpIPv6(addr.Unmap()) {
					if endpoint, err := warp.RandomWarpEndpoint(true, false); err == nil {
						*address = endpoint.Addr().String()
						if exit.fallback != nil && len(exit.address) == 1 {
							*exit.fallback = append(*exit.fallback, addr.Unmap())
						}
					}
				}
			}
		}
		if first == nil {
			continue
		}
		if first.detour == "" {
			*first.mtu = max(*first.mtu, networkMTU-first.overhead())
		}
		if need := *exit.mtu + exit.overhead(); need > *first.mtu {
			*exit.mtu = *first.mtu - exit.overhead()
		}
	}
}
