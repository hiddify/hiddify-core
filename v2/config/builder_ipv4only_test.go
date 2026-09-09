package config

import (
	"testing"

	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

func buildTunOptionsForMode(t *testing.T, mode option.DomainStrategy) *option.TunInboundOptions {
	t.Helper()

	options := &option.Options{}
	hiddifyOptions := DefaultHiddifyOptions()
	hiddifyOptions.EnableTun = true
	hiddifyOptions.IPv6Mode = mode

	setInbound(options, hiddifyOptions)

	if len(options.Inbounds) == 0 {
		t.Fatal("setInbound did not add a TUN inbound")
	}
	tunOptions, ok := options.Inbounds[0].Options.(*option.TunInboundOptions)
	if !ok {
		t.Fatalf("first inbound options type = %T, want *option.TunInboundOptions", options.Inbounds[0].Options)
	}
	return tunOptions
}

func countAddressFamilies(options *option.TunInboundOptions) (ipv4 int, ipv6 int) {
	for _, prefix := range options.Address {
		if prefix.Addr().Is4() {
			ipv4++
		}
		if prefix.Addr().Is6() {
			ipv6++
		}
	}
	return ipv4, ipv6
}

func TestSetInboundIPv4OnlyDoesNotAdvertiseIPv6(t *testing.T) {
	options := buildTunOptionsForMode(t, option.DomainStrategy(dns.DomainStrategyUseIPv4))
	ipv4, ipv6 := countAddressFamilies(options)

	if ipv4 != 1 {
		t.Fatalf("IPv4 address count = %d, want 1", ipv4)
	}
	if ipv6 != 0 {
		t.Fatalf("IPv6 address count = %d, want 0", ipv6)
	}
	if !options.AutoRoute {
		t.Fatal("AutoRoute = false, want true so the platform keeps the IPv6 leak guard route")
	}
}

func TestSetInboundUseIPv6PreservesDualStackAddresses(t *testing.T) {
	options := buildTunOptionsForMode(t, option.DomainStrategy(dns.DomainStrategyUseIPv6))
	ipv4, ipv6 := countAddressFamilies(options)

	if ipv4 != 1 {
		t.Fatalf("IPv4 address count = %d, want 1", ipv4)
	}
	if ipv6 != 1 {
		t.Fatalf("IPv6 address count = %d, want 1", ipv6)
	}
}
