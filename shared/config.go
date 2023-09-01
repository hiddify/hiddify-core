package shared

import (
	"fmt"
	"net/netip"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

type ConfigOptions struct {
	ExecuteAsIs             bool                  `json:"execute-config-as-is"`
	LogLevel                string                `json:"log-level"`
	ResolveDestination      bool                  `json:"resolve-destination"`
	IPv6Mode                option.DomainStrategy `json:"ipv6-mode"`
	RemoteDnsAddress        string                `json:"remote-dns-address"`
	RemoteDnsDomainStrategy option.DomainStrategy `json:"remote-dns-domain-strategy"`
	DirectDnsAddress        string                `json:"direct-dns-address"`
	DirectDnsDomainStrategy option.DomainStrategy `json:"direct-dns-domain-strategy"`
	MixedPort               uint16                `json:"mixed-port"`
	LocalDnsPort            uint16                `json:"local-dns-port"`
	MTU                     uint32                `json:"mtu"`
	ConnectionTestUrl       string                `json:"connection-test-url"`
	URLTestInterval         option.Duration       `json:"url-test-interval"`
	EnableClashApi          bool                  `json:"enable-clash-api"`
	ClashApiPort            uint16                `json:"clash-api-port"`
	EnableTun               bool                  `json:"enable-tun"`
	SetSystemProxy          bool                  `json:"set-system-proxy"`
}

func BuildConfig(configOpt ConfigOptions, input option.Options) option.Options {
	if configOpt.ExecuteAsIs {
		return applyOverrides(configOpt, input)
	}

	var options option.Options

	fmt.Printf("%+v\n", configOpt)

	if configOpt.EnableClashApi {
		options.Experimental = &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", configOpt.ClashApiPort),
				StoreSelected:      true,
			},
		}
	}

	options.Log = &option.LogOptions{
		Level:        configOpt.LogLevel,
		Output:       "box.log",
		Disabled:     false,
		Timestamp:    false,
		DisableColor: true,
	}

	options.DNS = &option.DNSOptions{
		DNSClientOptions: option.DNSClientOptions{
			IndependentCache: true,
		},
		Servers: []option.DNSServerOptions{
			{
				Tag:             "dns-remote",
				Address:         configOpt.RemoteDnsAddress,
				AddressResolver: "dns-direct",
				Strategy:        configOpt.RemoteDnsDomainStrategy,
				Detour:          "select",
			},
			{
				Tag:             "dns-direct",
				Address:         configOpt.DirectDnsAddress,
				AddressResolver: "dns-local",
				Strategy:        configOpt.DirectDnsDomainStrategy,
				Detour:          "direct",
			},
			{
				Tag:     "dns-local",
				Address: "local",
				Detour:  "direct",
			},
			{
				Tag:     "dns-block",
				Address: "rcode://success",
			},
		},
		Rules: []option.DNSRule{
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					Outbound: []string{"any"},
					// Server:   "dns-direct",
					Server: "dns-local",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					ClashMode: "Direct",
					Server:    "dns-local",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					ClashMode: "Global",
					Server:    "dns-remote",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					DomainSuffix: []string{"ir"},
					Server:       "dns-local",
				},
			},
		},
	}

	var inboundDomainStrategy option.DomainStrategy
	if !configOpt.ResolveDestination {
		inboundDomainStrategy = option.DomainStrategy(dns.DomainStrategyAsIS)
	} else {
		inboundDomainStrategy = configOpt.IPv6Mode
	}

	if configOpt.EnableTun {
		tunInbound := option.Inbound{
			Type: C.TypeTun,
			Tag:  "tun-in",
			TunOptions: option.TunInboundOptions{
				MTU:                    configOpt.MTU,
				AutoRoute:              true,
				StrictRoute:            true,
				EndpointIndependentNat: true,
				InboundOptions: option.InboundOptions{
					SniffEnabled:             true,
					SniffOverrideDestination: true,
					DomainStrategy:           inboundDomainStrategy,
				},
			},
		}
		switch configOpt.IPv6Mode {
		case option.DomainStrategy(dns.DomainStrategyUseIPv4):
			tunInbound.TunOptions.Inet4Address = []option.ListenPrefix{
				option.ListenPrefix(netip.MustParsePrefix("172.19.0.1/28")),
			}
		case option.DomainStrategy(dns.DomainStrategyUseIPv6):
			tunInbound.TunOptions.Inet6Address = []option.ListenPrefix{
				option.ListenPrefix(netip.MustParsePrefix("fdfe:dcba:9876::1/126")),
			}
		default:
			tunInbound.TunOptions.Inet4Address = []option.ListenPrefix{
				option.ListenPrefix(netip.MustParsePrefix("172.19.0.1/28")),
			}
			tunInbound.TunOptions.Inet6Address = []option.ListenPrefix{
				option.ListenPrefix(netip.MustParsePrefix("fdfe:dcba:9876::1/126")),
			}
		}
		options.Inbounds = append(options.Inbounds, tunInbound)
	}

	options.Inbounds = append(
		options.Inbounds,
		option.Inbound{
			Type: C.TypeMixed,
			Tag:  "mixed-in",
			MixedOptions: option.HTTPMixedInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen:     option.NewListenAddress(netip.MustParseAddr("127.0.0.1")),
					ListenPort: configOpt.MixedPort,
					InboundOptions: option.InboundOptions{
						SniffEnabled:             true,
						SniffOverrideDestination: true,
						DomainStrategy:           inboundDomainStrategy,
					},
				},
				SetSystemProxy: configOpt.SetSystemProxy,
			},
		},
	)

	options.Inbounds = append(options.Inbounds,
		option.Inbound{
			Type: C.TypeDirect,
			Tag:  "dns-in",
			DirectOptions: option.DirectInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen:     option.NewListenAddress(netip.MustParseAddr("127.0.0.1")),
					ListenPort: configOpt.LocalDnsPort,
				},
				OverrideAddress: "8.8.8.8",
				OverridePort:    53,
			},
		},
	)

	options.Route = &option.RouteOptions{
		Rules: []option.Rule{
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					Inbound:  []string{"dns-in"},
					Outbound: "dns-out",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					Port:     []uint16{53},
					Outbound: "dns-out",
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
					ClashMode: "Direct",
					Outbound:  "direct",
				},
			},
			{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					ClashMode: "Global",
					Outbound:  "select",
				},
			},
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
					GeoIP:        []string{"ir", "private"},
					DomainSuffix: []string{"ir"},
					Outbound:     "direct",
				},
			},
		},
		AutoDetectInterface: true,
		OverrideAndroidVPN:  true,
	}

	var outbounds []option.Outbound
	var tags []string
	for _, out := range input.Outbounds {
		switch out.Type {
		case C.TypeDirect, C.TypeBlock, C.TypeDNS:
			continue
		case C.TypeSelector, C.TypeURLTest:
			continue
		default:
			tags = append(tags, out.Tag)
			outbounds = append(outbounds, out)
		}
	}

	urlTest := option.Outbound{
		Type: C.TypeURLTest,
		Tag:  "auto",
		URLTestOptions: option.URLTestOutboundOptions{
			Outbounds: tags,
			URL:       configOpt.ConnectionTestUrl,
			Interval:  configOpt.URLTestInterval,
		},
	}

	selector := option.Outbound{
		Type: C.TypeSelector,
		Tag:  "select",
		SelectorOptions: option.SelectorOutboundOptions{
			Outbounds: append([]string{urlTest.Tag}, tags...),
			Default:   urlTest.Tag,
		},
	}

	outbounds = append([]option.Outbound{selector, urlTest}, outbounds...)

	options.Outbounds = append(
		outbounds,
		[]option.Outbound{
			{
				Tag:  "dns-out",
				Type: C.TypeDNS,
			},
			{
				Tag:  "direct",
				Type: C.TypeDirect,
			},
			{
				Tag:  "block",
				Type: C.TypeBlock,
			},
		}...,
	)

	return options
}

func applyOverrides(overrides ConfigOptions, options option.Options) option.Options {
	if overrides.EnableClashApi {
		options.Experimental.ClashAPI = &option.ClashAPIOptions{
			ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", overrides.ClashApiPort),
			StoreSelected:      true,
		}
	}

	options.Log = &option.LogOptions{
		Level:    overrides.LogLevel,
		Output:   "box.log",
		Disabled: false,
	}

	var inbounds []option.Inbound
	for _, inb := range options.Inbounds {
		if inb.Type == C.TypeTun && !overrides.EnableTun {
			continue
		}
		inbounds = append(inbounds, inb)
	}
	options.Inbounds = inbounds

	return options
}
