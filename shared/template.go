package shared

import (
	"net/netip"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

func DefaultTemplate(overrides ConfigOverrides) option.Options {
	var options option.Options

	options.Experimental = &option.ExperimentalOptions{
		ClashAPI: &option.ClashAPIOptions{
			ExternalController: "127.0.0.1:9090",
			StoreSelected:      true,
		},
	}

	options.Log = &option.LogOptions{
		Level:        "warn",
		Disabled:     false,
		Timestamp:    false,
		DisableColor: true,
	}

	options.DNS = &option.DNSOptions{
		DNSClientOptions: option.DNSClientOptions{
			Strategy:         option.DomainStrategy(dns.DomainStrategyPreferIPv4),
			IndependentCache: true,
		},
		Servers: []option.DNSServerOptions{
			{
				Tag:     "local",
				Address: "local",
				Detour:  "direct",
			},
			{
				Tag:             "dns-remote",
				Address:         pointerOrDefaultString(overrides.DNSRemote, "tcp://1.1.1.1"),
				AddressResolver: "local",
				Strategy:        option.DomainStrategy(dns.DomainStrategyPreferIPv4),
				Detour:          "select",
			},
		},
		Rules: []option.DNSRule{
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					ClashMode: "direct",
					Server:    "local",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					ClashMode: "global",
					Server:    "dns-remote",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					DomainSuffix: []string{"ir"},
					Server:       "local",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					Outbound: []string{"any"},
					Server:   "local",
				},
			},
		},
		ReverseMapping: true,
		Final:          "dns-remote",
	}

	if pointerOrDefaultBool(overrides.EnableTun, true) {
		options.Inbounds = append(
			options.Inbounds,
			option.Inbound{
				Type: C.TypeTun,
				Tag:  "tun-in",
				TunOptions: option.TunInboundOptions{
					Inet4Address: []option.ListenPrefix{
						option.ListenPrefix(netip.MustParsePrefix("172.19.0.1/30")),
					},
					MTU:                    9000,
					AutoRoute:              true,
					StrictRoute:            true,
					EndpointIndependentNat: true,
					InboundOptions: option.InboundOptions{
						SniffEnabled:             true,
						SniffOverrideDestination: true,
						DomainStrategy:           option.DomainStrategy(dns.DomainStrategyUseIPv4),
					},
				},
			},
		)
	}
	options.Inbounds = append(
		options.Inbounds,
		option.Inbound{
			Type: C.TypeMixed,
			Tag:  "mixed-in",
			MixedOptions: option.HTTPMixedInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen:     option.NewListenAddress(netip.MustParseAddr("127.0.0.1")),
					ListenPort: uint16(pointerOrDefaultInt(overrides.MixedPort, 2334)),
					InboundOptions: option.InboundOptions{
						SniffEnabled:             true,
						SniffOverrideDestination: true,
						DomainStrategy:           option.DomainStrategy(dns.DomainStrategyUseIPv4),
					},
				},
				SetSystemProxy: pointerOrDefaultBool(overrides.SetSystemProxy, true),
			},
		},
	)

	options.Route = &option.RouteOptions{
		Rules: []option.Rule{
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					Geosite:  []string{"category-ads-all"},
					Outbound: "block",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					Protocol: []string{"dns"},
					Outbound: "dns-out",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					ClashMode: "direct",
					Outbound:  "direct",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					ClashMode: "global",
					Outbound:  "select",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					GeoIP:        []string{"ir", "private"},
					DomainSuffix: []string{"ir"},
					Outbound:     "direct",
				},
			},
		},
		AutoDetectInterface: true,
		OverrideAndroidVPN:  true,
	}

	return options
}
