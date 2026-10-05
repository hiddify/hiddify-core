package config

import (
	"net/netip"
	"slices"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

func TestDNSGroupAddresses(t *testing.T) {
	cases := map[string][]string{
		"udp://1.1.1.1": {"udp://1.1.1.1", "tcp://1.1.1.1", "tls://1.1.1.1", "udp://8.8.8.8", "tcp://8.8.8.8", "tls://8.8.8.8"},
		// selected is already 8.8.8.8: fall back to 1.1.1.1
		"8.8.8.8":                      {"udp://8.8.8.8", "tcp://8.8.8.8", "tls://8.8.8.8", "udp://1.1.1.1", "tcp://1.1.1.1", "tls://1.1.1.1"},
		"https://dns.google/dns-query": {"https://dns.google/dns-query", "tcp://dns.google", "tls://dns.google", "udp://8.8.8.8", "tcp://8.8.8.8", "tls://8.8.8.8"},
		// duplicates of the selected address are dropped
		"tcp://9.9.9.9": {"tcp://9.9.9.9", "tls://9.9.9.9", "udp://8.8.8.8", "tcp://8.8.8.8", "tls://8.8.8.8"},
		// no host to derive variants from
		"local":                        {"local", "udp://8.8.8.8", "tcp://8.8.8.8", "tls://8.8.8.8"},
		"udp://[2606:4700:4700::1111]": {"udp://[2606:4700:4700::1111]", "tcp://[2606:4700:4700::1111]", "tls://[2606:4700:4700::1111]", "udp://8.8.8.8", "tcp://8.8.8.8", "tls://8.8.8.8"},
	}
	for selected, want := range cases {
		if got := dnsGroupAddresses(selected); !slices.Equal(got, want) {
			t.Errorf("%s:\n got %v\nwant %v", selected, got, want)
		}
	}
}

func TestBuildConfigDNSGroups(t *testing.T) {
	opts := DefaultHiddifyOptions()
	opts.RemoteDnsAddress = "udp://1.1.1.1"
	opts.DirectDnsAddress = "udp://9.9.9.9"
	options, err := BuildConfig(libbox.BaseContext(nil), opts, &ReadOptions{Content: chainTestProfile})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejected the config: %v", err)
	}
	servers := map[string]option.DNSServerOptions{}
	for _, server := range options.DNS.Servers {
		servers[server.Tag] = server
	}
	if options.DNS.Final != DNSMultiRemoteTag {
		t.Fatalf("DNS final = %q, want the remote group", options.DNS.Final)
	}
	for tag, want := range map[string]string{DNSMultiRemoteTag: DNSRemoteTag, DNSMultiDirectTag: DNSDirectTag} {
		group, ok := servers[tag]
		if !ok || group.Type != C.DNSTypeGroup {
			t.Fatalf("%s: group not found", tag)
		}
		g := group.Options.(*option.GroupDNSServerOptions)
		if g.Mode != C.DNSGroupModeSequential {
			t.Fatalf("%s: mode %q, want sequential", tag, g.Mode)
		}
		if len(g.IgnoreRanges) != 1 || netip.Prefix(g.IgnoreRanges[0]).String() != dnsGroupIgnoreRange {
			t.Fatalf("%s: ignore ranges %v", tag, g.IgnoreRanges)
		}
		if g.Servers[0] != want || len(g.Servers) != 6 {
			t.Fatalf("%s: members %v, want %s first and 6 members", tag, g.Servers, want)
		}
		wantTypes := []string{"", C.DNSTypeTCP, C.DNSTypeTLS, C.DNSTypeUDP, C.DNSTypeTCP, C.DNSTypeTLS}
		for i, member := range g.Servers[1:] {
			if servers[member].Type != wantTypes[i+1] {
				t.Fatalf("%s: member %d (%s) type %q, want %q", tag, i+1, member, servers[member].Type, wantTypes[i+1])
			}
		}
	}
	// remote members dial through the main selector
	if d := servers[DNSMultiRemoteTag+"-1"].Options.(*option.RemoteDNSServerOptions).Detour; d != OutboundSelectTag {
		t.Fatalf("remote tcp member detour = %q, want %q", d, OutboundSelectTag)
	}
}
