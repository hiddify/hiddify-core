package config

import (
	"net/netip"
	"net/url"
	"strings"
	"time"

	dnscode "github.com/miekg/dns"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/hiddify/ipinfo"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json/badjson"
	"github.com/sagernet/sing/common/json/badoption"
	M "github.com/sagernet/sing/common/metadata"
)

func setDns(options *option.Options, opt *HiddifyOptions, staticIps *map[string][]string) error {
	remote_dns, err := getDNSServerOptions(DNSRemoteTag, opt.RemoteDnsAddress, DNSDirectTag, OutboundMainDetour)
	if err != nil {
		return err
	}
	fallbackAddr := "tcp://8.8.8.8"
	if !strings.Contains(opt.RemoteDnsAddress, "://") {
		fallbackAddr = "tcp://" + opt.RemoteDnsAddress
	} else if strings.HasPrefix(opt.RemoteDnsAddress, "udp://") {
		fallbackAddr = strings.Replace(fallbackAddr, "udp://", "tcp://", 1)
	} else if strings.HasPrefix(opt.RemoteDnsAddress, "tcp://") {
		fallbackAddr = strings.Replace(fallbackAddr, "tcp://", "udp://", 1)
	}
	remote_dns_fallback, err := getDNSServerOptions(DNSRemoteTagFallback, fallbackAddr, DNSDirectTag, OutboundMainDetour)
	if err != nil {
		return err
	}
	remote_no_warp_dns, err := getDNSServerOptions(DNSRemoteNoWarpTag, opt.RemoteDnsAddress, DNSDirectTag, OutboundWARPConfigDetour)
	if err != nil {
		return err
	}
	direct_dns, err := getDNSServerOptions(DNSDirectTag, opt.DirectDnsAddress, DNSLocalTag, OutboundDirectFragmentTag)
	if err != nil {
		return err
	}
	trick_dns, err := getDNSServerOptions(DNSTricksDirectTag, "https://dns.cloudflare.com/dns-query", DNSDirectTag, OutboundDirectFragmentTag)
	if err != nil {
		return err
	}
	local_dns, err := getDNSServerOptions(DNSLocalTag, "local", "", "")
	if err != nil {
		return err
	}
	static_dns, err := getStaticDNSServerOptions(DNSStaticTag, staticIps)
	if err != nil {
		return err
	}
	// block_dns, err := getDNSServerOptions(DNSBlockTag, "rcode://name_error", "", "")
	// if err != nil {
	// 	return err
	// }

	dnsOptions := option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			DNSClientOptions: option.DNSClientOptions{
				IndependentCache: opt.IndependentDNSCache && !C.IsIos,
			},
			Final: DNSRemoteTag,

			Servers: []option.DNSServerOptions{
				*static_dns,
				*remote_dns,
				*remote_dns_fallback,
				*trick_dns,
				*direct_dns,
				*local_dns,
				*remote_no_warp_dns,
				// *block_dns,
			},
			Rules: []option.DNSRule{},
		},
	}
	if opt.EnableFakeDNS {
		inet4Range := badoption.Prefix(netip.MustParsePrefix("198.18.0.0/15"))
		inet6Range := badoption.Prefix(netip.MustParsePrefix("fc00::/18"))
		dnsOptions.Servers = append(dnsOptions.Servers, option.DNSServerOptions{
			Tag: DNSFakeTag,
			Options: &option.FakeIPDNSServerOptions{
				Inet4Range: &inet4Range,
				Inet6Range: &inet6Range,
			},
		})
	}
	options.DNS = &dnsOptions

	// options.DNS.StaticIPs["time.apple.com"] = []string{"time.g.aaplimg.com", "time.apple.com"}
	// options.DNS.StaticIPs["ipinfo.io"] = []string{"ipinfo.io"}
	// options.DNS.StaticIPs["dns.cloudflare.com"] = []string{"www.speedtest.net", "cloudflare.com"}
	// options.DNS.StaticIPs["ipwho.is"] = []string{"ipwho.is"}
	// options.DNS.StaticIPs["api.my-ip.io"] = []string{"api.my-ip.io"}
	// options.DNS.StaticIPs["myip.expert"] = []string{"myip.expert"}
	// options.DNS.StaticIPs["ip-api.com"] = []string{"ip-api.com"}
	// options.DNS.StaticIPs["freeipapi.com"] = []string{"www.speedtest.net", "cloudflare.com"}
	// options.DNS.StaticIPs["reallyfreegeoip.org"] = []string{"www.speedtest.net", "cloudflare.com"}
	// options.DNS.StaticIPs["ipapi.co"] = []string{"www.speedtest.net", "cloudflare.com"}
	// options.DNS.StaticIPs["api.ip.sb"] = []string{"www.speedtest.net", "cloudflare.com"}
	return nil
}
func getAllOutboundsOptions(options *option.Options) []any {
	outbounds := []any{}
	for _, o := range options.Outbounds {
		outbounds = append(outbounds, o.Options)
	}
	for _, o := range options.Endpoints {
		outbounds = append(outbounds, o.Options)
	}
	return outbounds
}
func addForceDirect(options *option.Options, hopt *HiddifyOptions) ([]option.DefaultDNSRule, error) {
	dnsMap := make(map[string]string)
	outbounds := getAllOutboundsOptions(options)

	for _, outbound := range outbounds {

		if server, ok := outbound.(option.ServerOptionsWrapper); ok {
			serverDomain := server.TakeServerOptions().Server
			detour := OutboundDirectTag
			if dialer, ok := outbound.(option.DialerOptionsWrapper); ok {
				if server_detour := dialer.TakeDialerOptions().Detour; server_detour != "" {
					detour = server_detour
				}
			}

			if host, err := getHostnameIfNotIP(serverDomain); err == nil {
				if _, ok := dnsMap[host]; !ok || detour == OutboundDirectTag {
					dnsMap[host] = detour
				}
			}
		}
	}

	// dnsMap[]
	forceDirectRules := []option.DefaultDNSRule{}
	if len(dnsMap) > 0 {
		unique_dns_detours := make(map[string]bool)
		for _, detour := range dnsMap {
			unique_dns_detours[detour] = true
		}

		for detour := range unique_dns_detours {
			domains := []string{}
			for domain, d := range dnsMap {
				if d == detour {
					domains = append(domains, domain)
				}
			}
			if len(domains) == 0 {
				continue
			}
			dns_detour := DNSDirectTag
			if detour != OutboundDirectTag {
				dns_detour = "dns-" + detour
				remote_dns, err := getDNSServerOptions(dns_detour, hopt.RemoteDnsAddress, DNSDirectTag, detour)
				if err != nil {
					return nil, err
				}
				options.DNS.Servers = append(options.DNS.Servers, *remote_dns)
				if err != nil {
					return nil, err
				}

			}

			forceDirectRules = append(forceDirectRules,
				option.DefaultDNSRule{
					RawDefaultDNSRule: option.RawDefaultDNSRule{
						Domain: domains,
					},
					DNSRuleAction: option.DNSRuleAction{
						Action: C.RuleActionTypeRoute,
						RouteOptions: option.DNSRouteActionOptions{
							Server:         dns_detour,
							BypassIfFailed: true,
						},
					},
				},
			)
		}
	}
	domains := []string{}

	forceDirectRules = append(forceDirectRules,
		option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{
				Domain: []string{"api.cloudflareclient.com"},
			},
			DNSRuleAction: option.DNSRuleAction{
				Action: C.RuleActionTypeRoute,
				RouteOptions: option.DNSRouteActionOptions{
					Server:         DNSRemoteNoWarpTag,
					BypassIfFailed: true,
				},
			},
		},
	)

	domains = append(domains, "api.cloudflareclient.com")
	for domain, _ := range dnsMap {
		domains = append(domains, domain)
	}

	for _, url := range hopt.ConnectionTestUrls { //To avoid dns bug when using urltest
		if host, err := getHostnameIfNotIP(url); err == nil {
			domains = append(domains, host)
		}
	}

	for _, d := range ipinfo.GetAllIPCheckerDomainsDomains() {
		domains = append(domains, d)
	}

	if len(domains) > 0 {
		forceDirectRules = append(forceDirectRules,
			option.DefaultDNSRule{
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					Domain: domains,
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server:         DNSDirectTag,
						BypassIfFailed: true,
					},
				},
			},
		)
		forceDirectRules = append(forceDirectRules,
			option.DefaultDNSRule{
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					Domain: domains,
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server:         DNSTricksDirectTag,
						BypassIfFailed: true,
					},
				},
			},
		)
		forceDirectRules = append(forceDirectRules,
			option.DefaultDNSRule{
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					Domain: domains,
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server:         DNSLocalTag,
						BypassIfFailed: true,
					},
				},
			},
		)
	}
	return forceDirectRules, nil

}

func getDNSServerOptions(tag string, dnsurl string, domain_resolver string, detour string) (*option.DNSServerOptions, error) {
	serverURL, _ := url.Parse(dnsurl)
	var serverType string
	if serverURL != nil && serverURL.Scheme != "" {
		serverType = serverURL.Scheme
	} else {
		switch dnsurl {
		case "local", "fakeip":
			serverType = dnsurl
		default:
			serverType = C.DNSTypeUDP
		}
	}
	remoteOptions := option.RemoteDNSServerOptions{
		RawLocalDNSServerOptions: option.RawLocalDNSServerOptions{
			DialerOptions: option.DialerOptions{
				Detour: detour,
				DomainResolver: &option.DomainResolveOptions{
					Server:   domain_resolver,
					Strategy: option.DomainStrategy(C.DomainStrategyPreferIPv4),
				},
			},
		},
	}
	o := option.DNSServerOptions{
		Tag: tag,
	}
	switch serverType {
	case C.DNSTypeLocal:
		o.Type = C.DNSTypeLocal
		o.Options = &option.LocalDNSServerOptions{
			RawLocalDNSServerOptions: remoteOptions.RawLocalDNSServerOptions,
		}
	case C.DNSTypeUDP:
		o.Type = C.DNSTypeUDP
		o.Options = &remoteOptions
		var serverAddr M.Socksaddr
		if serverURL == nil || serverURL.Scheme == "" {
			serverAddr = M.ParseSocksaddr(dnsurl)
		} else {
			serverAddr = M.ParseSocksaddr(serverURL.Host)
		}
		if !serverAddr.IsValid() {
			return nil, E.New("invalid server address")
		}
		remoteOptions.Server = serverAddr.AddrString()
		if serverAddr.Port != 0 && serverAddr.Port != 53 {
			remoteOptions.ServerPort = serverAddr.Port
		}
	case C.DNSTypeTCP:
		o.Type = C.DNSTypeTCP
		o.Options = &remoteOptions
		if serverURL == nil {
			return nil, E.New("invalid server address")
		}
		serverAddr := M.ParseSocksaddr(serverURL.Host)
		if !serverAddr.IsValid() {
			return nil, E.New("invalid server address")
		}
		remoteOptions.Server = serverAddr.AddrString()
		if serverAddr.Port != 0 && serverAddr.Port != 53 {
			remoteOptions.ServerPort = serverAddr.Port
		}
	case C.DNSTypeTLS, C.DNSTypeQUIC:
		o.Type = serverType
		if serverURL == nil {
			return nil, E.New("invalid server address")
		}
		serverAddr := M.ParseSocksaddr(serverURL.Host)
		if !serverAddr.IsValid() {
			return nil, E.New("invalid server address")
		}
		remoteOptions.Server = serverAddr.AddrString()
		if serverAddr.Port != 0 && serverAddr.Port != 853 {
			remoteOptions.ServerPort = serverAddr.Port
		}
		o.Options = &option.RemoteTLSDNSServerOptions{
			RemoteDNSServerOptions: remoteOptions,
		}
	case C.DNSTypeHTTPS, C.DNSTypeHTTP3:
		o.Type = serverType
		httpsOptions := option.RemoteHTTPSDNSServerOptions{
			RemoteTLSDNSServerOptions: option.RemoteTLSDNSServerOptions{
				RemoteDNSServerOptions: remoteOptions,
			},
		}
		o.Options = &httpsOptions
		if serverURL == nil {
			return nil, E.New("invalid server address")
		}
		serverAddr := M.ParseSocksaddr(serverURL.Host)
		if !serverAddr.IsValid() {
			return nil, E.New("invalid server address")
		}
		httpsOptions.Server = serverAddr.AddrString()
		if serverAddr.Port != 0 && serverAddr.Port != 443 {
			httpsOptions.ServerPort = serverAddr.Port
		}
		if serverURL.Path != "/dns-query" {
			httpsOptions.Path = serverURL.Path
		}
		if detour == OutboundDirectFragmentTag {
			httpsOptions.TLS = &option.OutboundTLSOptions{
				Enabled:               true,
				Fragment:              true,
				FragmentFallbackDelay: badoption.Duration(30 * time.Millisecond),
				RecordFragment:        true,
			}
		}
	case "rcode":
		var rcode int
		if serverURL == nil {
			return nil, E.New("invalid server address")
		}
		switch serverURL.Host {
		case "success":
			rcode = dnscode.RcodeSuccess
		case "format_error":
			rcode = dnscode.RcodeFormatError
		case "server_failure":
			rcode = dnscode.RcodeServerFailure
		case "name_error":
			rcode = dnscode.RcodeNameError
		case "not_implemented":
			rcode = dnscode.RcodeNotImplemented
		case "refused":
			rcode = dnscode.RcodeRefused
		default:
			return nil, E.New("unknown rcode: ", serverURL.Host)
		}
		o.Type = C.DNSTypeLegacyRcode
		o.Options = rcode
	case C.DNSTypeDHCP:
		o.Type = C.DNSTypeDHCP
		dhcpOptions := option.DHCPDNSServerOptions{}
		if serverURL == nil {
			return nil, E.New("invalid server address")
		}
		if serverURL.Host != "" && serverURL.Host != "auto" {
			dhcpOptions.Interface = serverURL.Host
		}
		o.Options = &dhcpOptions
	case C.DNSTypeFakeIP:
		o.Type = C.DNSTypeFakeIP
		fakeipOptions := option.FakeIPDNSServerOptions{}
		// if legacyOptions, loaded := ctx.Value((*option.LegacyDNSFakeIPOptions)(nil)).(*option.LegacyDNSFakeIPOptions); loaded {
		// 	fakeipOptions.Inet4Range = legacyOptions.Inet4Range
		// 	fakeipOptions.Inet6Range = legacyOptions.Inet6Range
		// }
		o.Options = &fakeipOptions
	default:
		return nil, E.New("unsupported DNS server scheme: ", serverType)

	}
	return &o, nil
}

func getStaticDNSServerOptions(tag string, staticIps *map[string][]string) (*option.DNSServerOptions, error) {
	domain_ips := badjson.TypedMap[string, badoption.Listable[netip.Addr]]{}
	for domain, ips := range *staticIps {
		ipsConverted := make([]netip.Addr, 0, len(ips))
		for _, ip := range ips {
			addr, err := netip.ParseAddr(ip)
			if err != nil {
				return nil, err
			}
			ipsConverted = append(ipsConverted, addr)
		}
		domain_ips.Put(domain, ipsConverted)
	}
	o := option.DNSServerOptions{
		Tag:  tag,
		Type: C.DNSTypeHosts,
		Options: &option.HostsDNSServerOptions{
			Predefined: &domain_ips,
		},
	}
	return &o, nil
}
