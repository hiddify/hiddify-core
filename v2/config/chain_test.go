package config

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

const chainTestProfile = `{"outbounds":[
	{"type":"socks","tag":"proxy-a","server":"127.0.0.1","server_port":1080},
	{"type":"socks","tag":"proxy-b","server":"127.0.0.1","server_port":1081}
]}`

// settings in the shape the app sends (SingboxConfigOption.toJson)
func chainTestOptions(t *testing.T, settings string) *HiddifyOptions {
	t.Helper()
	opts := DefaultHiddifyOptions()
	if err := json.Unmarshal([]byte(settings), opts); err != nil {
		t.Fatalf("settings: %v", err)
	}
	return opts
}

func buildChainConfig(t *testing.T, settings string) *option.Options {
	t.Helper()
	options, err := BuildConfig(libbox.BaseContext(nil), chainTestOptions(t, settings), &ReadOptions{Content: chainTestProfile})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejected the config: %v", err)
	}
	return options
}

func outboundDetour(t *testing.T, options *option.Options, tag string) string {
	t.Helper()
	for _, out := range options.Outbounds {
		if out.Tag == tag {
			return out.Options.(option.DialerOptionsWrapper).TakeDialerOptions().Detour
		}
	}
	t.Fatalf("outbound %q not found", tag)
	return ""
}

func selectorItems(options *option.Options) []string {
	for _, out := range options.Outbounds {
		if out.Tag == OutboundSelectTag {
			return out.Options.(*option.SelectorOutboundOptions).Outbounds
		}
	}
	return nil
}

func TestChainExtraSecurityWarp(t *testing.T) {
	options := buildChainConfig(t, `{"chain-status":"extra_security",
		"extra-security":{"mode":"warp","warp":{"license-key":"LICENSE-1"},"psiphon":{"region":"AUTO","conduit-pairing-id":""},"profile":{"id":null}}}`)

	var hop *option.WARPEndpointOptions
	for _, ep := range options.Endpoints {
		if ep.Tag == ChainExtraSecurityTag {
			hop = ep.Options.(*option.WARPEndpointOptions)
		}
	}
	if hop == nil {
		t.Fatal("extra security WARP endpoint not created")
	}
	if hop.Detour != OutboundSelectTag || hop.Profile.Detour != OutboundSelectTag {
		t.Fatalf("extra security must dial through the main selector, got detour %q / profile detour %q", hop.Detour, hop.Profile.Detour)
	}
	if hop.Profile.License != "LICENSE-1" {
		t.Fatalf("license key not applied: %q", hop.Profile.License)
	}
	if options.Route.Final != ChainExtraSecurityTag {
		t.Fatalf("traffic must enter the extra security hop, route final = %q", options.Route.Final)
	}
	if outboundDetour(t, options, "proxy-a") != "" {
		t.Fatal("main outbounds must not dial through the extra security hop")
	}
	if slices.Contains(selectorItems(options), ChainExtraSecurityTag) {
		t.Fatal("the chain hop must not be selectable")
	}
}

func TestChainUnblockerPsiphon(t *testing.T) {
	options := buildChainConfig(t, `{"chain-status":"unblocker",
		"unblocker":{"mode":"psiphon","psiphon":{"region":"DE","conduit-pairing-id":""},"warp":{"license-key":"","clean-ip":"","clean-port":0,"noise":"","noise-size":"","noise-delay":"","noise-mode":""},"profile":{"id":null}}}`)

	var hop *option.PsiphonOutboundOptions
	for _, out := range options.Outbounds {
		if out.Tag == ChainUnblockerTag && out.Type == C.TypePsiphon {
			hop = out.Options.(*option.PsiphonOutboundOptions)
		}
	}
	if hop == nil {
		t.Fatal("unblocker psiphon outbound not created")
	}
	if hop.Config != "hiddify" {
		t.Fatalf("the psiphon hop must use the embedded hiddify config, got %q", hop.Config)
	}
	if hop.Detour != "" || hop.EgressRegion != "DE" {
		t.Fatalf("unblocker must dial directly with the chosen region, got detour %q region %q", hop.Detour, hop.EgressRegion)
	}
	for _, tag := range []string{"proxy-a", "proxy-b"} {
		if got := outboundDetour(t, options, tag); got != ChainUnblockerTag {
			t.Fatalf("%s must dial through the unblocker, detour = %q", tag, got)
		}
	}
	if options.Route.Final != OutboundSelectTag {
		t.Fatalf("traffic must still enter the main selector, route final = %q", options.Route.Final)
	}
	if slices.Contains(selectorItems(options), ChainUnblockerTag) {
		t.Fatal("the chain hop must not be selectable")
	}
}

func TestChainUnblockerWarpCleanIPAndNoise(t *testing.T) {
	options := buildChainConfig(t, `{"chain-status":"unblocker",
		"unblocker":{"mode":"warp","warp":{"license-key":"","clean-ip":"162.159.192.1","clean-port":2408,"noise":"5-10","noise-size":"10-30","noise-delay":"10-30","noise-mode":"m4"},"psiphon":{"region":"AUTO","conduit-pairing-id":""},"profile":{"id":null}}}`)

	var hop *option.WARPEndpointOptions
	for _, ep := range options.Endpoints {
		if ep.Tag == ChainUnblockerTag {
			hop = ep.Options.(*option.WARPEndpointOptions)
		}
	}
	if hop == nil {
		t.Fatal("unblocker WARP endpoint not created")
	}
	if hop.Detour != "" || hop.Server != "162.159.192.1" || hop.ServerPort != 2408 {
		t.Fatalf("unexpected unblocker WARP dialer/server: detour %q server %s:%d", hop.Detour, hop.Server, hop.ServerPort)
	}
	if !hop.Noise.FakePacket.Enabled || hop.Noise.FakePacket.Mode != "m4" {
		t.Fatalf("noise not applied: %+v", hop.Noise.FakePacket)
	}
	if got := outboundDetour(t, options, "proxy-a"); got != ChainUnblockerTag {
		t.Fatalf("proxy-a must dial through the unblocker, detour = %q", got)
	}
}

func TestChainOffAddsNothing(t *testing.T) {
	options := buildChainConfig(t, `{"chain-status":"off","extra-security":{"mode":"warp"},"unblocker":{"mode":"psiphon"}}`)
	for _, out := range options.Outbounds {
		if strings.Contains(out.Tag, "Extra Security") || strings.Contains(out.Tag, "Unblocker") {
			t.Fatalf("unexpected chain outbound %q", out.Tag)
		}
	}
	for _, ep := range options.Endpoints {
		if strings.Contains(ep.Tag, "Extra Security") || strings.Contains(ep.Tag, "Unblocker") {
			t.Fatalf("unexpected chain endpoint %q", ep.Tag)
		}
	}
	if options.Route.Final != OutboundSelectTag {
		t.Fatalf("route final = %q", options.Route.Final)
	}
}

func TestChainProfileModeIsRefused(t *testing.T) {
	opts := chainTestOptions(t, `{"chain-status":"extra_security","extra-security":{"mode":"profile","profile":{"id":"abc"}}}`)
	_, err := BuildConfig(libbox.BaseContext(nil), opts, &ReadOptions{Content: chainTestProfile})
	if err == nil || !strings.Contains(err.Error(), "profile mode is not supported") {
		t.Fatalf("expected profile mode to be refused, got %v", err)
	}
}
