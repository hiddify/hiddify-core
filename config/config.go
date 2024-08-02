package config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

const (
	DNSRemoteTag       = "dns-remote"
	DNSLocalTag        = "dns-local"
	DNSDirectTag       = "dns-direct"
	DNSBlockTag        = "dns-block"
	DNSFakeTag         = "dns-fake"
	DNSTricksDirectTag = "dns-trick-direct"

	OutboundDirectTag         = "direct"
	OutboundBypassTag         = "bypass"
	OutboundBlockTag          = "block"
	OutboundSelectTag         = "select"
	OutboundURLTestTag        = "auto"
	OutboundDNSTag            = "dns-out"
	OutboundDirectFragmentTag = "direct-fragment"

	InboundTUNTag   = "tun-in"
	InboundMixedTag = "mixed-in"
	InboundDNSTag   = "dns-in"
)

var OutboundMainProxyTag = OutboundSelectTag

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
func BuildConfig(opt ConfigOptions, input option.Options) (*option.Options, error) {
	fmt.Printf("config options: %+v\n", opt)

	var options option.Options
	if opt.EnableFullConfig {
		options.Inbounds = input.Inbounds
		options.DNS = input.DNS
		options.Route = input.Route
	}
	directDNSDomains := make(map[string]bool)
	dnsRules := []option.DefaultDNSRule{}

	var bind string
	if opt.AllowConnectionFromLAN {
		bind = "0.0.0.0"
	} else {
		bind = "127.0.0.1"
	}

	if opt.EnableClashApi {
		if opt.ClashApiSecret == "" {
			opt.ClashApiSecret = generateRandomString(16)
		}
		options.Experimental = &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", opt.ClashApiPort),
				Secret:             opt.ClashApiSecret,
			},

			CacheFile: &option.CacheFileOptions{
				Enabled: true,
				Path:    "clash.db",
			},
		}
	}

	options.Log = &option.LogOptions{
		Level:        opt.LogLevel,
		Output:       "box.log",
		Disabled:     false,
		Timestamp:    true,
		DisableColor: true,
	}

	options.DNS = &option.DNSOptions{
		StaticIPs: map[string][]string{
			"sky.rethinkdns.com": getIPs([]string{"www.speedtest.net", "sky.rethinkdns.com"}),
		},
		DNSClientOptions: option.DNSClientOptions{
			IndependentCache: opt.IndependentDNSCache,
		},
		Final: DNSRemoteTag,
		Servers: []option.DNSServerOptions{
			{
				Tag:             DNSRemoteTag,
				Address:         opt.RemoteDnsAddress,
				AddressResolver: DNSDirectTag,
				Strategy:        opt.RemoteDnsDomainStrategy,
			},
			{
				Tag:     DNSTricksDirectTag,
				Address: "https://sky.rethinkdns.com/",
				// AddressResolver: "dns-local",
				Strategy: opt.DirectDnsDomainStrategy,
				Detour:   OutboundDirectFragmentTag,
			},
			{
				Tag:             DNSDirectTag,
				Address:         opt.DirectDnsAddress,
				AddressResolver: DNSLocalTag,
				Strategy:        opt.DirectDnsDomainStrategy,
				Detour:          OutboundDirectTag,
			},
			{
				Tag:     DNSLocalTag,
				Address: "local",
				Detour:  OutboundDirectTag,
			},
			{
				Tag:     DNSBlockTag,
				Address: "rcode://success",
			},
		},
	}

	var inboundDomainStrategy option.DomainStrategy
	if !opt.ResolveDestination {
		inboundDomainStrategy = option.DomainStrategy(dns.DomainStrategyAsIS)
	} else {
		inboundDomainStrategy = opt.IPv6Mode
	}
	if opt.EnableTunService {
		ActivateTunnelService(opt)
	} else if opt.EnableTun {
		tunInbound := option.Inbound{
			Type: C.TypeTun,
			Tag:  InboundTUNTag,
			TunOptions: option.TunInboundOptions{
				Stack:                  opt.TUNStack,
				MTU:                    opt.MTU,
				AutoRoute:              true,
				StrictRoute:            opt.StrictRoute,
				EndpointIndependentNat: true,
				// GSO:                    runtime.GOOS != "windows",
				InboundOptions: option.InboundOptions{
					SniffEnabled:             true,
					SniffOverrideDestination: true,
					DomainStrategy:           inboundDomainStrategy,
				},
			},
		}
		switch opt.IPv6Mode {
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

	options.Inbounds = append(
		options.Inbounds,
		option.Inbound{
			Type: C.TypeMixed,
			Tag:  InboundMixedTag,
			MixedOptions: option.HTTPMixedInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen:     option.NewListenAddress(netip.MustParseAddr(bind)),
					ListenPort: opt.MixedPort,
					InboundOptions: option.InboundOptions{
						SniffEnabled:             true,
						SniffOverrideDestination: true,
						DomainStrategy:           inboundDomainStrategy,
					},
				},
				SetSystemProxy: opt.SetSystemProxy,
			},
		},
	)

	options.Inbounds = append(
		options.Inbounds,
		option.Inbound{
			Type: C.TypeDirect,
			Tag:  InboundDNSTag,
			DirectOptions: option.DirectInboundOptions{
				ListenOptions: option.ListenOptions{
					Listen:     option.NewListenAddress(netip.MustParseAddr(bind)),
					ListenPort: opt.LocalDnsPort,
				},
				// OverrideAddress: "1.1.1.1",
				// OverridePort:    53,
			},
		},
	)

	remoteDNSAddress := opt.RemoteDnsAddress
	if strings.Contains(remoteDNSAddress, "://") {
		remoteDNSAddress = strings.SplitAfter(remoteDNSAddress, "://")[1]
	}
	parsedUrl, err := url.Parse(fmt.Sprintf("https://%s", remoteDNSAddress))
	if err == nil && net.ParseIP(parsedUrl.Host) == nil {
		directDNSDomains["full:"+parsedUrl.Host] = true
		//TODO: IS it really needed
	}

	routeRules := []option.Rule{
		{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				Inbound:  []string{InboundDNSTag},
				Outbound: OutboundDNSTag,
			},
		},
		{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				Port:     []uint16{53},
				Outbound: OutboundDNSTag,
			},
		},
		{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				ClashMode: "Direct",
				Outbound:  OutboundDirectTag,
			},
		},
		{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				ClashMode: "Global",
				Outbound:  OutboundMainProxyTag,
			},
		},
	}
	if opt.EnableTun {
		routeRules = append(
			routeRules,
			option.Rule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					ProcessName: []string{"Hiddify", "Hiddify.exe", "HiddifyCli", "HiddifyCli.exe"},
					Outbound:    OutboundBypassTag,
				},
			},
		)
	}
	if opt.BypassLAN {
		routeRules = append(
			routeRules,
			option.Rule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					GeoIP:    []string{"private"},
					Outbound: OutboundBypassTag,
				},
			},
		)
	}

	if opt.EnableFakeDNS {
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
				Tag:      DNSFakeTag,
				Address:  "fakeip",
				Strategy: option.DomainStrategy(dns.DomainStrategyUseIPv4),
			},
		)
		options.DNS.Rules = append(
			options.DNS.Rules,
			option.DNSRule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					Inbound:      []string{InboundTUNTag},
					Server:       DNSFakeTag,
					DisableCache: true,
				},
			},
		)

	}

	for _, rule := range opt.Rules {
		routeRule := rule.MakeRule()
		switch rule.Outbound {
		case "bypass":
			routeRule.Outbound = OutboundBypassTag
		case "block":
			routeRule.Outbound = OutboundBlockTag
		case "proxy":
			routeRule.Outbound = OutboundDNSTag
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
			dnsRule.Server = DNSDirectTag
		case "block":
			dnsRule.Server = DNSBlockTag
			dnsRule.DisableCache = true
		case "proxy":
			if opt.EnableFakeDNS {
				fakeDnsRule := dnsRule
				fakeDnsRule.Server = DNSFakeTag
				fakeDnsRule.Inbound = []string{InboundTUNTag}
				dnsRules = append(dnsRules, fakeDnsRule)
			}
			dnsRule.Server = DNSRemoteTag
		}
		dnsRules = append(dnsRules, dnsRule)
	}

	if opt.EnableDNSRouting {
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
	parsedURL, err := url.Parse(opt.ConnectionTestUrl)
	if err == nil {
		options.DNS.Rules = append(
			options.DNS.Rules,
			option.DNSRule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					Domain:       []string{parsedURL.Host},
					Server:       DNSRemoteTag,
					RewriteTTL:   &dnsCPttl,
					DisableCache: false,
				},
			},
		)
	}

	options.Route = &option.RouteOptions{
		Rules:               routeRules,
		AutoDetectInterface: true,
		OverrideAndroidVPN:  true,
		// RuleSet:             []option.RuleSet{},
		// GeoIP: &option.GeoIPOptions{
		// 	Path: opt.GeoIPPath,
		// },
		// Geosite: &option.GeositeOptions{
		// 	Path: opt.GeoSitePath,
		// },
	}
	if opt.Region != "other" {
		options.DNS.Rules = append(
			options.DNS.Rules,
			option.DNSRule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					RuleSet: []string{
						"geoip-" + opt.Region,
						"geosite-" + opt.Region,
					},
					Server: DNSDirectTag,
				},
			},
		)
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geoip-" + opt.Region,
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL: "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/country/geoip-" + opt.Region + ".srs",

				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-" + opt.Region,
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/country/geosite-" + opt.Region + ".srs",
				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})

		routeRuleIp := option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RuleSet: []string{
					"geoip-" + opt.Region,
					"geosite-" + opt.Region,
				},
				Outbound: OutboundDirectTag,
			},
		}

		options.Route.Rules = append([]option.Rule{routeRuleIp}, options.Route.Rules...)
	}
	if opt.BlockAds {
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-ads",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-category-ads-all.srs",
				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-malware",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-malware.srs",
				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-phishing",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-phishing.srs",
				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-cryptominers",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-cryptominers.srs",
				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geoip-phishing",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geoip-phishing.srs",
				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})
		options.Route.RuleSet = append(options.Route.RuleSet, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geoip-malware",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geoip-malware.srs",
				UpdateInterval: option.Duration(5 * time.Hour * 24),
			},
		})

		routeRule := option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RuleSet: []string{
					"geosite-ads",
					"geosite-malware",
					"geosite-phishing",
					"geosite-cryptominers",
					"geoip-malware",
					"geoip-phishing",
				},
				Outbound: OutboundBlockTag,
			},
		}
		options.Route.Rules = append([]option.Rule{routeRule}, options.Route.Rules...)
	}
	var outbounds []option.Outbound
	var tags []string
	OutboundMainProxyTag = OutboundSelectTag
	//inbound==warp over proxies
	//outbound==proxies over warp
	if opt.Warp.EnableWarp {
		for _, out := range input.Outbounds {
			if out.Type == C.TypeCustom {
				if warp, ok := out.CustomOptions["warp"].(map[string]interface{}); ok {
					key, _ := warp["key"].(string)
					if key == "p1" {
						opt.Warp.EnableWarp = false
						break
					}
				}
			}
			if out.Type == C.TypeWireGuard && (out.WireGuardOptions.PrivateKey == opt.Warp.WireguardConfig.PrivateKey || out.WireGuardOptions.PrivateKey == "p1") {
				opt.Warp.EnableWarp = false
				break
			}
		}
	}
	if opt.Warp.EnableWarp && (opt.Warp.Mode == "warp_over_proxy" || opt.Warp.Mode == "proxy_over_warp") {
		out, err := GenerateWarpSingbox(opt.Warp.WireguardConfig, opt.Warp.CleanIP, opt.Warp.CleanPort, opt.Warp.FakePackets, opt.Warp.FakePacketSize, opt.Warp.FakePacketDelay, opt.Warp.FakePacketMode)
		if err != nil {
			return nil, fmt.Errorf("failed to generate warp config: %v", err)
		}
		out.Tag = "Hiddify Warp ✅"
		if opt.Warp.Mode == "warp_over_proxy" {
			out.WireGuardOptions.Detour = OutboundSelectTag
			OutboundMainProxyTag = out.Tag
		} else {
			out.WireGuardOptions.Detour = OutboundDirectTag
		}
		patchWarp(out, &opt, true, nil)
		outbounds = append(outbounds, *out)
		// tags = append(tags, out.Tag)
	}
	for _, out := range input.Outbounds {
		outbound, serverDomain, err := patchOutbound(out, opt, options.DNS.StaticIPs)
		if err != nil {
			return nil, err
		}

		if serverDomain != "" {
			directDNSDomains[serverDomain] = true
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
			if !strings.Contains(out.Tag, "§hide§") {
				tags = append(tags, out.Tag)
			}
			out = patchHiddifyWarpFromConfig(out, opt)
			outbounds = append(outbounds, out)
		}
	}

	urlTest := option.Outbound{
		Type: C.TypeURLTest,
		Tag:  OutboundURLTestTag,
		URLTestOptions: option.URLTestOutboundOptions{
			Outbounds: tags,
			URL:       opt.ConnectionTestUrl,
			Interval:  option.Duration(opt.URLTestInterval.Duration()),
			// IdleTimeout: option.Duration(opt.URLTestIdleTimeout.Duration()),
			Tolerance:   1,
			IdleTimeout: option.Duration(opt.URLTestInterval.Duration().Nanoseconds() * 3),
		},
	}
	defaultSelect := urlTest.Tag

	for _, tag := range tags {
		if strings.Contains(tag, "§default§") {
			defaultSelect = "§default§"
		}
	}
	selector := option.Outbound{
		Type: C.TypeSelector,
		Tag:  OutboundSelectTag,
		SelectorOptions: option.SelectorOutboundOptions{
			Outbounds: append([]string{urlTest.Tag}, tags...),
			Default:   defaultSelect,
		},
	}

	outbounds = append([]option.Outbound{selector, urlTest}, outbounds...)

	options.Outbounds = append(
		outbounds,
		[]option.Outbound{
			{
				Tag:  OutboundDNSTag,
				Type: C.TypeDNS,
			},
			{
				Tag:  OutboundDirectTag,
				Type: C.TypeDirect,
			},
			{
				Tag:  OutboundDirectFragmentTag,
				Type: C.TypeDirect,
				DirectOptions: option.DirectOutboundOptions{
					DialerOptions: option.DialerOptions{
						TCPFastOpen: false,
						TLSFragment: &option.TLSFragmentOptions{
							Enabled: true,
							Size:    opt.TLSTricks.FragmentSize,
							Sleep:   opt.TLSTricks.FragmentSleep,
						},
					},
				},
			},
			{
				Tag:  OutboundBypassTag,
				Type: C.TypeDirect,
			},
			{
				Tag:  OutboundBlockTag,
				Type: C.TypeBlock,
			},
		}...,
	)
	if len(directDNSDomains) > 0 {
		// trickDnsDomains := []string{}
		// directDNSDomains = removeDuplicateStr(directDNSDomains)
		// b, _ := batch.New(context.Background(), batch.WithConcurrencyNum[bool](10))
		// for _, d := range directDNSDomains {
		// 	b.Go(d, func() (bool, error) {
		// 		return isBlockedDomain(d), nil
		// 	})
		// }
		// b.Wait()
		// for domain, isBlock := range b.Result() {
		// 	if isBlock.Value {
		// 		trickDnsDomains = append(trickDnsDomains, domain)
		// 	}
		// }

		// trickDomains := strings.Join(trickDnsDomains, ",")
		// trickRule := Rule{Domains: trickDomains, Outbound: OutboundBypassTag}
		// trickDnsRule := trickRule.MakeDNSRule()
		// trickDnsRule.Server = DNSTricksDirectTag
		// options.DNS.Rules = append([]option.DNSRule{{Type: C.RuleTypeDefault, DefaultOptions: trickDnsRule}}, options.DNS.Rules...)

		directDNSDomainskeys := make([]string, 0, len(directDNSDomains))
		for key := range directDNSDomains {
			directDNSDomainskeys = append(directDNSDomainskeys, key)
		}

		domains := strings.Join(directDNSDomainskeys, ",")
		directRule := Rule{Domains: domains, Outbound: OutboundBypassTag}
		dnsRule := directRule.MakeDNSRule()
		dnsRule.Server = DNSDirectTag
		options.DNS.Rules = append([]option.DNSRule{{Type: C.RuleTypeDefault, DefaultOptions: dnsRule}}, options.DNS.Rules...)
	}
	options.Route.Final = OutboundMainProxyTag
	return &options, nil
}

func patchHiddifyWarpFromConfig(out option.Outbound, opt ConfigOptions) option.Outbound {
	if opt.Warp.EnableWarp && opt.Warp.Mode == "proxy_over_warp" {
		if out.DirectOptions.Detour == "" {
			out.DirectOptions.Detour = "Hiddify Warp ✅"
		}
		if out.HTTPOptions.Detour == "" {
			out.HTTPOptions.Detour = "Hiddify Warp ✅"
		}
		if out.Hysteria2Options.Detour == "" {
			out.Hysteria2Options.Detour = "Hiddify Warp ✅"
		}
		if out.HysteriaOptions.Detour == "" {
			out.HysteriaOptions.Detour = "Hiddify Warp ✅"
		}
		if out.SSHOptions.Detour == "" {
			out.SSHOptions.Detour = "Hiddify Warp ✅"
		}
		if out.ShadowTLSOptions.Detour == "" {
			out.ShadowTLSOptions.Detour = "Hiddify Warp ✅"
		}
		if out.ShadowsocksOptions.Detour == "" {
			out.ShadowsocksOptions.Detour = "Hiddify Warp ✅"
		}
		if out.ShadowsocksROptions.Detour == "" {
			out.ShadowsocksROptions.Detour = "Hiddify Warp ✅"
		}
		if out.SocksOptions.Detour == "" {
			out.SocksOptions.Detour = "Hiddify Warp ✅"
		}
		if out.TUICOptions.Detour == "" {
			out.TUICOptions.Detour = "Hiddify Warp ✅"
		}
		if out.TorOptions.Detour == "" {
			out.TorOptions.Detour = "Hiddify Warp ✅"
		}
		if out.TrojanOptions.Detour == "" {
			out.TrojanOptions.Detour = "Hiddify Warp ✅"
		}
		if out.VLESSOptions.Detour == "" {
			out.VLESSOptions.Detour = "Hiddify Warp ✅"
		}
		if out.VMessOptions.Detour == "" {
			out.VMessOptions.Detour = "Hiddify Warp ✅"
		}
		if out.WireGuardOptions.Detour == "" {
			out.WireGuardOptions.Detour = "Hiddify Warp ✅"
		}
	}
	return out
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

func generateRandomString(length int) string {
	// Determine the number of bytes needed
	bytesNeeded := (length*6 + 7) / 8

	// Generate random bytes
	randomBytes := make([]byte, bytesNeeded)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "hiddify"
	}

	// Encode random bytes to base64
	randomString := base64.URLEncoding.EncodeToString(randomBytes)

	// Trim padding characters and return the string
	return randomString[:length]
}
