package config

import (
	context "context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"net/url"
	"strings"
	sync "sync"
	"time"

	"github.com/hiddify/hiddify-core/v2/hutils"
	mDNS "github.com/miekg/dns"
	C "github.com/sagernet/sing-box/constant"
	sdns "github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
	"github.com/sagernet/wireguard-go/hiddify"
)

const (
	DNSRemoteTag         = "dns-remote"
	DNSRemoteTagFallback = "dns-remote-fallback"
	DNSLocalTag          = "dns-local"
	DNSStaticTag         = "dns-static"
	DNSDirectTag         = "dns-direct"
	DNSRemoteNoWarpTag   = "dns-remote-no-warp"
	// DNSBlockTag        = "dns-block"
	DNSFakeTag         = "dns-fake"
	DNSTricksDirectTag = "dns-trick-direct"
	// DNSMultiDirectTag  = "dns-multi-direct"
	// DNSMultiRemoteTag  = "dns-multi-remote"
	DNSMultiDirectTag = "dns-direct"
	DNSMultiRemoteTag = "dns-remote"

	OutboundDirectTag = "direct §hide§"
	OutboundBypassTag = "bypass §hide§"
	// OutboundBlockTag          = "block §hide§"
	OutboundSelectTag         = "select"
	OutboundURLTestTag        = "lowest"
	OutboundRoundRobinTag     = "balance"
	OutboundDNSTag            = "dns-out §hide§"
	OutboundDirectFragmentTag = "direct-fragment §hide§"

	WARPConfigTag = "🔒 WARP"

	InboundTUNTag    = "tun-in"
	InboundMixedTag  = "mixed-in"
	InboundTProxy    = "tproxy-in"
	InboundRedirect  = "redirect-in"
	InboundDirectTag = "dns-in"
)

var (
	OutboundMainDetour       = OutboundSelectTag
	OutboundWARPConfigDetour = OutboundDirectFragmentTag
	PredefinedOutboundTags   = []string{OutboundDirectTag, OutboundBypassTag, OutboundSelectTag, OutboundURLTestTag, OutboundDNSTag, OutboundDirectFragmentTag, WARPConfigTag}
)

// TODO include selectors
func BuildConfig(ctx context.Context, hopts *HiddifyOptions, inputOpt *ReadOptions) (*option.Options, error) {

	input, err := ReadSingOptions(ctx, inputOpt)
	if err != nil {
		return nil, err
	}

	var options option.Options
	if hopts.EnableFullConfig {
		options.Inbounds = input.Inbounds
		options.DNS = input.DNS
		options.Route = input.Route
	}

	setExperimental(&options, hopts)

	setLog(&options, hopts)
	setInbound(&options, hopts)
	staticIPs := make(map[string][]string)
	// staticIPs["api.cloudflareclient.com"] = []string{"104.16.192.82", "2606:4700::6810:1854", getRandomWarpIP()}
	// setNTP(&options)
	if err := setOutbounds(&options, input, hopts, &staticIPs); err != nil {
		return nil, err
	}
	if err := setDns(&options, hopts, &staticIPs); err != nil {
		return nil, err
	}

	if err := setRoutingOptions(&options, hopts); err != nil {
		return nil, err
	}

	return &options, nil
}

func setNTP(options *option.Options) {
	options.NTP = &option.NTPOptions{
		Enabled:       true,
		ServerOptions: option.ServerOptions{ServerPort: 123, Server: "time.apple.com"},
		Interval:      badoption.Duration(12 * time.Hour),
		DialerOptions: option.DialerOptions{
			Detour: OutboundDirectTag,
		},
	}
}

func getHostnameIfNotIP(inp string) (string, error) {
	if inp == "" {
		return "", fmt.Errorf("empty hostname: %s", inp)
	}
	if net.ParseIP(strings.Trim(inp, "[]")) == nil {
		inp2 := inp
		if !strings.Contains(inp, "://") {
			inp2 = "http://" + inp
		}
		u, err := url.Parse(inp2)
		if err != nil {
			return inp, nil
		}
		if net.ParseIP(strings.Trim(u.Host, "[]")) == nil {
			return u.Host, nil
		}
	}
	return "", fmt.Errorf("not a hostname: %s", inp)
}

func setOutbounds(options *option.Options, input *option.Options, opt *HiddifyOptions, staticIPs *map[string][]string) error {
	var outbounds []option.Outbound
	var endpoints []option.Endpoint
	var tags []string
	// OutboundMainProxyTag = OutboundSelectTag
	// inbound==warp over proxies
	// outbound==proxies over warp
	OutboundMainDetour = OutboundSelectTag
	OutboundWARPConfigDetour = OutboundDirectFragmentTag
	hasPsiphon := false
	for _, out := range input.Outbounds {

		if contains(PredefinedOutboundTags, out.Tag) {
			continue
		}
		outbound, err := patchOutbound(out, *opt, staticIPs)
		if err != nil {
			return err
		}
		out = *outbound

		switch out.Type {
		case C.TypeBlock, C.TypeDNS:
			continue
		case C.TypeSelector, C.TypeURLTest:
			continue
		case C.TypeCustom:
			continue
		default:

			if contains([]string{"direct", "bypass", "block"}, out.Tag) {
				continue
			}
			if out.Type == C.TypePsiphon {
				if hasPsiphon {
					continue
				}
				hasPsiphon = true
			}
			if !strings.Contains(out.Tag, "§hide§") {
				tags = append(tags, out.Tag)
			}
			// OutboundWARPConfigDetour = OutboundSelectTag
			out = *patchHiddifyWarpFromConfig(&out, *opt)
			outbounds = append(outbounds, out)
		}
	}

	if opt.Warp.EnableWarp {
		// wg := getOrGenerateWarpLocallyIfNeeded(&opt.Warp)

		// out, err := GenerateWarpSingbox(wg, opt.Warp.CleanIP, opt.Warp.CleanPort, &option.WireGuardHiddify{
		// 	FakePackets:      opt.Warp.FakePackets,
		// 	FakePacketsSize:  opt.Warp.FakePacketSize,
		// 	FakePacketsDelay: opt.Warp.FakePacketDelay,
		// 	FakePacketsMode:  opt.Warp.FakePacketMode,
		// })
		out, err := GenerateWarpSingboxNew("p1", &hiddify.NoiseOptions{})
		if err != nil {
			return fmt.Errorf("failed to generate warp config: %v", err)
		}
		out.Tag = WARPConfigTag
		if opts, ok := out.Options.(*option.WireGuardWARPEndpointOptions); ok {
			if opt.Warp.Mode == "warp_over_proxy" {
				opts.Detour = OutboundSelectTag
				opts.MTU = 1280
			} else {
				opts.Detour = OutboundDirectTag
				opt.MTU = max(opt.MTU, 1340)
			}

		}

		OutboundMainDetour = WARPConfigTag
		// patchWarp(out, opt, true, nil)
		out, err = patchEndpoint(out, *opt, staticIPs)
		if err != nil {
			return err
		}
		endpoints = append(endpoints, *out)
	}
	for _, end := range input.Endpoints {
		if contains(PredefinedOutboundTags, end.Tag) {
			continue
		}
		if opt.Warp.EnableWarp {
			if end.Type == C.TypeWARP {
				if opts, ok := end.Options.(*option.WireGuardWARPEndpointOptions); ok {
					if opts.UniqueIdentifier == "p1" {
						continue
					}
					if opt.Warp.EnableWarp && opt.Warp.Mode == "warp_over_proxy" {
						opt.MTU = max(opt.MTU, 1340)
					}
				}
			}
			if end.Type == C.TypeWireGuard {
				if opts, ok := end.Options.(*option.WireGuardEndpointOptions); ok {
					if opts.PrivateKey == opt.Warp.WireguardConfig.PrivateKey {
						continue
					}
					if opt.Warp.EnableWarp && opt.Warp.Mode == "warp_over_proxy" {
						opt.MTU = max(opt.MTU, 1340)
					}
				}
			}
		}

		out, err := patchEndpoint(&end, *opt, staticIPs)
		if err != nil {
			return err
		}

		if !strings.Contains(out.Tag, "§hide§") {
			tags = append(tags, out.Tag)
		}

		endpoints = append(endpoints, *out)
	}
	if len(opt.ConnectionTestUrls) == 0 {
		opt.ConnectionTestUrls = []string{opt.ConnectionTestUrl, "https://www.google.com/generate_204", "http://captive.apple.com/generate_204", "https://cp.cloudflare.com"}
		if isBlockedConnectionTestUrl(opt.ConnectionTestUrl) {
			opt.ConnectionTestUrls = []string{opt.ConnectionTestUrl}
		}
	}
	// urlTest := option.Outbound{
	// 	Type: C.TypeURLTest,
	// 	Tag:  OutboundURLTestTag,
	// 	Options: &option.URLTestOutboundOptions{
	// 		Outbounds: tags,
	// 		URL:       opt.ConnectionTestUrl,
	// 		URLs:      opt.ConnectionTestUrls,
	// 		Interval:  badoption.Duration(opt.URLTestInterval.Duration()),
	// 		// IdleTimeout: badoption.Duration(opt.URLTestIdleTimeout.Duration()),
	// 		Tolerance:                 1,
	// 		IdleTimeout:               badoption.Duration(opt.URLTestInterval.Duration().Nanoseconds() * 3),
	// 		InterruptExistConnections: true,
	// 	},
	// }
	urlTest := option.Outbound{
		Type: C.TypeBalancer,
		Tag:  OutboundURLTestTag,
		Options: &option.BalancerOutboundOptions{
			Outbounds:            tags,
			Strategy:             "lowest-delay",
			DelayAcceptableRatio: 2,
			// URL:       opt.ConnectionTestUrl,
			// URLs:      opt.ConnectionTestUrls,
			// Interval:  badoption.Duration(opt.URLTestInterval.Duration()),
			// IdleTimeout: badoption.Duration(opt.URLTestIdleTimeout.Duration()),
			Tolerance: 1,
			// IdleTimeout:               badoption.Duration(opt.URLTestInterval.Duration().Nanoseconds() * 3),
			InterruptExistConnections: true,
		},
	}

	balancer := option.Outbound{
		Type: C.TypeBalancer,
		Tag:  OutboundRoundRobinTag,
		Options: &option.BalancerOutboundOptions{
			Outbounds:            tags,
			Strategy:             opt.BalancerStrategy,
			DelayAcceptableRatio: 2,
			// URL:       opt.ConnectionTestUrl,
			// URLs:      opt.ConnectionTestUrls,
			// Interval:  badoption.Duration(opt.URLTestInterval.Duration()),
			// IdleTimeout: badoption.Duration(opt.URLTestIdleTimeout.Duration()),
			Tolerance: 1,
			// IdleTimeout:               badoption.Duration(opt.URLTestInterval.Duration().Nanoseconds() * 3),
			InterruptExistConnections: true,
		},
	}
	defaultSelect := tags[0]

	for _, tag := range tags {
		if strings.Contains(tag, "§default§") {
			defaultSelect = "§default§"
		}
	}

	selectorTags := tags
	if len(tags) > 1 {
		if OutboundMainDetour == WARPConfigTag {
			outbounds = append([]option.Outbound{urlTest}, outbounds...)
			selectorTags = append([]string{urlTest.Tag}, selectorTags...)
			defaultSelect = urlTest.Tag
		} else {
			outbounds = append([]option.Outbound{balancer, urlTest}, outbounds...)
			selectorTags = append([]string{urlTest.Tag, balancer.Tag}, selectorTags...)
			defaultSelect = balancer.Tag

		}
	}
	selector := option.Outbound{
		Type: C.TypeSelector,
		Tag:  OutboundSelectTag,
		Options: &option.SelectorOutboundOptions{
			Outbounds:                 selectorTags,
			Default:                   defaultSelect,
			InterruptExistConnections: true,
		},
	}
	outbounds = append([]option.Outbound{selector}, outbounds...)

	options.Endpoints = endpoints
	options.Outbounds = append(
		outbounds,
		[]option.Outbound{
			{
				Tag:     OutboundDirectTag,
				Type:    C.TypeDirect,
				Options: &option.DirectOutboundOptions{},
			},
			{
				Tag:  OutboundDirectFragmentTag,
				Type: C.TypeDirect,
				Options: &option.DirectOutboundOptions{
					DialerOptions: option.DialerOptions{
						TCPFastOpen: false,

						TLSFragment: option.TLSFragmentOptions{
							Enabled: true,
							Size:    opt.TLSTricks.FragmentSize,
							Sleep:   opt.TLSTricks.FragmentSleep,
						},
					},
				},
			},
		}...,
	)

	return nil
}

func isBlockedConnectionTestUrl(d string) bool {
	u, err := url.Parse(d)
	if err != nil {
		return false
	}
	return isBlockedDomain(u.Host)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func setExperimental(options *option.Options, hopt *HiddifyOptions) {
	if len(hopt.ConnectionTestUrls) == 0 {
		hopt.ConnectionTestUrls = []string{hopt.ConnectionTestUrl, "http://captive.apple.com/generate_204", "https://cp.cloudflare.com", "https://google.com/generate_204"}
		if isBlockedConnectionTestUrl(hopt.ConnectionTestUrl) {
			hopt.ConnectionTestUrls = []string{hopt.ConnectionTestUrl}
		}
	}
	if hopt.EnableClashApi {
		if hopt.ClashApiSecret == "" {
			hopt.ClashApiSecret = generateRandomString(16)
		}
		options.Experimental = &option.ExperimentalOptions{
			UnifiedDelay: &option.UnifiedDelayOptions{
				Enabled: true,
			},
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", hopt.ClashApiPort),
				Secret:             hopt.ClashApiSecret,
			},

			CacheFile: &option.CacheFileOptions{
				Enabled:         true,
				StoreWARPConfig: true,
				Path:            "data/clash.db",
			},

			Monitoring: &option.MonitoringOptions{
				URLs:           hopt.ConnectionTestUrls,
				Interval:       badoption.Duration(hopt.URLTestInterval.Duration()),
				DebounceWindow: badoption.Duration(time.Millisecond * 500),
				IdleTimeout:    badoption.Duration(hopt.URLTestInterval.Duration().Nanoseconds() * 3),
			},
		}
	}
}

func setLog(options *option.Options, opt *HiddifyOptions) {
	options.Log = &option.LogOptions{
		Level:        opt.LogLevel,
		Output:       opt.LogFile,
		Disabled:     false,
		Timestamp:    false,
		DisableColor: true,
	}
}
func isIPv6Supported() bool {
	if C.IsIos || C.IsDarwin {
		return true
	}
	_, err := net.ResolveIPAddr("ip6", "::1")
	return err == nil
}
func setInbound(options *option.Options, hopt *HiddifyOptions) {
	// var inboundDomainStrategy option.DomainStrategy
	// if !opt.ResolveDestination {
	// 	inboundDomainStrategy = option.DomainStrategy(dns.DomainStrategyAsIS)
	// } else {
	// 	inboundDomainStrategy = opt.IPv6Mode
	// }
	ipv6Enable := isIPv6Supported()
	if hopt.EnableTun {

		opts := option.TunInboundOptions{
			Stack:       hopt.TUNStack,
			MTU:         hopt.MTU,
			AutoRoute:   true,
			StrictRoute: hopt.StrictRoute,

			// EndpointIndependentNat: true,
			// GSO:                    runtime.GOOS != "windows",

		}
		tunInbound := option.Inbound{
			Type: C.TypeTun,
			Tag:  InboundTUNTag,

			Options: &opts,
		}
		// switch hopt.IPv6Mode {
		// case option.DomainStrategy(dns.DomainStrategyUseIPv4):
		// 	opts.Address = []netip.Prefix{
		// 		netip.MustParsePrefix("172.19.0.1/28"),
		// 	}
		// case option.DomainStrategy(dns.DomainStrategyUseIPv6):
		// 	opts.Address = []netip.Prefix{
		// 		netip.MustParsePrefix("fdfe:dcba:9876::1/126"),
		// 	}
		// default:

		// }
		opts.Address = []netip.Prefix{netip.MustParsePrefix("172.19.0.1/28")}
		if ipv6Enable {
			opts.Address = append(opts.Address, netip.MustParsePrefix("fdfe:dcba:9876::1/126"))
		}

		options.Inbounds = append(options.Inbounds, tunInbound)

	}

	binds := []string{}

	if hopt.AllowConnectionFromLAN {
		if ipv6Enable {
			binds = append(binds, "::")
		} else {
			binds = append(binds, "0.0.0.0")
		}
	} else {
		if ipv6Enable {
			binds = append(binds, "::1")
		}
		binds = append(binds, "127.0.0.1")
	}

	for _, bind := range binds {
		addr := badoption.Addr(netip.MustParseAddr(bind))

		options.Inbounds = append(
			options.Inbounds,
			option.Inbound{
				Type: C.TypeMixed,
				Tag:  InboundMixedTag + bind,
				Options: &option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     &addr,
						ListenPort: hopt.MixedPort,
						// InboundOptions: option.InboundOptions{
						// 	SniffEnabled:             true,
						// 	SniffOverrideDestination: true,
						// 	DomainStrategy:           inboundDomainStrategy,
						// },
					},
					SetSystemProxy: hopt.SetSystemProxy,
				},
			},
		)
		if C.IsLinux && !C.IsAndroid && hopt.TProxyPort > 0 && hutils.IsAdmin() {
			options.Inbounds = append(
				options.Inbounds,
				option.Inbound{
					Type: C.TypeTProxy,
					Tag:  InboundTProxy + bind,
					Options: &option.TProxyInboundOptions{
						ListenOptions: option.ListenOptions{
							Listen:     &addr,
							ListenPort: hopt.TProxyPort,
						},
					},
				},
			)
		}
		if (C.IsLinux || C.IsDarwin) && !C.IsAndroid && hopt.RedirectPort > 0 {
			options.Inbounds = append(
				options.Inbounds,
				option.Inbound{
					Type: C.TypeRedirect,
					Tag:  InboundRedirect + bind,
					Options: &option.RedirectInboundOptions{
						ListenOptions: option.ListenOptions{
							Listen:     &addr,
							ListenPort: hopt.RedirectPort,
						},
					},
				},
			)
		}
		if hopt.DirectPort > 0 {
			options.Inbounds = append(
				options.Inbounds,
				option.Inbound{
					Type: C.TypeDirect,
					Tag:  InboundDirectTag + bind,
					Options: &option.DirectInboundOptions{
						ListenOptions: option.ListenOptions{
							Listen:     &addr,
							ListenPort: hopt.DirectPort,
						},
					},
				},
			)
		}
	}
}

func setRoutingOptions(options *option.Options, hopt *HiddifyOptions) error {
	dnsRules := []option.DefaultDNSRule{}
	routeRules := []option.Rule{}
	rulesets := []option.RuleSet{}

	// if opt.EnableTun && runtime.GOOS == "android" {
	// 	// routeRules = append(
	// 	// 	routeRules,
	// 	// 	option.Rule{
	// 	// 		Type: C.RuleTypeDefault,

	// 	// 		DefaultOptions: option.DefaultRule{
	// 	// 			Inbound:     []string{InboundTUNTag},
	// 	// 			PackageName: []string{"app.hiddify.com"},
	// 	// 			Outbound:    OutboundBypassTag,
	// 	// 		},
	// 	// 	},
	// 	// )
	// }
	// if opt.EnableTun && runtime.GOOS == "windows" {
	// 	// routeRules = append(
	// 	// 	routeRules,
	// 	// 	option.Rule{
	// 	// 		Type: C.RuleTypeDefault,
	// 	// 		DefaultOptions: option.DefaultRule{
	// 	// 			ProcessName: []string{"Hiddify", "Hiddify.exe", "HiddifyCli", "HiddifyCli.exe"},
	// 	// 			Outbound:    OutboundBypassTag,
	// 	// 		},
	// 	// 	},
	// 	// )
	// }

	// dnsRules = append(dnsRules, option.DefaultDNSRule{
	// 	RawDefaultDNSRule: option.RawDefaultDNSRule{},
	// 	DNSRuleAction: option.DNSRuleAction{
	// 		Action: C.RuleActionTypeRoute,
	// 		RouteOptions: option.DNSRouteActionOptions{
	// 			Server:         DNSStaticTag,
	// 			BypassIfFailed: false,
	// 		},
	// 	},
	// },
	// )
	forceDirectRules, err := addForceDirect(options, hopt)
	if err != nil {
		return err
	}

	dnsRules = append(dnsRules, forceDirectRules...)

	routeRules = append(routeRules, option.Rule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultRule{
			RuleAction: option.RuleAction{
				Action: C.RuleActionTypeSniff,
			},
		},
	})
	routeRules = append(routeRules, option.Rule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultRule{
			RawDefaultRule: option.RawDefaultRule{
				Protocol: []string{C.ProtocolDNS},
			},
			RuleAction: option.RuleAction{
				Action: C.RuleActionTypeHijackDNS,
			},
		},
	})

	routeRules = append(routeRules, option.Rule{
		Type: C.RuleTypeDefault,

		DefaultOptions: option.DefaultRule{
			RawDefaultRule: option.RawDefaultRule{
				IPCIDR: []string{
					"10.10.34.0/24",
					"2001:4188:2:600:10:10:34:0/120",
				},
			},
			RuleAction: option.RuleAction{
				Action: C.RuleActionTypeRoute,
				RouteOptions: option.RouteActionOptions{
					Outbound: OutboundMainDetour,
				},
			},
		},
	})
	// {
	// 	Type: C.RuleTypeDefault,
	// 	DefaultOptions: option.DefaultRule{
	// 		ClashMode: "Direct",
	// 		Outbound:  OutboundDirectTag,
	// 	},
	// },
	// {
	// 	Type: C.RuleTypeDefault,
	// 	DefaultOptions: option.DefaultRule{
	// 		ClashMode: "Global",
	// 		Outbound:  OutboundMainProxyTag,
	// 	},
	// },	}

	if hopt.BypassLAN {
		routeRules = append(
			routeRules,
			option.Rule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					RawDefaultRule: option.RawDefaultRule{
						IPIsPrivate: true,
					},
					RuleAction: option.RuleAction{
						Action: C.RuleActionTypeRoute,
						RouteOptions: option.RouteActionOptions{
							Outbound: OutboundDirectTag,
						},
					},
				},
			},
		)
	}

	// for _, rule := range opt.Rules {
	// 	routeRule := rule.MakeRule()
	// 	switch rule.Outbound {
	// 	case "bypass":
	// 		routeRule.Outbound = OutboundBypassTag
	// 	case "block":
	// 		routeRule.Outbound = OutboundBlockTag
	// 	case "proxy":
	// 		routeRule.Outbound = OutboundMainProxyTag
	// 	}

	// 	if routeRule.IsValid() {
	// 		routeRules = append(
	// 			routeRules,
	// 			option.Rule{
	// 				Type:           C.RuleTypeDefault,
	// 				DefaultOptions: routeRule,
	// 			},
	// 		)
	// 	}

	// 	dnsRule := rule.MakeDNSRule()
	// 	switch rule.Outbound {
	// 	case "bypass":
	// 		dnsRule.Server = DNSDirectTag
	// 	case "block":
	// 		dnsRule.Server = DNSBlockTag
	// 		dnsRule.DisableCache = true
	// 	case "proxy":
	// 		if opt.EnableFakeDNS {
	// 			fakeDnsRule := dnsRule
	// 			fakeDnsRule.Server = DNSFakeTag
	// 			fakeDnsRule.Inbound = []string{InboundTUNTag, InboundMixedTag}
	// 			dnsRules = append(dnsRules, fakeDnsRule)
	// 		}
	// 		dnsRule.Server = DNSRemoteTag
	// 	}
	// 	dnsRules = append(dnsRules, dnsRule)
	// }
	forceDirectRoute := make([]string, 0)
	if options.NTP != nil && options.NTP.Enabled {
		forceDirectRoute = append(forceDirectRoute, options.NTP.Server)
	}

	// parsedURL, err := url.Parse(opt.ConnectionTestUrl)
	// if err == nil {
	// 	dnsRules = append(dnsRules, option.DefaultDNSRule{
	// 		Domain:       []string{parsedURL.Host},
	// 		Server:       DNSRemoteTag,
	// 		RewriteTTL:   &dnsCPttl,
	// 		DisableCache: false,
	// 	})
	// }

	if len(forceDirectRoute) > 0 {

		dnsRules = append(dnsRules, option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{
				Domain: forceDirectRoute,
			},
			DNSRuleAction: option.DNSRuleAction{
				Action: C.RuleActionTypeRoute,
				RouteOptions: option.DNSRouteActionOptions{
					Server:         DNSMultiDirectTag,
					Strategy:       hopt.DirectDnsDomainStrategy,
					RewriteTTL:     &DEFAULT_DNS_TTL,
					DisableCache:   false,
					BypassIfFailed: false,
				},
			},
		})
		routeRules = append(routeRules, option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RawDefaultRule: option.RawDefaultRule{
					Domain: forceDirectRoute,
				},
				RuleAction: option.RuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.RouteActionOptions{
						Outbound: OutboundDirectTag,
					},
				},
			},
		})
	}
	rejectRCode := (option.DNSRCode(sdns.RcodeRefused))
	rejectDnsAction := option.DNSRuleAction{
		Action: C.RuleActionTypePredefined,
		PredefinedOptions: option.DNSRouteActionPredefined{
			Rcode: &rejectRCode,
		},
	}
	if hopt.BlockAds {
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-ads",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-category-ads-all.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-malware",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-malware.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-phishing",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-phishing.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-cryptominers",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-cryptominers.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geoip-phishing",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geoip-phishing.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geoip-malware",
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geoip-malware.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})

		routeRules = append(routeRules, option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RawDefaultRule: option.RawDefaultRule{
					RuleSet: []string{
						"geosite-ads",
						"geosite-malware",
						"geosite-phishing",
						"geosite-cryptominers",
						"geoip-malware",
						"geoip-phishing",
					},
				},
				RuleAction: option.RuleAction{
					Action: C.RuleActionTypeReject,
					RejectOptions: option.RejectActionOptions{
						Method: C.RuleActionRejectMethodDefault,
					},
				},
			},
		})
		dnsRules = append(dnsRules, option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{

				RuleSet: []string{
					"geosite-ads",
					"geosite-malware",
					"geosite-phishing",
					"geosite-cryptominers",
				},
			},
			DNSRuleAction: rejectDnsAction,
		})
	}
	if hopt.Region != "other" {
		dnsRules = append(dnsRules, option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{
				DomainSuffix: []string{"." + hopt.Region},
			},
			DNSRuleAction: option.DNSRuleAction{
				Action: C.RuleActionTypeRoute,
				RouteOptions: option.DNSRouteActionOptions{
					Server:         DNSMultiDirectTag,
					Strategy:       hopt.DirectDnsDomainStrategy,
					RewriteTTL:     &DEFAULT_DNS_TTL,
					BypassIfFailed: false,
				},
			},
		})
		routeRules = append(routeRules, option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RawDefaultRule: option.RawDefaultRule{
					DomainSuffix: []string{"." + hopt.Region},
				},
				RuleAction: option.RuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.RouteActionOptions{
						Outbound: OutboundDirectTag,
					},
				},
			},
		})

		dnsRules = append(dnsRules, option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{

				RuleSet: []string{
					"geosite-" + hopt.Region,
				},
			},
			DNSRuleAction: option.DNSRuleAction{
				Action: C.RuleActionTypeRoute,
				RouteOptions: option.DNSRouteActionOptions{
					Server:         DNSMultiDirectTag,
					Strategy:       hopt.DirectDnsDomainStrategy,
					RewriteTTL:     &DEFAULT_DNS_TTL,
					BypassIfFailed: false,
				},
			},
		})

		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geoip-" + hopt.Region,
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/country/geoip-" + hopt.Region + ".srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    "geosite-" + hopt.Region,
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/country/geosite-" + hopt.Region + ".srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})

		routeRules = append(routeRules, option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RawDefaultRule: option.RawDefaultRule{
					RuleSet: []string{
						"geoip-" + hopt.Region,
						"geosite-" + hopt.Region,
					},
				},
				RuleAction: option.RuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.RouteActionOptions{
						Outbound: OutboundDirectTag,
					},
				},
			},
		})
	}
	if hopt.RouteOptions.BlockQuic {
		routeRules = append(routeRules, option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RawDefaultRule: option.RawDefaultRule{
					Protocol: []string{C.ProtocolQUIC},
				},
				RuleAction: option.RuleAction{
					Action: C.RuleActionTypeReject,
					RejectOptions: option.RejectActionOptions{
						Method: C.RuleActionRejectMethodDefault,
					},
				},
			},
		})
	}
	options.Route = &option.RouteOptions{
		Rules:               routeRules,
		Final:               OutboundMainDetour,
		AutoDetectInterface: (!C.IsAndroid && !C.IsIos) && (hopt.EnableTun || hopt.EnableTunService),
		DefaultDomainResolver: &option.DomainResolveOptions{
			Server:   DNSMultiDirectTag,
			Strategy: hopt.DirectDnsDomainStrategy,
		},
		// OverrideAndroidVPN: hopt.EnableTun && C.IsAndroid,
		RuleSet:     rulesets,
		FindProcess: false,
		// GeoIP: &option.GeoIPOptions{
		// 	Path: opt.GeoIPPath,
		// },
		// Geosite: &option.GeositeOptions{
		// 	Path: opt.GeoSitePath,
		// },
	}
	// if opt.EnableDNSRouting {
	if hopt.EnableFakeDNS {
		// inbounds := []string{InboundTUNTag}
		// for _, inp := range options.Inbounds {
		// 	if strings.Contains(inp.Tag, InboundDirectTag) || strings.Contains(inp.Tag, InboundRedirect) || strings.Contains(inp.Tag, InboundTProxy) {
		// 		inbounds = append(inbounds, inp.Tag)
		// 	}
		// }
		dnsRules = append(
			dnsRules,
			option.DefaultDNSRule{
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					// Inbound: inbounds,
					QueryType: badoption.Listable[option.DNSQueryType]{
						option.DNSQueryType(mDNS.StringToType["A"]),
						option.DNSQueryType(mDNS.StringToType["AAAA"]),
					},
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server:         DNSFakeTag,
						Strategy:       hopt.RemoteDnsDomainStrategy,
						RewriteTTL:     &DEFAULT_DNS_TTL,
						DisableCache:   true,
						BypassIfFailed: false,
					},
				},
			})

	}

	dnsRules = append(dnsRules, option.DefaultDNSRule{
		RawDefaultDNSRule: option.RawDefaultDNSRule{},
		DNSRuleAction: option.DNSRuleAction{
			Action: C.RuleActionTypeRoute,
			RouteOptions: option.DNSRouteActionOptions{
				Server:         DNSMultiRemoteTag,
				Strategy:       hopt.RemoteDnsDomainStrategy,
				RewriteTTL:     &DEFAULT_DNS_TTL,
				BypassIfFailed: false,
			},
		},
	},
	)
	// dnsRules = append(dnsRules, option.DefaultDNSRule{
	// 	RawDefaultDNSRule: option.RawDefaultDNSRule{},
	// 	DNSRuleAction: option.DNSRuleAction{
	// 		Action: C.RuleActionTypeRoute,
	// 		RouteOptions: option.DNSRouteActionOptions{
	// 			Server:         DNSRemoteTagFallback,
	// 			Strategy:       hopt.RemoteDnsDomainStrategy,
	// 			RewriteTTL:     &DEFAULT_DNS_TTL,
	// 			BypassIfFailed: false,
	// 		},
	// 	},
	// },
	// )

	// dnsRules = append(dnsRules, option.DefaultDNSRule{

	// 	RawDefaultDNSRule: option.RawDefaultDNSRule{},
	// 	DNSRuleAction: option.DNSRuleAction{
	// 		Action: C.RuleActionTypeRoute,
	// 		RouteOptions: option.DNSRouteActionOptions{
	// 			Server:         DNSTricksDirectTag,
	// 			BypassIfFailed: false,
	// 		},
	// 	},
	// },
	// )
	// dnsRules = append(dnsRules, option.DefaultDNSRule{
	// 	RawDefaultDNSRule: option.RawDefaultDNSRule{},
	// 	DNSRuleAction: option.DNSRuleAction{
	// 		Action: C.RuleActionTypeRoute,
	// 		RouteOptions: option.DNSRouteActionOptions{
	// 			Server:         DNSDirectTag,
	// 			BypassIfFailed: false,
	// 		},
	// 	},
	// },
	// )
	// dnsRules = append(dnsRules, option.DefaultDNSRule{
	// 	RawDefaultDNSRule: option.RawDefaultDNSRule{},
	// 	DNSRuleAction: option.DNSRuleAction{
	// 		Action: C.RuleActionTypeRoute,
	// 		RouteOptions: option.DNSRouteActionOptions{
	// 			Server: DNSLocalTag,
	// 			// BypassIfFailed: false,
	// 		},
	// 	},
	// },
	// )

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
	// }
	return nil
}

func patchHiddifyWarpFromConfig(out *option.Outbound, opt HiddifyOptions) *option.Outbound {
	if out.Type == C.TypePsiphon {
		return out
	}
	if opt.Warp.EnableWarp && opt.Warp.Mode == "proxy_over_warp" {
		if opts, ok := out.Options.(option.DialerOptionsWrapper); ok {
			dialer := opts.TakeDialerOptions()
			dialer.Detour = WARPConfigTag
			opts.ReplaceDialerOptions(dialer)
		}
	}
	return out
}

var (
	ipMaps      = map[string][]string{}
	ipMapsMutex sync.Mutex
)

func getIPs(domains ...string) []string {
	var wg sync.WaitGroup
	resChan := make(chan string, len(domains)*10) // Collect both IPv4 and IPv6
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	for _, d := range domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", domain)
			if err != nil {
				return
			}
			for _, ip := range ips {
				ipStr := ip.String()
				if !isBlockedIP(ipStr) {
					resChan <- ipStr
				}
			}
		}(d)
	}

	go func() {
		wg.Wait()
		close(resChan)
	}()

	var res []string
	for ip := range resChan {
		res = append(res, ip)
	}
	if len(res) == 0 && ipMaps[domains[0]] != nil {
		return ipMaps[domains[0]]
	}
	ipMapsMutex.Lock()
	ipMaps[domains[0]] = res
	ipMapsMutex.Unlock()

	return res
}

func isBlockedDomain(domain string) bool {
	if strings.HasPrefix("full:", domain) {
		return false
	}
	if strings.Contains(domain, "instagram") || strings.Contains(domain, "facebook") || strings.Contains(domain, "telegram") || strings.Contains(domain, "t.me") {
		return true
	}
	ips := getIPs(domain)
	if len(ips) == 0 {
		// fmt.Println(err)
		return true
	}

	// // Print the IP addresses associated with the domain
	// fmt.Printf("IP addresses for %s:\n", domain)
	// for _, ip := range ips {
	// 	if isBlockedIP(ip) {
	// 		return true
	// 	}
	// }
	return false
}

func isBlockedIP(ip string) bool {
	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "2001:4188:2:600:10") {
		return true
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
