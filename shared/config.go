package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"

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
	StrictRoute             bool                  `json:"strict-route"`
	TUNStack                string                `json:"tun-stack"`
	ConnectionTestUrl       string                `json:"connection-test-url"`
	URLTestInterval         option.Duration       `json:"url-test-interval"`
	EnableClashApi          bool                  `json:"enable-clash-api"`
	ClashApiPort            uint16                `json:"clash-api-port"`
	EnableTun               bool                  `json:"enable-tun"`
	SetSystemProxy          bool                  `json:"set-system-proxy"`
	BypassLAN               bool                  `json:"bypass-lan"`
	EnableFakeDNS           bool                  `json:"enable-fake-dns"`
	IndependentDNSCache     bool                  `json:"independent-dns-cache"`
	GeoIPPath               string                `json:"geoip-path"`
	GeoSitePath             string                `json:"geosite-path"`
	Rules                   []Rule                `json:"rules"`
	TLSTricks
}

type TLSTricks struct {
	EnableTLSFragment  bool   `json:"enable-tls-fragment"`
	TLSFragmentSize    string `json:"tls-fragment-size"`
	TLSFragmentSleep   string `json:"tls-fragment-sleep"`
	EnableMixedSNICase bool   `json:"enable-tls-mixed-sni-case"`
}

func BuildConfigJson(configOpt ConfigOptions, input option.Options) (string, error) {
	options := BuildConfig(configOpt, input)
	var buffer bytes.Buffer
	json.NewEncoder(&buffer)
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(options)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// TODO include selectors
func BuildConfig(configOpt ConfigOptions, input option.Options) option.Options {
	if configOpt.ExecuteAsIs {
		return applyOverrides(configOpt, input)
	}

	fmt.Printf("config options: %+v\n", configOpt)

	var options option.Options
	directDNSDomains := []string{}
	dnsRules := []option.DefaultDNSRule{}

	if configOpt.EnableClashApi {
		options.Experimental = &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{
				ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", configOpt.ClashApiPort),
				StoreSelected:      true,
				CacheFile:          "clash.db",
			},
		}
	}

	options.Log = &option.LogOptions{
		Level:        configOpt.LogLevel,
		Output:       "box.log",
		Disabled:     false,
		Timestamp:    true,
		DisableColor: true,
	}

	options.DNS = &option.DNSOptions{
		DNSClientOptions: option.DNSClientOptions{
			IndependentCache: configOpt.IndependentDNSCache,
		},
		Servers: []option.DNSServerOptions{
			{
				Tag:             "dns-remote",
				Address:         configOpt.RemoteDnsAddress,
				AddressResolver: "dns-direct",
				Strategy:        configOpt.RemoteDnsDomainStrategy,
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

	options.Inbounds = append(
		options.Inbounds,
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

	remoteDNSAddress := configOpt.RemoteDnsAddress
	if strings.Contains(remoteDNSAddress, "://") {
		remoteDNSAddress = strings.SplitAfter(remoteDNSAddress, "://")[1]
	}
	parsedUrl, err := url.Parse(fmt.Sprintf("https://%s", remoteDNSAddress))
	if err == nil && net.ParseIP(parsedUrl.Host) == nil {
		directDNSDomains = append(directDNSDomains, fmt.Sprintf("full:%s", parsedUrl.Host))
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
		jsonData, err := out.MarshalJSON()
		if err == nil {
			var obj map[string]interface{}
			err = json.Unmarshal(jsonData, &obj)
			if err == nil {
				if value, ok := obj["server"]; ok {
					server := value.(string)
					if server != "" && net.ParseIP(server) == nil {
						directDNSDomains = append(directDNSDomains, fmt.Sprintf("full:%s", server))
					}
				}
				if value, ok := obj["tls"]; ok {
					tls := value.(map[string]interface{})
					tls["mixedcase_sni"] = configOpt.TLSTricks.EnableMixedSNICase
				}
				modifiedJson, err := json.Marshal(obj)
				if err == nil {
					err = out.UnmarshalJSON(modifiedJson)
					if err != nil {
						fmt.Println("error: ", err)
					}
				}
			}
		}

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
		domains := strings.Join(removeDuplicateStr(directDNSDomains), ",")
		directRule := Rule{Domains: domains, Outbound: "bypass"}
		dnsRule := directRule.MakeDNSRule()
		dnsRule.Server = "dns-direct"
		options.DNS.Rules = append([]option.DNSRule{{Type: C.RuleTypeDefault, DefaultOptions: dnsRule}}, options.DNS.Rules...)
	}

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
