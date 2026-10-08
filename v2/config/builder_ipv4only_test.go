package config

import (
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
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
	options := buildTunOptionsForMode(t, option.DomainStrategy(C.DomainStrategyIPv4Only))
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

func TestShouldEnableIPv6(t *testing.T) {
	tests := []struct {
		name             string
		mode             option.DomainStrategy
		hostSupportsIPv6 bool
		want             bool
	}{
		{"IPv4Only disables IPv6 on a capable host", option.DomainStrategy(C.DomainStrategyIPv4Only), true, false},
		{"IPv4Only stays disabled on an IPv4-only host", option.DomainStrategy(C.DomainStrategyIPv4Only), false, false},
		{"AsIs preserves IPv6 on a capable host", option.DomainStrategy(C.DomainStrategyAsIS), true, true},
		{"UseIPv6 preserves existing host capability check", option.DomainStrategy(C.DomainStrategyIPv6Only), false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ipv6EnabledFor(test.mode, test.hostSupportsIPv6); got != test.want {
				t.Fatalf("ipv6EnabledFor() = %v, want %v", got, test.want)
			}
		})
	}
}
