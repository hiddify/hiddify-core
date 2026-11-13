package config

import (
	"net/netip"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

const testPrivateKey = "aGlkZGlmeS1oaWRkaWZ5LWhpZGRpZnktaGlkZGlmeT0="

func testWireGuardEndpoint(address string, port uint16, publicKey string, mtu uint32) option.Endpoint {
	return option.Endpoint{Type: C.TypeWireGuard, Tag: "wg", Options: &option.WireGuardEndpointOptions{
		PrivateKey: testPrivateKey,
		MTU:        mtu,
		Address:    []netip.Prefix{netip.MustParsePrefix("172.16.0.2/32"), netip.MustParsePrefix("2606:4700:110:8a36::1/128")},
		Peers: []option.WireGuardPeer{{
			Address: address, Port: port, PublicKey: publicKey,
			AllowedIPs: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0")},
		}},
	}}
}

func patchTestEndpoint(t *testing.T, endpoint *option.Endpoint) {
	t.Helper()
	if err := patchWarp(endpoint, DefaultHiddifyOptions(), true, map[string][]string{}); err != nil {
		t.Fatal(err)
	}
}

// hiddify-app#2533: a WireGuard server given by a domain keeps its domain and port, even when the
// system resolver does not answer it; it used to become a random WARP address.
func TestPatchWarpKeepsOwnServer(t *testing.T) {
	for _, address := range []string{"wg-peer.hiddify.invalid", "t.me.hiddify.invalid", "198.51.100.1"} {
		endpoint := testWireGuardEndpoint(address, 1024, testPrivateKey, 0)
		patchTestEndpoint(t, &endpoint)
		opts, ok := endpoint.Options.(*option.WireGuardEndpointOptions)
		if !ok || endpoint.Type != C.TypeWireGuard {
			t.Fatalf("%s: became %s", address, endpoint.Type)
		}
		if opts.Peers[0].Address != address || opts.Peers[0].Port != 1024 {
			t.Fatalf("%s: peer changed to %s:%d", address, opts.Peers[0].Address, opts.Peers[0].Port)
		}
	}
}

// a WARP peer whose address is left to the app becomes a warp endpoint with the scanner on, its
// own key and addresses as the profile, no random port, and auto4/auto6 limited to their family
func TestPatchWarpAutoBecomesWarpScanner(t *testing.T) {
	for _, test := range []struct {
		address string
		port    uint16
		key     string
		family  string // "4", "6" or "" for both
	}{
		{"auto", 0, warpPublicKey, ""},
		{"auto4", 0, warpPublicKey, "4"},
		{"auto6", 0, warpPublicKey, "6"},
		{"random", 0, "", ""},
		{"engage.cloudflareclient.com", 2408, warpPublicKey, ""},
		{"162.159.192.7", 0, warpPublicKey, ""},
		{"auto4", 4500, warpPublicKey, "4"},
	} {
		t.Run(test.address, func(t *testing.T) {
			endpoint := testWireGuardEndpoint(test.address, test.port, test.key, 0)
			patchTestEndpoint(t, &endpoint)
			opts, ok := endpoint.Options.(*option.WARPEndpointOptions)
			if !ok || endpoint.Type != C.TypeWARP {
				t.Fatalf("not a warp endpoint: %s %T", endpoint.Type, endpoint.Options)
			}
			if opts.Scanner == nil || !opts.Scanner.Enabled || opts.Server != "" || opts.ServerPort != 0 {
				t.Fatalf("scanner off or server fixed: %+v %s:%d", opts.Scanner, opts.Server, opts.ServerPort)
			}
			if test.port != 0 && (len(opts.Scanner.Ports) != 1 || opts.Scanner.Ports[0] != test.port) {
				t.Fatalf("scanned ports %v, want only %d", opts.Scanner.Ports, test.port)
			}
			if test.port == 0 && len(opts.Scanner.Ports) != 0 {
				t.Fatalf("scanned ports %v, want WARP's", opts.Scanner.Ports)
			}
			for _, prefix := range opts.Scanner.CIDRs {
				if test.family == "" || prefix.Addr().Is4() != (test.family == "4") {
					t.Fatalf("scanner range %s for family %q", prefix, test.family)
				}
			}
			if test.family != "" && len(opts.Scanner.CIDRs) == 0 {
				t.Fatal("no scanner ranges for ", test.family)
			}
			config := opts.WARPConfig
			if config == nil || config.PrivateKey != testPrivateKey || config.Interface.Addresses.V4 != "172.16.0.2" ||
				config.Interface.Addresses.V6 != "2606:4700:110:8a36::1" || config.Peers[0].PublicKey != warpPublicKey {
				t.Fatalf("profile not kept: %+v", config)
			}
			peer := config.Peers[0].Endpoint
			if (peer.V4 == "") != (test.family == "6") || (peer.V6 == "") != (test.family == "4") {
				t.Fatalf("profile endpoints %q %q for family %q", peer.V4, peer.V6, test.family)
			}
		})
	}
}

// the MTU is found by discovery: a configured one is the ceiling, at least 1280; none stays the default
func TestPatchWarpAutoMTU(t *testing.T) {
	for _, test := range []struct{ mtu, want uint32 }{{0, 0}, {1000, minWarpMTU}, {1420, 1420}} {
		endpoint := testWireGuardEndpoint("auto", 0, warpPublicKey, test.mtu)
		patchTestEndpoint(t, &endpoint)
		opts := endpoint.Options.(*option.WARPEndpointOptions)
		if opts.MTU != test.want {
			t.Fatalf("mtu %d became %d, want %d", test.mtu, opts.MTU, test.want)
		}
		if opts.MTUDiscovery != nil && !*opts.MTUDiscovery {
			t.Fatal("MTU discovery turned off")
		}
	}
}

// sing-box accepts the converted endpoint, and an auto4 one behind a detour falls back to IPv4
func TestPatchWarpAutoConfigIsValid(t *testing.T) {
	endpoint := testWireGuardEndpoint("auto4", 0, warpPublicKey, 0)
	patchTestEndpoint(t, &endpoint)
	opts := endpoint.Options.(*option.WARPEndpointOptions)
	opts.Detour = "direct"
	warpBehindDetour(nil, []option.Endpoint{endpoint})
	if len(opts.FallbackAddress) != 1 || !opts.FallbackAddress[0].Is4() {
		t.Fatalf("fallback %v, want one IPv4 address", opts.FallbackAddress)
	}
	options := option.Options{
		Endpoints: []option.Endpoint{endpoint},
		Outbounds: []option.Outbound{{Type: C.TypeDirect, Tag: "direct", Options: &option.DirectOutboundOptions{}}},
	}
	if err := libbox.CheckConfigOptions(&options); err != nil {
		t.Fatal(err)
	}
}
