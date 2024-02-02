package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"runtime"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

func BuildConfigJson(configOpt ConfigOptions, input option.Options) (string, error) {
	options, err := BuildConfig(configOpt, input)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	json.NewEncoder(&buffer)
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(options)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// TODO include selectors
func BuildConfig(configOpt ConfigOptions, input option.Options) (*option.Options, error) {
	if configOpt.ExecuteAsIs {
		return applyOverrides(configOpt, input), nil
	}

	fmt.Printf("config options: %+v\n", configOpt)

	var options option.Options
	directDNSDomains := []string{}
	dnsRules := []option.DefaultDNSRule{}

	var bind string
	if configOpt.AllowConnectionFromLAN {
		bind = "0.0.0.0"
	} else {
		bind = "127.0.0.1"
	}

	if configOpt.EnableClashApi {
		options.Experimental = &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", configOpt.ClashApiPort),
			},
			// CacheFile: &option.CacheFileOptions{
			// 	Enabled: true,
			// 	Path:    "clash.db",
			// },
		}
	}

	options.Log = &option.LogOptions{
		Level: configOpt.LogLevel,
		// Output:       "box.log",
		Disabled:     false,
		Timestamp:    true,
		DisableColor: true,
	}

	options.DNS = &option.DNSOptions{
		StaticIPs: map[string][]string{
			"sky.rethinkdns.com": getIPs([]string{"zula.ir", "www.speedtest.net", "sky.rethinkdns.com"}),
		},
		DNSClientOptions: option.DNSClientOptions{
			IndependentCache: configOpt.IndependentDNSCache,
		},
		Final: "dns-remote",
		Servers: []option.DNSServerOptions{
			{
				Tag:             "dns-remote",
				Address:         configOpt.RemoteDnsAddress,
				AddressResolver: "dns-direct",
				Strategy:        configOpt.RemoteDnsDomainStrategy,
			},
			{
				Tag:     "dns-trick-direct",
				Address: "https://sky.rethinkdns.com/",
				// AddressResolver: "dns-local",
				Strategy: configOpt.DirectDnsDomainStrategy,
				Detour:   "direct-fragment",
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
	}

	var inboundDomainStrategy option.DomainStrategy
	if !configOpt.ResolveDestination {
		inboundDomainStrategy = option.DomainStrategy(dns.DomainStrategyAsIS)
	} else {
		inboundDomainStrategy = configOpt.IPv6Mode
	}

	if configOpt.EnableTun {
		if runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
			tunInbound := option.Inbound{
				Type: C.TypeTun,
				Tag:  "tun-in",
				TunOptions: option.TunInboundOptions{
					Stack:                  configOpt.TUNStack,
					MTU:                    configOpt.MTU,
					AutoRoute:              true,
					StrictRoute:            configOpt.StrictRoute,
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
				tunInbound.TunOptions.Inet4Address = []netip.Prefix{
					netip.MustParsePrefix("172.19.0.1/28"),
				}
			case option.DomainStrategy(dns.DomainStrategyUseIPv6):
				tunInbound.TunOptions.Inet6Address = []netip.Prefix{
					netip.MustParsePrefix("fdfe:dcba:9876::1/126"),
				}
			default:
				tunInbound.TunOptions.Inet4Address = []netip.Prefix{
					netip.MustParsePrefix("172.19.0.1/28"),
				}
				tunInbound.TunOptions.Inet6Address = []netip.Prefix{
					netip.MustParsePrefix("fdfe:dcba:9876::1/126"),
				}
			}
			options.Inbounds = append(options.Inbounds, tunInbound)
		}
	}

	options.Inbounds = append(
		options.Inbounds,
		option.Inbound{
			Type: C.TypeMixed,
			Tag:  "mixed-in",
			MixedOptions: option.HTTPMixedInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen:     option.NewListenAddress(netip.MustParseAddr(bind)),
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

	options.Inbounds = append(
		options.Inbounds,
		option.Inbound{
			Type: C.TypeDirect,
			Tag:  "dns-in",
			DirectOptions: option.DirectInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen:     option.NewListenAddress(netip.MustParseAddr(bind)),
					ListenPort: configOpt.LocalDnsPort,
				},
				OverrideAddress: "1.1.1.1",
				OverridePort:    53,
			},
		},
	)

	remoteDNSAddress := configOpt.RemoteDnsAddress
	if strings.Contains(remoteDNSAddress, "://") {
		remoteDNSAddress = strings.SplitAfter(remoteDNSAddress, "://")[1]
	}
	parsedUrl, err := url.Parse(fmt.Sprintf("https://%s", remoteDNSAddress))
	if err == nil && net.ParseIP(parsedUrl.Host) == nil {
		directDNSDomains = append(directDNSDomains, fmt.Sprintf("full:%s", parsedUrl.Host))
		//TODO: IS it really needed
	}

	routeRules := []option.Rule{
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
	}

	if configOpt.BypassLAN {
		routeRules = append(
			routeRules,
			option.Rule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					GeoIP:    []string{"private"},
					Outbound: "bypass",
				},
			},
		)
	}

	if configOpt.EnableFakeDNS {
		inet4Range := netip.MustParsePrefix("198.18.0.0/15")
		inet6Range := netip.MustParsePrefix("fc00::/18")
		options.DNS.FakeIP = &option.DNSFakeIPOptions{
			Enabled:    true,
			Inet4Range: &inet4Range,
			Inet6Range: &inet6Range,
		}
		options.DNS.Servers = append(
			options.DNS.Servers,
			option.DNSServerOptions{
				Tag:      "dns-fake",
				Address:  "fakeip",
				Strategy: option.DomainStrategy(dns.DomainStrategyUseIPv4),
			},
		)
		options.DNS.Rules = append(
			options.DNS.Rules,
			option.DNSRule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					Inbound:      []string{"tun-in"},
					Server:       "dns-fake",
					DisableCache: true,
				},
			},
		)

	}

	for _, rule := range configOpt.Rules {
		routeRule := rule.MakeRule()
		switch rule.Outbound {
		case "bypass":
			routeRule.Outbound = "bypass"
		case "block":
			routeRule.Outbound = "block"
		case "proxy":
			routeRule.Outbound = "dns-out"
		}

		if routeRule.IsValid() {
			routeRules = append(
				routeRules,
				option.Rule{
					Type:           C.RuleTypeDefault,
					DefaultOptions: routeRule,
				},
			)
		}

		dnsRule := rule.MakeDNSRule()
		switch rule.Outbound {
		case "bypass":
			dnsRule.Server = "dns-direct"
		case "block":
			dnsRule.Server = "dns-block"
			dnsRule.DisableCache = true
		case "proxy":
			if configOpt.EnableFakeDNS {
				fakeDnsRule := dnsRule
				fakeDnsRule.Server = "dns-fake"
				fakeDnsRule.Inbound = []string{"tun-in"}
				dnsRules = append(dnsRules, fakeDnsRule)
			}
			dnsRule.Server = "dns-remote"
		}
		dnsRules = append(dnsRules, dnsRule)
	}

	if configOpt.EnableDNSRouting {
		for _, dnsRule := range dnsRules {
			if dnsRule.IsValid() {
				options.DNS.Rules = append(
					options.DNS.Rules,
					option.DNSRule{
						Type:           C.RuleTypeDefault,
						DefaultOptions: dnsRule,
					},
				)
			}
		}
	}
	if options.DNS.Rules == nil {
		options.DNS.Rules = []option.DNSRule{}
	}
	var dnsCPttl uint32 = 3000
	options.DNS.Rules = append(
		options.DNS.Rules,
		option.DNSRule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultDNSRule{
				Domain:       []string{"cp.cloudflare.com"},
				Server:       "dns-remote",
				RewriteTTL:   &dnsCPttl,
				DisableCache: false,
			},
		},
	)

	options.Route = &option.RouteOptions{
		Rules:               routeRules,
		AutoDetectInterface: true,
		OverrideAndroidVPN:  true,
		GeoIP: &option.GeoIPOptions{
			Path: configOpt.GeoIPPath,
		},
		Geosite: &option.GeositeOptions{
			Path: configOpt.GeoSitePath,
		},
	}

	var outbounds []option.Outbound
	var tags []string
	for _, out := range input.Outbounds {
		outbound, serverDomain, err := patchOutbound(out, configOpt)
		if err != nil {
			return nil, err
		}

		if serverDomain != "" {
			directDNSDomains = append(directDNSDomains, serverDomain)
		}
		out = *outbound

		switch out.Type {
		case C.TypeDirect, C.TypeBlock, C.TypeDNS:
			continue
		case C.TypeSelector, C.TypeURLTest:
			continue
		case C.TypeCustom:
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
			Outbounds:   tags,
			URL:         configOpt.ConnectionTestUrl,
			Interval:    configOpt.URLTestInterval,
			IdleTimeout: configOpt.URLTestIdleTimeout,
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
				Tag:  "direct-fragment",
				Type: C.TypeDirect,
				DirectOptions: option.DirectOutboundOptions{
					DialerOptions: option.DialerOptions{
						TLSFragment: &option.TLSFragmentOptions{
							Enabled: true,
							Size:    configOpt.TLSTricks.FragmentSize,
							Sleep:   configOpt.TLSTricks.FragmentSleep,
						},
					},
				},
			},
			{
				Tag:  "bypass",
				Type: C.TypeDirect,
			},
			{
				Tag:  "block",
				Type: C.TypeBlock,
			},
		}...,
	)
	if len(directDNSDomains) > 0 {
		trickDnsDomains := []string{}
		directDNSDomains = removeDuplicateStr(directDNSDomains)
		for _, d := range directDNSDomains {
			if isBlockedDomain(d) {
				trickDnsDomains = append(trickDnsDomains, d)
			}
		}
		trickDomains := strings.Join(trickDnsDomains, ",")
		trickRule := Rule{Domains: trickDomains, Outbound: "bypass"}
		trickdnsRule := trickRule.MakeDNSRule()
		trickdnsRule.Server = "dns-trick-direct"
		options.DNS.Rules = append([]option.DNSRule{{Type: C.RuleTypeDefault, DefaultOptions: trickdnsRule}}, options.DNS.Rules...)

		domains := strings.Join(directDNSDomains, ",")
		directRule := Rule{Domains: domains, Outbound: "bypass"}
		dnsRule := directRule.MakeDNSRule()
		dnsRule.Server = "dns-direct"
		options.DNS.Rules = append([]option.DNSRule{{Type: C.RuleTypeDefault, DefaultOptions: dnsRule}}, options.DNS.Rules...)
	}

	return &options, nil
}

func getIPs(domains []string) []string {
	res := []string{}
	for _, d := range domains {
		ips, err := net.LookupHost(d)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			if !strings.HasPrefix(ip, "10.") {
				res = append(res, ip)
			}
		}
	}
	return res
}

func isBlockedDomain(domain string) bool {
	if strings.HasPrefix("full:", domain) {
		return false
	}
	ips, err := net.LookupHost(domain)
	if err != nil {
		// fmt.Println(err)
		return true
	}

	// Print the IP addresses associated with the domain
	fmt.Printf("IP addresses for %s:\n", domain)
	for _, ip := range ips {
		if strings.HasPrefix(ip, "10.") {
			return true
		}
	}
	return false
}

func applyOverrides(overrides ConfigOptions, options option.Options) *option.Options {
	if overrides.EnableClashApi {
		options.Experimental.ClashAPI = &option.ClashAPIOptions{
			ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", overrides.ClashApiPort),
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

	return &options
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
