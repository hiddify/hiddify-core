package config

import (
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

const psiphonTestProfile = `{"outbounds":[
	{"type":"psiphon","tag":"psi-a"},
	{"type":"psiphon","tag":"psi-b"},
	{"type":"socks","tag":"proxy-a","server":"127.0.0.1","server_port":1080},
	{"type":"socks","tag":"via-psi-b","server":"127.0.0.1","server_port":1081,"detour":"psi-b"}
]}`

func buildWithProfile(t *testing.T, settings, profile string) *option.Options {
	t.Helper()
	options, err := BuildConfig(libbox.BaseContext(nil), chainTestOptions(t, settings), &ReadOptions{Content: profile})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejected the config: %v", err)
	}
	return options
}

func psiphonTags(options *option.Options) []string {
	var tags []string
	for _, out := range options.Outbounds {
		if out.Type == C.TypePsiphon {
			tags = append(tags, out.Tag)
		}
	}
	return tags
}

// without a Psiphon chain hop only the first Psiphon of the profile is kept, and a detour to a
// dropped one goes to the kept one
func TestOnlyOnePsiphonOutbound(t *testing.T) {
	options := buildWithProfile(t, `{"chain-status":"off"}`, psiphonTestProfile)
	tags := psiphonTags(options)
	if len(tags) != 1 || tags[0] != "psi-a § 0" && tags[0] != "psi-a" {
		t.Fatalf("expected only the first psiphon, got %v", tags)
	}
	if detour := outboundDetour(t, options, tagOf(options, "via-psi-b")); detour != tags[0] {
		t.Fatalf("detour to the dropped psiphon must go to the kept one, got %q", detour)
	}
	for _, tag := range selectorItems(options) {
		if tag == "psi-b" || tag == "psi-b § 1" {
			t.Fatal("the dropped psiphon is still selectable")
		}
	}
}

// a Psiphon chain hop replaces every Psiphon of the profile
func TestPsiphonChainRemovesProfilePsiphon(t *testing.T) {
	for _, c := range []struct{ name, settings, hop string }{
		{"unblocker", `{"chain-status":"unblocker","unblocker":{"mode":"psiphon","psiphon":{"region":"AUTO"}}}`, ChainUnblockerTag},
		{"extra security", `{"chain-status":"extra_security","extra-security":{"mode":"psiphon","psiphon":{"region":"AUTO"}}}`, ChainExtraSecurityTag},
	} {
		t.Run(c.name, func(t *testing.T) {
			options := buildWithProfile(t, c.settings, psiphonTestProfile)
			tags := psiphonTags(options)
			if len(tags) != 1 || tags[0] != c.hop {
				t.Fatalf("only the chain hop may be a psiphon, got %v", tags)
			}
			detour := outboundDetour(t, options, tagOf(options, "via-psi-b"))
			want := ""
			if c.hop == ChainUnblockerTag {
				want = ChainUnblockerTag // main outbounds dial through the unblocker
			}
			if detour != want {
				t.Fatalf("detour to a dropped psiphon: got %q, want %q", detour, want)
			}
		})
	}
}

// the legacy WARP setting no longer adds a WARP endpoint or routes through it
func TestLegacyWarpSettingIsIgnored(t *testing.T) {
	for _, mode := range []string{"proxy_over_warp", "warp_over_proxy"} {
		options := buildWithProfile(t, `{"chain-status":"off","warp":{"enable":true,"mode":"`+mode+`"}}`, chainTestProfile)
		for _, ep := range options.Endpoints {
			if ep.Tag == "🔒 WARP" || ep.Type == C.TypeWARP {
				t.Fatalf("%s: legacy WARP endpoint %q was added", mode, ep.Tag)
			}
		}
		if detour := outboundDetour(t, options, "proxy-a"); detour != "" {
			t.Fatalf("%s: outbounds must not dial through the legacy WARP, detour = %q", mode, detour)
		}
		if options.Route.Final != OutboundSelectTag {
			t.Fatalf("%s: route final = %q", mode, options.Route.Final)
		}
	}
}

// tagOf finds the built tag of a profile outbound (the builder may add an index suffix)
func tagOf(options *option.Options, name string) string {
	for _, out := range options.Outbounds {
		if out.Tag == name || len(out.Tag) > len(name) && out.Tag[:len(name)] == name {
			return out.Tag
		}
	}
	return name
}
