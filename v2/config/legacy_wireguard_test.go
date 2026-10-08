package config

import (
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

// a sing-box JSON subscription with the removed legacy WireGuard outbound gets a WireGuard endpoint
func TestLegacyWireGuardOutboundBecomesEndpoint(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	options, err := ParseConfig(ctx, &ReadOptions{Content: `{"outbounds": [
		{"type": "selector", "tag": "Select", "outbounds": ["wg", "proxy"]},
		{"type": "socks", "tag": "proxy", "server": "127.0.0.1", "server_port": 1080},
		{"type": "wireguard", "tag": "wg", "server": "198.51.100.1", "server_port": 51820,
		 "local_address": "10.0.0.2/32", "private_key": "aGlkZGlmeS1oaWRkaWZ5LWhpZGRpZnktaGlkZGlmeT0=",
		 "peer_public_key": "aGlkZGlmeS1oaWRkaWZ5LWhpZGRpZnktaGlkZGlmeT0=", "mtu": 1380}
	]}`}, false, DefaultHiddifyOptions(), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, outbound := range options.Outbounds {
		if outbound.Tag == "wg" {
			t.Fatalf("legacy WireGuard is still an outbound (%s)", outbound.Type)
		}
	}
	if len(options.Endpoints) != 1 || options.Endpoints[0].Type != C.TypeWireGuard {
		t.Fatalf("expected one WireGuard endpoint, got %+v", options.Endpoints)
	}
	endpoint := options.Endpoints[0].Options.(*option.WireGuardEndpointOptions)
	if endpoint.Address[0].String() != "10.0.0.2/32" || len(endpoint.Peers) != 1 ||
		endpoint.Peers[0].Address != "198.51.100.1" || endpoint.Peers[0].Port != 51820 {
		t.Fatalf("unexpected endpoint: %+v", endpoint)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejected the converted config: %v", err)
	}
}

// a JSON profile that is a single outbound object
func TestSingleOutboundJSONProfile(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	options, err := ParseConfig(ctx, &ReadOptions{Content: `{"type": "socks", "tag": "proxy", "server": "127.0.0.1", "server_port": 1080}`}, false, DefaultHiddifyOptions(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(options.Outbounds) != 1 || options.Outbounds[0].Tag != "proxy" {
		t.Fatalf("expected the socks outbound, got %+v", options.Outbounds)
	}
}
