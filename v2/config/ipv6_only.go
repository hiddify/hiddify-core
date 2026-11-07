package config

import (
	"net"
	"net/netip"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

// An IPv6-only host (no IPv4 address at all, as on some servers and mobile networks without
// 464XLAT) reached nothing with the defaults: IPv6 mode "ipv4_only" rejects IPv6 and resolves only
// A records, and the default resolvers are IPv4 addresses (8.8.8.8, 1.1.1.1). The config is
// adapted to such a host: IPv6 is preferred, and the well-known resolvers are reached on their own
// IPv6 addresses. Nothing changes on a host with IPv4.

// the TUN's own IPv4 network, which is no way out
var tunIPv4Network = netip.MustParsePrefix("172.19.0.0/28")

// hostHasIPv4 reports whether any interface has a usable IPv4 address. When the interfaces cannot
// be listed (a sandbox, Android's restrictions) the host is taken to have IPv4: nothing is adapted.
var hostHasIPv4 = func() bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return true
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			prefix, err := netip.ParsePrefix(address.String())
			if err != nil {
				continue
			}
			ip := prefix.Addr().Unmap()
			if ip.Is4() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !tunIPv4Network.Contains(ip) {
				return true
			}
		}
	}
	return false
}

// the IPv6 addresses of the public resolvers whose IPv4 addresses the options may name
var resolverIPv6 = map[string]string{
	"8.8.8.8":         "2001:4860:4860::8888",
	"8.8.4.4":         "2001:4860:4860::8844",
	"1.1.1.1":         "2606:4700:4700::1111",
	"1.0.0.1":         "2606:4700:4700::1001",
	"9.9.9.9":         "2620:fe::fe",
	"149.112.112.112": "2620:fe::9",
	"94.140.14.14":    "2a10:50c0::ad1:ff",
	"94.140.15.15":    "2a10:50c0::ad2:ff",
}

// forIPv6OnlyHost returns the options to build with on an IPv6-only host: "ipv4_only" becomes
// "prefer_ipv6". Any other choice is the user's and stays.
func forIPv6OnlyHost(hopts *HiddifyOptions, ipv6Only bool) *HiddifyOptions {
	if !ipv6Only || hopts.IPv6Mode != option.DomainStrategy(C.DomainStrategyIPv4Only) {
		return hopts
	}
	adapted := *hopts
	adapted.IPv6Mode = option.DomainStrategy(C.DomainStrategyPreferIPv6)
	log.Warn("this host has no IPv4 address: IPv6 mode ipv4_only runs as prefer_ipv6")
	return &adapted
}

// useIPv6Resolvers points the DNS servers named by a public resolver's IPv4 address at the same
// resolver's IPv6 address. Their certificates name both, so DoH and DoT keep verifying.
func useIPv6Resolvers(options *option.Options) {
	if options.DNS == nil {
		return
	}
	for i := range options.DNS.Servers {
		var address *option.DNSServerAddressOptions
		switch server := options.DNS.Servers[i].Options.(type) {
		case *option.RemoteDNSServerOptions:
			address = &server.DNSServerAddressOptions
		case *option.RemoteTLSDNSServerOptions:
			address = &server.DNSServerAddressOptions
		case *option.RemoteHTTPSDNSServerOptions:
			address = &server.DNSServerAddressOptions
		}
		if address == nil {
			continue
		}
		if ipv6, found := resolverIPv6[address.Server]; found {
			address.Server = ipv6
		}
	}
}
