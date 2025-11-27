package config

import (
	"bytes"
	"encoding/base64"
	json "github.com/goccy/go-json"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"net/url"
	"runtime"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
	badoption "github.com/sagernet/sing/common/json/badoption"
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

func BuildConfigJson(configOpt HiddifyOptions, input option.Options) (string, error) {
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
func BuildConfig(opt HiddifyOptions, input option.Options) (*option.Options, error) {
	fmt.Printf("config options: %++v\n", opt)

	var options option.Options
	if opt.EnableFullConfig {
		options.Inbounds = input.Inbounds
		options.DNS = input.DNS
		options.Route = input.Route
	}

	setClashAPI(&options, &opt)
	setLog(&options, &opt)
	setInbound(&options, &opt)
	setDns(&options, &opt)
	setRoutingOptions(&options, &opt)
	setFakeDns(&options, &opt)
	err := setOutbounds(&options, &input, &opt)
	if err != nil {
		return nil, err
	}

	return &options, nil
}

func addForceDirect(options *option.Options, opt *HiddifyOptions, directDNSDomains map[string]bool) {
	remoteDNSAddress := opt.RemoteDnsAddress
	if strings.Contains(remoteDNSAddress, "://") {
		remoteDNSAddress = strings.SplitAfter(remoteDNSAddress, "://")[1]
	}
	parsedUrl, err := url.Parse(fmt.Sprintf("https://%s", remoteDNSAddress))
	if err == nil && net.ParseIP(parsedUrl.Host) == nil {
		directDNSDomains[parsedUrl.Host] = true
	}
	if len(directDNSDomains) > 0 {
		directDNSDomainskeys := make([]string, 0, len(directDNSDomains))
		for key := range directDNSDomains {
			directDNSDomainskeys = append(directDNSDomainskeys, key)
		}

		domains := strings.Join(directDNSDomainskeys, ",")
		directRule := Rule{Domains: domains, Outbound: OutboundBypassTag}
		dnsRule := directRule.MakeDNSRule()
		dnsRule.DNSRuleAction.Action = C.RuleActionTypeRoute
		dnsRule.DNSRuleAction.RouteOptions = option.DNSRouteActionOptions{Server: DNSDirectTag}
		options.DNS.Rules = append([]option.DNSRule{{Type: C.RuleTypeDefault, DefaultOptions: dnsRule}}, options.DNS.Rules...)
	}
}

func setOutbounds(options *option.Options, input *option.Options, opt *HiddifyOptions) error {
	directDNSDomains := make(map[string]bool)
	var outbounds []option.Outbound
	var tags []string
	OutboundMainProxyTag = OutboundSelectTag
	// inbound==warp over proxies
	// outbound==proxies over warp
	// Note: WARP detour wiring and fragmentation will be adapted later for v1.13 API
	if opt.Masque.Enable {
		// out, err := GenerateWarpSingbox(opt.Warp.WireguardConfig, opt.Warp.CleanIP, opt.Warp.CleanPort, opt.Warp.FakePackets, opt.Warp.FakePacketSize, opt.Warp.FakePacketDelay, opt.Warp.FakePacketMode)
		// if err != nil {
		// 	return fmt.Errorf("failed to generate warp config: %v", err)
		// }
		// out.Tag = "Hiddify Warp "
		// OutboundMainProxyTag = out.Tag
		// outbounds = append(outbounds, *out)
	}
	for _, out := range input.Outbounds {
		outbound, serverDomain, err := patchOutbound(out, *opt, nil)
		if err != nil {
			return err
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
		default:
			if !strings.Contains(out.Tag, "§hide§") {
				tags = append(tags, out.Tag)
			}
			outbounds = append(outbounds, out)
		}
	}

	urlTest := option.Outbound{
		Type:    C.TypeURLTest,
		Tag:     OutboundURLTestTag,
		Options: option.URLTestOutboundOptions{Outbounds: tags, URL: opt.ConnectionTestUrl, Interval: badoption.Duration(opt.URLTestInterval.Duration()), Tolerance: 1, IdleTimeout: badoption.Duration(opt.URLTestInterval.Duration().Nanoseconds() * 3), InterruptExistConnections: true},
	}
	defaultSelect := urlTest.Tag

	for _, tag := range tags {
		if strings.Contains(tag, "§default§") {
			defaultSelect = "§default§"
		}
	}
	selector := option.Outbound{Type: C.TypeSelector, Tag: OutboundSelectTag, Options: option.SelectorOutboundOptions{Outbounds: append([]string{urlTest.Tag}, tags...), Default: defaultSelect, InterruptExistConnections: true}}

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
				Options: option.DirectOutboundOptions{
					DialerOptions: option.DialerOptions{
						TCPFastOpen: false,
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

	addForceDirect(options, opt, directDNSDomains)
	return nil
}

func setClashAPI(options *option.Options, opt *HiddifyOptions) {
	if opt.EnableClashApi {
		if opt.ClashApiSecret == "" {
			opt.ClashApiSecret = generateRandomString(16)
		}
		options.Experimental = &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: fmt.Sprintf("%s:%d", opt.ClashApiHost, opt.ClashApiPort),
				Secret:             opt.ClashApiSecret,
			},

			CacheFile: &option.CacheFileOptions{
				Enabled: true,
				Path:    "clash.db",
			},
		}
	}
}

func setLog(options *option.Options, opt *HiddifyOptions) {
    options.Log = &option.LogOptions{
        Level:        opt.LogLevel,
        Output:       opt.LogFile,
        Disabled:     false,
        Timestamp:    true,
        DisableColor: true,
    }
}

func setInbound(options *option.Options, opt *HiddifyOptions) {
    var inboundDomainStrategy option.DomainStrategy
    if !opt.ResolveDestination {
        inboundDomainStrategy = option.DomainStrategy(dns.DomainStrategyAsIS)
    } else {
        inboundDomainStrategy = opt.IPv6Mode
    }
    if opt.EnableTunService {
        ActivateTunnelService(*opt)
    } else if opt.EnableTun {
        tunOpt := option.TunInboundOptions{
            Stack:       opt.TUNStack,
            MTU:         opt.MTU,
            AutoRoute:   true,
            StrictRoute: opt.StrictRoute,
            InboundOptions: option.InboundOptions{
                SniffEnabled:             true,
                SniffOverrideDestination: false,
                DomainStrategy:           inboundDomainStrategy,
            },
        }
        options.Inbounds = append(options.Inbounds, option.Inbound{Type: C.TypeTun, Tag: InboundTUNTag, Options: tunOpt})
    }

    var bind string
    if opt.AllowConnectionFromLAN {
        bind = "0.0.0.0"
    } else {
        bind = "127.0.0.1"
    }

    mixedListenAddr := badoption.Addr(netip.MustParseAddr(bind))
    options.Inbounds = append(options.Inbounds, option.Inbound{
        Type: C.TypeMixed,
        Tag:  InboundMixedTag,
        Options: option.HTTPMixedInboundOptions{
            ListenOptions: option.ListenOptions{
                Listen:     &mixedListenAddr,
                ListenPort: opt.MixedPort,
                InboundOptions: option.InboundOptions{
                    SniffEnabled:             true,
                    SniffOverrideDestination: true,
                    DomainStrategy:           inboundDomainStrategy,
                },
            },
            SetSystemProxy: opt.SetSystemProxy,
        },
    })

    dnsListenAddr := badoption.Addr(netip.MustParseAddr(bind))
    options.Inbounds = append(options.Inbounds, option.Inbound{
        Type: C.TypeDirect,
        Tag:  InboundDNSTag,
        Options: option.DirectInboundOptions{
            ListenOptions: option.ListenOptions{
                Listen:     &dnsListenAddr,
                ListenPort: opt.LocalDnsPort,
            },
        },
    })
}

func setDns(options *option.Options, opt *HiddifyOptions) {
    options.DNS = &option.DNSOptions{
        RawDNSOptions: option.RawDNSOptions{
            DNSClientOptions: option.DNSClientOptions{
                IndependentCache: opt.IndependentDNSCache,
            },
            Final: DNSRemoteTag,
            Servers: []option.DNSServerOptions{
                {Tag: DNSRemoteTag, Options: option.LegacyDNSServerOptions{Address: opt.RemoteDnsAddress, AddressResolver: DNSDirectTag, Strategy: opt.RemoteDnsDomainStrategy}},
                {Tag: DNSTricksDirectTag, Options: option.LegacyDNSServerOptions{Address: "https://sky.rethinkdns.com/", Strategy: opt.DirectDnsDomainStrategy, Detour: OutboundDirectTag}},
                {Tag: DNSDirectTag, Options: option.LegacyDNSServerOptions{Address: opt.DirectDnsAddress, AddressResolver: DNSLocalTag, Strategy: opt.DirectDnsDomainStrategy, Detour: OutboundDirectTag}},
                {Tag: DNSLocalTag, Type: C.DNSTypeLocal, Options: option.LocalDNSServerOptions{}},
                {Tag: DNSBlockTag, Options: option.LegacyDNSServerOptions{Address: "rcode://success"}},
            },
        },
    }
}

func setFakeDns(options *option.Options, opt *HiddifyOptions) {
    if opt.EnableFakeDNS {
        inet4Range := netip.MustParsePrefix("198.18.0.0/15")
        inet6Range := netip.MustParsePrefix("fc00::/18")
        inet4Prefix := badoption.Prefix(inet4Range)
        inet6Prefix := badoption.Prefix(inet6Range)
        options.DNS.Servers = append(options.DNS.Servers, option.DNSServerOptions{
            Tag:  DNSFakeTag,
            Type: C.DNSTypeFakeIP,
            Options: option.FakeIPDNSServerOptions{
                Inet4Range: &inet4Prefix,
                Inet6Range: &inet6Prefix,
            },
        })
        options.DNS.Rules = append(options.DNS.Rules, option.DNSRule{
            Type: C.RuleTypeDefault,
            DefaultOptions: option.DefaultDNSRule{
                RawDefaultDNSRule: option.RawDefaultDNSRule{
                    Inbound: badoption.Listable[string]{InboundTUNTag},
                },
                DNSRuleAction: option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSFakeTag, DisableCache: true}},
            },
        })
    }
}

func setRoutingOptions(options *option.Options, opt *HiddifyOptions) {
    dnsRules := []option.DefaultDNSRule{}
    routeRules := []option.Rule{}
    rulesets := []option.RuleSet{}

    if opt.EnableTun && runtime.GOOS == "android" {
        routeRules = append(
            routeRules,
            option.Rule{
                Type: C.RuleTypeDefault,

                DefaultOptions: option.DefaultRule{
                    RawDefaultRule: option.RawDefaultRule{
                        Inbound:     badoption.Listable[string]{InboundTUNTag},
                        PackageName: badoption.Listable[string]{"app.hiddify.com"},
                    },
                    RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundBypassTag}},
                },
            },
        )
        // routeRules = append(
        // 	routeRules,
        // 	option.Rule{
        // 		Type: C.RuleTypeDefault,
        // 		DefaultOptions: option.DefaultRule{
        // 			ProcessName: []string{"Hiddify", "Hiddify.exe", "HiddifyCli", "HiddifyCli.exe"},
        // 			Outbound:    OutboundBypassTag,
        // 		},
        // 	},
        // )
    }
    routeRules = append(routeRules, option.Rule{
        Type: C.RuleTypeDefault,
        DefaultOptions: option.DefaultRule{
            RawDefaultRule: option.RawDefaultRule{
                Inbound: badoption.Listable[string]{InboundDNSTag},
            },
            RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundDNSTag}},
        },
    })
    routeRules = append(routeRules, option.Rule{
        Type: C.RuleTypeDefault,
        DefaultOptions: option.DefaultRule{
            RawDefaultRule: option.RawDefaultRule{
                Port: badoption.Listable[uint16]{53},
            },
            RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundDNSTag}},
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

    if opt.BypassLAN {
        routeRules = append(
            routeRules,
            option.Rule{
                Type: C.RuleTypeDefault,
                DefaultOptions: option.DefaultRule{
                    RawDefaultRule: option.RawDefaultRule{
                        // GeoIP:    []string{"private"},
                        IPIsPrivate: true,
                    },
                    RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundBypassTag}},
                },
            },
        )
    }

    if len(opt.ExtraBypassCIDRs) > 0 {
        routeRules = append(routeRules, option.Rule{
            Type: C.RuleTypeDefault,
            DefaultOptions: option.DefaultRule{
                RawDefaultRule: option.RawDefaultRule{
                    IPCIDR: badoption.Listable[string](opt.ExtraBypassCIDRs),
                },
                RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundBypassTag}},
            },
        })
    }

    for _, rule := range opt.Rules {
        routeRule := rule.MakeRule()
        switch rule.Outbound {
        case "bypass":
            routeRule.RuleAction = option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundBypassTag}}
        case "block":
            routeRule.RuleAction = option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundBlockTag}}
        case "proxy":
            routeRule.RuleAction = option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundMainProxyTag}}
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
            dnsRule.DNSRuleAction = option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSDirectTag}}
        case "block":
            dnsRule.DNSRuleAction = option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSBlockTag, DisableCache: true}}
        case "proxy":
            if opt.EnableFakeDNS {
                fakeDnsRule := dnsRule
                fakeDnsRule.DNSRuleAction = option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSFakeTag}}
                fakeDnsRule.RawDefaultDNSRule.Inbound = badoption.Listable[string]{InboundTUNTag, InboundMixedTag}
                dnsRules = append(dnsRules, fakeDnsRule)
            }
            dnsRule.DNSRuleAction = option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSRemoteTag}}
        }
        dnsRules = append(dnsRules, dnsRule)
    }

    parsedURL, err := url.Parse(opt.ConnectionTestUrl)
    if err == nil {
        var dnsCPttl uint32 = 3000
        dnsRules = append(dnsRules, option.DefaultDNSRule{
            RawDefaultDNSRule: option.RawDefaultDNSRule{
                Domain: badoption.Listable[string]{parsedURL.Host},
            },
            DNSRuleAction: option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSRemoteTag, RewriteTTL: &dnsCPttl}},
        })
    }

    if opt.BlockAds {
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geosite-ads",
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/block/geosite-category-ads-all.srs", opt.GeoRulesBaseURL),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geosite-malware",
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/block/geosite-malware.srs", opt.GeoRulesBaseURL),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geosite-phishing",
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/block/geosite-phishing.srs", opt.GeoRulesBaseURL),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geosite-cryptominers",
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/block/geosite-cryptominers.srs", opt.GeoRulesBaseURL),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geoip-phishing",
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/block/geoip-phishing.srs", opt.GeoRulesBaseURL),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geoip-malware",
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/block/geoip-malware.srs", opt.GeoRulesBaseURL),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })

        routeRules = append(routeRules, option.Rule{
            Type: C.RuleTypeDefault,
            DefaultOptions: option.DefaultRule{
                RawDefaultRule: option.RawDefaultRule{
                    RuleSet: badoption.Listable[string]{
                        "geosite-ads",
                        "geosite-malware",
                        "geosite-phishing",
                        "geosite-cryptominers",
                        "geoip-malware",
                        "geoip-phishing",
                    },
                },
                RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundBlockTag}},
            },
        })
        dnsRules = append(dnsRules, option.DefaultDNSRule{
            RawDefaultDNSRule: option.RawDefaultDNSRule{
                RuleSet: badoption.Listable[string]{
                    "geosite-ads",
                    "geosite-malware",
                    "geosite-phishing",
                    "geosite-cryptominers",
                    "geoip-malware",
                    "geoip-phishing",
                },
            },
            DNSRuleAction: option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSBlockTag}},
        })
        //		DisableCache: true,
    }

    if opt.Region == "ir" {
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geosite-category-ads-ir",
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/block/geosite-category-ads-ir.srs", opt.GeoRulesBaseURL),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })
        routeRules = append(routeRules, option.Rule{
            Type: C.RuleTypeDefault,
            DefaultOptions: option.DefaultRule{
                RawDefaultRule: option.RawDefaultRule{
                    RuleSet: badoption.Listable[string]{"geosite-category-ads-ir"},
                },
                RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundBlockTag}},
            },
        })
    }

    if opt.Region != "other" {
        dnsRules = append(dnsRules, option.DefaultDNSRule{
            RawDefaultDNSRule: option.RawDefaultDNSRule{
                DomainSuffix: badoption.Listable[string]{"." + opt.Region},
            },
            DNSRuleAction: option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSDirectTag}},
        })
        routeRules = append(routeRules, option.Rule{
            Type: C.RuleTypeDefault,
            DefaultOptions: option.DefaultRule{
                RawDefaultRule: option.RawDefaultRule{
                    DomainSuffix: badoption.Listable[string]{"." + opt.Region},
                },
                RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundDirectTag}},
            },
        })
        dnsRules = append(dnsRules, option.DefaultDNSRule{
            RawDefaultDNSRule: option.RawDefaultDNSRule{
                RuleSet: badoption.Listable[string]{
                    "geoip-" + opt.Region,
                    "geosite-" + opt.Region,
                },
            },
            DNSRuleAction: option.DNSRuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.DNSRouteActionOptions{Server: DNSDirectTag}},
        })

        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geoip-" + opt.Region,
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/country/geoip-%s.srs", opt.GeoRulesBaseURL, opt.Region),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })
        rulesets = append(rulesets, option.RuleSet{
            Type:   C.RuleSetTypeRemote,
            Tag:    "geosite-" + opt.Region,
            Format: C.RuleSetFormatBinary,
            RemoteOptions: option.RemoteRuleSet{
                URL:            fmt.Sprintf("%s/rule-set/country/geosite-%s.srs", opt.GeoRulesBaseURL, opt.Region),
                UpdateInterval: badoption.Duration(5 * time.Hour * 24),
            },
        })

        routeRules = append(routeRules, option.Rule{
            Type: C.RuleTypeDefault,
            DefaultOptions: option.DefaultRule{
                RawDefaultRule: option.RawDefaultRule{
                    RuleSet: badoption.Listable[string]{
                        "geoip-" + opt.Region,
                        "geosite-" + opt.Region,
                    },
                },
                RuleAction: option.RuleAction{Action: C.RuleActionTypeRoute, RouteOptions: option.RouteActionOptions{Outbound: OutboundDirectTag}},
            },
        })

    }
    options.Route = &option.RouteOptions{
        Rules:               routeRules,
        Final:               OutboundMainProxyTag,
        AutoDetectInterface: true,
        OverrideAndroidVPN:  true,
        RuleSet:             rulesets,
        // GeoIP: &option.GeoIPOptions{
        // 	Path: opt.GeoIPPath,
        // },
        // Geosite: &option.GeositeOptions{
        // 	Path: opt.GeoSitePath,
        // },
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
}

func patchHiddifyWarpFromConfig(out option.Outbound, opt HiddifyOptions) option.Outbound {
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
