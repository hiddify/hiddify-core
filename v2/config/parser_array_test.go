package config

import (
	"context"
	"strings"
	"testing"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
)

// testContext wires the same registries the app builds at runtime, so the
// parser runs against real outbound constructors instead of an empty registry.
func testContext(t *testing.T) context.Context {
	t.Helper()
	return box.Context(
		context.Background(),
		include.InboundRegistry(),
		include.OutboundRegistry(),
		include.EndpointRegistry(),
		include.DNSTransportRegistry(),
		include.ServiceRegistry(),
		include.CertificateProviderRegistry(),
	)
}

func TestParseConfigContentAcceptsTopLevelOutboundArray(t *testing.T) {
	content := []byte(`[
  {"type": "socks", "tag": "proxy-1", "server": "127.0.0.1", "server_port": 1080},
  {"type": "socks", "tag": "proxy-2", "server": "127.0.0.2", "server_port": 1080}
]`)

	opts, err := parseConfigContent(testContext(t), content, false, DefaultHiddifyOptions(), false)
	if err != nil {
		t.Fatalf("top-level outbound array rejected: %v", err)
	}
	if len(opts.Outbounds) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(opts.Outbounds))
	}
}

func TestParseConfigContentAcceptsEmptyOutboundArray(t *testing.T) {
	_, err := parseConfigContent(testContext(t), []byte(`[]`), false, DefaultHiddifyOptions(), false)
	if err == nil {
		t.Fatal("expected an empty array to be reported as a config with no outbounds")
	}
	if !strings.Contains(err.Error(), "outbounds") {
		t.Fatalf("error should mention outbounds, got: %v", err)
	}
}

func TestParseConfigContentRejectsArrayOfNonObjects(t *testing.T) {
	_, err := parseConfigContent(testContext(t), []byte(`["vmess://a", "vmess://b"]`), false, DefaultHiddifyOptions(), false)
	if err == nil {
		t.Fatal("expected an array of strings to be rejected")
	}
	if !strings.Contains(err.Error(), "index 0") {
		t.Fatalf("error should point at the offending element, got: %v", err)
	}
}

func TestParseConfigContentErrorNamesUnexpectedType(t *testing.T) {
	for _, content := range []string{`42`, `"a string"`, `null`} {
		_, err := parseConfigContent(testContext(t), []byte(content), false, DefaultHiddifyOptions(), false)
		if err == nil {
			t.Fatalf("expected %q to be rejected", content)
		}
		if !strings.Contains(err.Error(), "expected a json object") {
			t.Fatalf("error for %q should say what was expected, got: %v", content, err)
		}
	}
}
