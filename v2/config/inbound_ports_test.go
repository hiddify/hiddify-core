package config

import (
	"fmt"
	"net/netip"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

// mixedInbounds builds the config of settings and returns its mixed inbounds by "address port", and
// each transparent port that opened by its type.
func mixedInbounds(t *testing.T, settings string) map[string]*option.HTTPMixedInboundOptions {
	t.Helper()
	options, err := BuildConfig(libbox.BaseContext(nil), chainTestOptions(t, settings), &ReadOptions{Content: oneProxyProfile})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	mixed := map[string]*option.HTTPMixedInboundOptions{}
	for _, inbound := range options.Inbounds {
		switch inbound.Type {
		case C.TypeMixed:
			opts := inbound.Options.(*option.HTTPMixedInboundOptions)
			mixed[fmt.Sprint(netip.Addr(*opts.Listen), " ", opts.ListenPort)] = opts
		case C.TypeRedirect, C.TypeTProxy, C.TypeDirect:
			mixed[inbound.Type] = nil
		}
	}
	return mixed
}

const ports = `"mixed-port":12334,"lan-sharing-port":12338,"redirect-port":12336,"direct-port":12337,"lan-sharing-password":"s3cret"`

func asksTheSecret(opts *option.HTTPMixedInboundOptions) bool {
	return len(opts.Users) == 1 && opts.Users[0].Username == ProxyUser && opts.Users[0].Password == "s3cret"
}

func TestVPNModeOpensNoPort(t *testing.T) {
	if got := mixedInbounds(t, `{"enable-tun":true,`+ports+`}`); len(got) != 0 {
		t.Fatalf("VPN mode opened %v", got)
	}
}

func TestVPNModeOpensTheLocalPortWhenAsked(t *testing.T) {
	got := mixedInbounds(t, `{"enable-tun":true,"local-proxy-in-vpn":true,`+ports+`}`)
	opts := got["127.0.0.1 12334"]
	if opts == nil || len(opts.Users) != 0 || len(got) != 2 {
		t.Fatalf("VPN mode, local port asked for: %v", got)
	}
}

func TestSharingOpensItsOwnPortWithTheSecret(t *testing.T) {
	got := mixedInbounds(t, `{"enable-tun":true,"allow-connection-from-lan":true,`+ports+`}`)
	opts := got[":: 12338"]
	if opts == nil || !asksTheSecret(opts) || opts.SetSystemProxy {
		t.Fatalf("shared VPN: %v", got)
	}
	if got["127.0.0.1 12334"] != nil {
		t.Fatal("a shared VPN opened the local port too")
	}
}

func TestProxyModeLocalPortHasNoPasswordByDefault(t *testing.T) {
	got := mixedInbounds(t, `{`+ports+`}`)
	opts := got["127.0.0.1 12334"]
	if opts == nil || len(opts.Users) != 0 || opts.SetSystemProxy {
		t.Fatalf("proxy mode local port = %+v", got)
	}
}

func TestProxyModeLocalPortAsksTheSecretWhenAsked(t *testing.T) {
	got := mixedInbounds(t, `{"local-proxy-auth":true,`+ports+`}`)
	if opts := got["127.0.0.1 12334"]; opts == nil || !asksTheSecret(opts) {
		t.Fatalf("proxy mode local port with a password = %+v", got)
	}
}

func TestSystemProxyNeverAsksThisDevice(t *testing.T) {
	got := mixedInbounds(t, `{"set-system-proxy":true,"local-proxy-auth":true,"allow-connection-from-lan":true,`+ports+`}`)
	if opts := got["127.0.0.1 12334"]; opts == nil || len(opts.Users) != 0 || !opts.SetSystemProxy {
		t.Fatalf("the system proxy's own listener = %+v", opts)
	}
	if opts := got[":: 12338"]; opts == nil || !asksTheSecret(opts) || opts.SetSystemProxy {
		t.Fatalf("the network's listener = %+v", opts)
	}
}

// The ::1 listener starts after the 127.0.0.1 one; were it the system proxy too, Windows would be
// left pointed at http://[::1]:port, which Telegram can not use.
func TestSystemProxyIsTheIPv4LoopbackOnly(t *testing.T) {
	got := mixedInbounds(t, `{"set-system-proxy":true,`+ports+`}`)
	if opts := got["127.0.0.1 12334"]; opts == nil || !opts.SetSystemProxy {
		t.Fatalf("the IPv4 loopback listener = %+v", opts)
	}
	if opts := got["::1 12334"]; opts == nil || opts.SetSystemProxy {
		t.Fatalf("the IPv6 loopback listener = %+v", opts)
	}
}

func TestSharingOnTheLocalPortTakesItOver(t *testing.T) {
	got := mixedInbounds(t, `{"allow-connection-from-lan":true,`+ports+`,"lan-sharing-port":12334}`)
	opts := got[":: 12334"]
	if opts == nil || !asksTheSecret(opts) || len(got) != 1 {
		t.Fatalf("sharing on the local port: %v", got)
	}
}

func TestPortSwitchesTurnPortsOff(t *testing.T) {
	got := mixedInbounds(t, `{"enable-mixed-port":false,"enable-redirect-port":false,"enable-direct-port":false,`+ports+`}`)
	if len(got) != 0 {
		t.Fatalf("switched off ports opened %v", got)
	}
}

func TestTransparentPortsAreOffUntilAskedFor(t *testing.T) {
	got := mixedInbounds(t, `{"set-system-proxy":true,`+ports+`}`)
	for addr, opts := range got {
		if opts == nil {
			t.Fatalf("an unset transparent port opened: %s", addr)
		}
	}
	if got["127.0.0.1 12334"] == nil {
		t.Fatal("the proxy port, on by default, did not open")
	}
}
