package config

import (
	"slices"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

const legacyDNSOutboundProfile = `{
  "outbounds": [
    {"type": "socks", "tag": "proxy-a", "server": "127.0.0.1", "server_port": 1080},
    {"type": "dns", "tag": "dns-out"}
  ],
  "route": {
    "rules": [
      {"protocol": "dns", "outbound": "dns-out"},
      {"domain_suffix": ".ir", "outbound": "proxy-a"}
    ]
  }
}`

func TestBuildConfigDropsLegacyDNSOutbound(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	opts := DefaultHiddifyOptions()

	options, err := BuildConfig(ctx, opts, &ReadOptions{Content: legacyDNSOutboundProfile})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	foundProxy := false
	for _, out := range options.Outbounds {
		if out.Tag == "proxy-a" {
			foundProxy = true
		}
		if out.Tag == "dns-out" {
			t.Fatalf("dns-out outbound should be removed, got type %q", out.Type)
		}
		if selector, ok := out.Options.(*option.SelectorOutboundOptions); ok && slices.Contains(selector.Outbounds, "dns-out") {
			t.Fatalf("dns-out must not be selectable in %q", out.Tag)
		}
	}
	if !foundProxy {
		t.Fatal("regular outbound proxy-a should be kept")
	}
	if _, err := options.MarshalJSONContext(ctx); err != nil {
		t.Fatalf("marshal built config: %v", err)
	}
}

func TestParseConfigDropsLegacyDNSOutbound(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	options, err := ParseConfig(ctx, &ReadOptions{Content: legacyDNSOutboundProfile}, false, nil, false)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	for _, out := range options.Outbounds {
		if isDNSOutbound(out) || out.Tag == "dns-out" {
			t.Fatalf("dns outbound %q should be dropped from the parsed profile", out.Tag)
		}
	}
	if len(options.Outbounds) != 1 || options.Outbounds[0].Tag != "proxy-a" {
		t.Fatalf("expected only proxy-a, got %d outbounds", len(options.Outbounds))
	}
}
