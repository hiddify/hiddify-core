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

	C "github.com/sagernet/sing-box/constant"
	sdns "github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// A balancer's member takes the selection when faster by more than the larger of the two: a
// share of the selected member's delay (100 ms is left for 60 ms, not for 90 ms), with a floor
// against the jitter of close pings (25 ms is not left for 22 ms). An absolute 100 ms kept a
// 59 ms member over a 25 ms one.
const (
	BalancerTolerance      uint16 = 30
	BalancerToleranceRatio        = 0.2
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
	// sequential groups (selected server, its TCP/TLS variants, then a fallback resolver);
	// all DNS rules and the DNS final use these
	DNSMultiDirectTag = "dns-direct-group"
	DNSMultiRemoteTag = "dns-remote-group"
	// DoH through the profile alone, past the extra security hop: the remote group's last resort
	DNSRemoteViaProfileTag = "dns-remote-via-profile"

	OutboundDirectTag = "direct §hide§"
	OutboundBypassTag = "bypass §hide§"
	// OutboundBlockTag          = "block §hide§"
	OutboundSelectTag         = "select"
	OutboundURLTestTag        = "lowest"
	OutboundRoundRobinTag     = "balance"
	OutboundDirectFragmentTag = "direct-fragment §hide§"

	InboundTUNTag   = "tun-in"
	InboundMixedTag = "mixed-in"
	// the LAN sharing port, apart from this device's own (proxyListeners)
	InboundLanSharingTag = "lan-sharing-in"
	InboundTProxy        = "tproxy-in"
	InboundRedirect      = "redirect-in"
	InboundDirectTag     = "dns-in"
)

var (
	OutboundMainDetour       = OutboundSelectTag
	OutboundWARPConfigDetour = OutboundDirectFragmentTag
	PredefinedOutboundTags   = []string{OutboundDirectTag, OutboundBypassTag, OutboundSelectTag, OutboundURLTestTag, OutboundDirectFragmentTag, ChainExtraSecurityTag, ChainUnblockerTag}
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
	declareIPv6FromInput(&options, input)

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

// declareIPv6FromInput passes the links' IPv6 declarations (ray2sing, a link's ipv6=1 or ipv6=0)
// to the monitoring, which takes them as known from the start.
func declareIPv6FromInput(options *option.Options, input *option.Options) {
	if input == nil || input.Experimental == nil || input.Experimental.Monitoring == nil || len(input.Experimental.Monitoring.IPv6Declared) == 0 {
		return
	}
	if options.Experimental == nil || options.Experimental.Monitoring == nil {
		return
	}
	options.Experimental.Monitoring.IPv6Declared = input.Experimental.Monitoring.IPv6Declared
}

// dialDetour maps the built-in empty direct outbounds to "" (dial directly),
// since sing-box rejects a detour to an empty direct outbound.
func dialDetour(tag string) string {
	if tag == OutboundDirectTag || tag == OutboundDirectFragmentTag {
		return ""
	}
	return tag
}

func setNTP(options *option.Options) {
	options.NTP = &option.NTPOptions{
		Enabled:       true,
		ServerOptions: option.ServerOptions{ServerPort: 123, Server: "time.apple.com"},
		Interval:      badoption.Duration(12 * time.Hour),
		DialerOptions: option.DialerOptions{
			Detour: dialDetour(OutboundDirectTag),
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

func isDNSOutbound(out option.Outbound) bool {
	if out.Type == C.TypeDNS {
		return true
	}
	invalid, ok := out.Options.(*option.HInvalidOptions)
	return ok && invalid.OriginalType == C.TypeDNS
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
	chainPsiphon := chainUsesPsiphon(opt)
	keptPsiphon := ""
	removedPsiphon := map[string]bool{}
	for _, out := range input.Outbounds {

		if contains(PredefinedOutboundTags, out.Tag) {
			continue
		}
		// dns outbounds were removed in sing-box 1.13; the parser turns them into hinvalid
		// placeholders, which must not end up as selectable proxies
		if isDNSOutbound(out) {
			continue
		}
		outbound, err := patchOutbound(out, *opt, staticIPs)
		if err != nil {
			return err
		}
		out = *outbound

		switch out.Type {
		case C.TypeBlock:
			continue
		case C.TypeSelector, C.TypeURLTest:
			continue
		case C.TypeCustom:
			continue
		default:

			if contains([]string{"direct", "bypass", "block"}, out.Tag) {
				continue
			}
			// Psiphon runs as one instance per process: a Psiphon chain hop replaces every
			// Psiphon of the profile, otherwise only the first one is kept
			if out.Type == C.TypePsiphon {
				if chainPsiphon || keptPsiphon != "" {
					removedPsiphon[out.Tag] = true
					continue
				}
				keptPsiphon = out.Tag
			}
			if !strings.Contains(out.Tag, "§hide§") {
				tags = append(tags, out.Tag)
			}
			outbounds = append(outbounds, out)
		}
	}

	for _, end := range input.Endpoints {
		if contains(PredefinedOutboundTags, end.Tag) {
			continue
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
	redirectRemovedPsiphon(outbounds, endpoints, removedPsiphon, keptPsiphon)
	if err := setChainHop(opt, &outbounds, &endpoints); err != nil {
		return err
	}
	fitWireGuardInWireGuard(endpoints)
	warpBehindDetour(outbounds, endpoints)
	systemEngine := ""
	if opt.UseSystemTLSEngine == nil || *opt.UseSystemTLSEngine {
		systemEngine = usableTLSEngine()
	}
	chooseTLSEngines(outbounds, systemEngine)
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
			Tolerance:      BalancerTolerance,
			ToleranceRatio: BalancerToleranceRatio,
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
			// no tolerance: it is lowest-delay's, and the pool reads it too, and with it a faster
			// member outside would take a slot, moving its sticky keys
			// IdleTimeout:               badoption.Duration(opt.URLTestInterval.Duration().Nanoseconds() * 3),
			InterruptExistConnections: true,
		},
	}
	if len(tags) == 0 {
		return fmt.Errorf("profile has no usable outbound")
	}
	defaultSelect := tags[0]

	for _, tag := range tags {
		if strings.Contains(tag, "§default§") {
			defaultSelect = "§default§"
		}
	}

	selectorTags := tags
	if len(tags) > 1 {
		outbounds = append([]option.Outbound{balancer, urlTest}, outbounds...)
		selectorTags = append([]string{urlTest.Tag, balancer.Tag}, selectorTags...)
		switch opt.DefaultProxy {
		case DefaultProxyLowest:
			defaultSelect = urlTest.Tag
		case DefaultProxyFirst:
		default:
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
						AbstractDialerOptions: option.AbstractDialerOptions{
							TCPFastOpen: false,
						},
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
			hopt.ClashApiSecret = defaultClashApiSecret()
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
				Enabled: true, // WARP and MASQUE devices are stored by default
				Path:    "data/clash.db",
			},

			Monitoring: &option.MonitoringOptions{
				URLs:      hopt.ConnectionTestUrls,
				IpInfoURL: hopt.IpInfoUrl,
				// each proxy's IPv6 is verified once at its URL test, with the same test URL over
				// IPv6, when the apps get IPv6 at all (see ipv6ForClients)
				VerifyIPv6:     ipv6ForClients(hopt.IPv6Mode),
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

// dnsStrategy is the strategy of a DNS rule: the explicit setting when there is one, else what
// the IPv6 mode asks. Neither ever becomes ipv4_only or ipv6_only: the core's own lookups then
// favour one family, but a client still gets both A and AAAA, and a config whose server has only
// the other family still routes (hiddify-app #2307, #2420 are about IPv6-first sites on proxies
// without IPv6; the mode used to reach only the local listeners).
func dnsStrategy(explicit, ipv6Mode option.DomainStrategy) option.DomainStrategy {
	strategy := explicit
	if strategy == option.DomainStrategy(C.DomainStrategyAsIS) {
		strategy = ipv6Mode
	}
	switch strategy {
	case option.DomainStrategy(C.DomainStrategyIPv4Only):
		return option.DomainStrategy(C.DomainStrategyPreferIPv4)
	case option.DomainStrategy(C.DomainStrategyIPv6Only):
		return option.DomainStrategy(C.DomainStrategyPreferIPv6)
	}
	return strategy
}

// ipv6ForClients says whether the apps on the device get IPv6 at all: AAAA answers, and IPv6
// connections through the proxy. Off unless the IPv6 mode turns it on (the user, or a
// subscription's ipv6-mode header): most proxies' exits have no IPv6, and a proxy accepts an
// IPv6 connection and then closes it, which a browser shows as a dead site and never retries over
// IPv4 (hiddify-app #2307, #2420). On, IPv6 goes only through a proxy found carrying it: the
// monitoring verifies each proxy once at its URL test (verify_ipv6), the router carries over IPv4
// a connection the proxy cannot (ipv6_fallback) and learns from it, and the DNS and route rules
// (ipv6_by_outbound, ipv6_unavailable_via) keep IPv6 away from a proxy found without. The core's
// own lookups (the proxy's address, the test URLs) use both families either way, by dnsStrategy.
func ipv6ForClients(ipv6Mode option.DomainStrategy) bool {
	switch ipv6Mode {
	case option.DomainStrategy(C.DomainStrategyPreferIPv4), option.DomainStrategy(C.DomainStrategyPreferIPv6), option.DomainStrategy(C.DomainStrategyIPv6Only):
		return true
	}
	return false
}

// ipv6ByOutbound names the outbound whose IPv6, as the monitoring knows it, limits a DNS rule's
// AAAA answers: the detour the rule's traffic takes, when the apps get IPv6 at all.
func ipv6ByOutbound(hopt *HiddifyOptions, detour string) string {
	if !ipv6ForClients(hopt.IPv6Mode) {
		return ""
	}
	return detour
}

// clientDNSStrategy is the strategy of a DNS rule that answers the apps: ipv4_only unless the
// IPv6 mode turns IPv6 on (then the mode's preference, see dnsStrategy), or the explicit setting
// names IPv6 itself, which is kept as asked.
func clientDNSStrategy(explicit, ipv6Mode option.DomainStrategy) option.DomainStrategy {
	switch explicit {
	case option.DomainStrategy(C.DomainStrategyPreferIPv6), option.DomainStrategy(C.DomainStrategyIPv6Only):
		return explicit
	}
	if !ipv6ForClients(ipv6Mode) {
		return option.DomainStrategy(C.DomainStrategyIPv4Only)
	}
	return dnsStrategy(explicit, ipv6Mode)
}

// shouldEnableIPv6 says whether the local listeners also bind IPv6: whenever the host has it. The
// IPv6 mode is a preference between the families, never a switch; it used to leave ::1 unbound.
func shouldEnableIPv6() bool {
	return isIPv6Supported()
}

func setInbound(options *option.Options, hopt *HiddifyOptions) {
	// var inboundDomainStrategy option.DomainStrategy
	// if !opt.ResolveDestination {
	// 	inboundDomainStrategy = option.DomainStrategy(dns.DomainStrategyAsIS)
	// } else {
	// 	inboundDomainStrategy = opt.IPv6Mode
	// }
	ipv6Enable := shouldEnableIPv6()
	if hopt.EnableTun {

		opts := option.TunInboundOptions{
			Stack:       TunStack(hopt.TUNStack),
			MTU:         hopt.MTU,
			AutoRoute:   true,
			StrictRoute: hopt.StrictRoute,
			// nftables routing on Linux, with pre-matching through nfqueue: better routing and
			// performance than the TUN alone, and no conflict with Docker bridges. Left out where it
			// cannot run, and after a start it made fail (see AutoRedirectFailed).
			AutoRedirect: hopt.AutoRedirect && AutoRedirectAvailable(),

			// EndpointIndependentNat: true,
			// GSO:                    runtime.GOOS != "windows",

		}
		tunInbound := option.Inbound{
			Type: C.TypeTun,
			Tag:  InboundTUNTag,

			Options: &opts,
		}
		// switch hopt.IPv6Mode {
		// case option.DomainStrategy(C.DomainStrategyIPv4Only):
		// 	opts.Address = []netip.Prefix{
		// 		netip.MustParsePrefix("172.19.0.1/28"),
		// 	}
		// case option.DomainStrategy(dns.DomainStrategyUseIPv6):
		// 	opts.Address = []netip.Prefix{
		// 		netip.MustParsePrefix("fdfe:dcba:9876::1/126"),
		// 	}
		// default:

		// }
		opts.Address = []netip.Prefix{
			netip.MustParsePrefix("172.19.0.1/28"),
			// Always on the TUN when auto-routing. Without it, sing-tun installs no ::/0
			// and IPv6 bypasses the VPN (NE builds guard this in Swift; admin_bgservice /
			// desktop TUN must do it here). ipv6Mode only affects DNS strategy below.
			netip.MustParsePrefix("fdfe:dcba:9876::1/126"),
		}

		options.Inbounds = append(options.Inbounds, tunInbound)

	}

	for _, listener := range proxyListeners(hopt, ipv6Enable) {
		addr := badoption.Addr(listener.addr)
		options.Inbounds = append(options.Inbounds, option.Inbound{
			Type: C.TypeMixed,
			Tag:  listener.tag,
			Options: &option.HTTPMixedInboundOptions{
				ListenOptions:  option.ListenOptions{Listen: &addr, ListenPort: listener.port},
				Users:          listener.users,
				SetSystemProxy: listener.systemProxy,
			},
		})
	}

	if !transparentPortsOpen(hopt) {
		return
	}
	// the transparent ports, on the addresses they always had: every one while shared (a Linux
	// gateway sends the network's traffic to them), this device's own otherwise
	binds := []string{"127.0.0.1"}
	if ipv6Enable {
		binds = append(binds, "::1")
	}
	if hopt.AllowConnectionFromLAN {
		binds = []string{"0.0.0.0"}
		if ipv6Enable {
			binds = []string{"::"}
		}
	}
	for _, bind := range binds {
		addr := badoption.Addr(netip.MustParseAddr(bind))
		listen := option.ListenOptions{Listen: &addr}
		if tproxyAvailable() && portEnabled(hopt.EnableTProxyPort, hopt.TProxyPort, false) {
			listen.ListenPort = hopt.TProxyPort
			options.Inbounds = append(options.Inbounds, option.Inbound{
				Type:    C.TypeTProxy,
				Tag:     InboundTProxy + bind,
				Options: &option.TProxyInboundOptions{ListenOptions: listen},
			})
		}
		if redirectAvailable() && portEnabled(hopt.EnableRedirectPort, hopt.RedirectPort, false) {
			listen.ListenPort = hopt.RedirectPort
			options.Inbounds = append(options.Inbounds, option.Inbound{
				Type:    C.TypeRedirect,
				Tag:     InboundRedirect + bind,
				Options: &option.RedirectInboundOptions{ListenOptions: listen},
			})
		}
		if !C.IsAndroid && !C.IsIos && portEnabled(hopt.EnableDirectPort, hopt.DirectPort, false) {
			listen.ListenPort = hopt.DirectPort
			options.Inbounds = append(options.Inbounds, option.Inbound{
				Type:    C.TypeDirect,
				Tag:     InboundDirectTag + bind,
				Options: &option.DirectInboundOptions{ListenOptions: listen},
			})
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

	// An IPv6 connection is refused at once, before its handshake, so the app falls back to IPv4
	// instead of being accepted and then closed by the proxy: every one while IPv6 is off for the
	// apps (see ipv6ForClients); otherwise those through a proxy the monitoring found without
	// IPv6 (ipv6_unavailable_via), the others going on, carried over IPv4 by the router when the
	// proxy turns out not to carry them. The rule goes before the sniff rule: the TUN judges a
	// TCP connection before its handshake only up to the first rule that needs the stream, and a
	// reject after the handshake is the very "connection closed" the browser gives up on. The TUN
	// keeps ::/0, so nothing leaks around the VPN. The LAN keeps its IPv6 when the LAN is bypassed.
	{
		if hopt.BypassLAN && (hopt.RouteRule == nil || hopt.RouteRule.RouteRule == nil) {
			routeRules = append(routeRules, option.Rule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultRule{
					RawDefaultRule: option.RawDefaultRule{IPVersion: 6, IPIsPrivate: true},
					RuleAction: option.RuleAction{
						Action:       C.RuleActionTypeRoute,
						RouteOptions: option.RouteActionOptions{Outbound: OutboundDirectTag},
					},
				},
			})
		}
		routeRules = append(routeRules, option.Rule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultRule{
				RawDefaultRule: option.RawDefaultRule{IPVersion: 6, IPv6UnavailableVia: ipv6ByOutbound(hopt, OutboundMainDetour)},
				RuleAction: option.RuleAction{
					Action:        C.RuleActionTypeReject,
					RejectOptions: option.RejectActionOptions{Method: C.RuleActionRejectMethodDefault},
				},
			},
		})
	}
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
				IPCIDR: prefixables([]string{
					"10.10.34.0/24",
					"2001:4188:2:600:10:10:34:0/120",
				}),
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

	appRules := buildAppRouteRules(hopt)
	useAppRules := hopt.RouteRule != nil && hopt.RouteRule.RouteRule != nil
	if useAppRules {
		routeRules = append(routeRules, appRules.routeRules...)
	}
	if !useAppRules && hopt.BypassLAN {
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
					Server: DNSMultiDirectTag,
					AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
						Strategy:       clientDNSStrategy(hopt.DirectDnsDomainStrategy, hopt.IPv6Mode),
						IPv6ByOutbound: ipv6ByOutbound(hopt, OutboundDirectTag),
						RewriteTTL:     &DEFAULT_DNS_TTL,
						DisableCache:   false,
						BypassIfFailed: false,
					},
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
	if useAppRules {
		dnsRules = append(dnsRules, appRules.dnsRules...)
		rulesets = append(rulesets, appRules.ruleSets...)
	}
	if !useAppRules && hopt.BlockAds {
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geosite-ads"},
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-category-ads-all.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geosite-malware"},
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-malware.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geosite-phishing"},
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-phishing.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geosite-cryptominers"},
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geosite-cryptominers.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geoip-phishing"},
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/block/geoip-phishing.srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geoip-malware"},
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
	if !useAppRules && hopt.Region != "other" {
		dnsRules = append(dnsRules, option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{
				DomainSuffix: []string{"." + hopt.Region},
			},
			DNSRuleAction: option.DNSRuleAction{
				Action: C.RuleActionTypeRoute,
				RouteOptions: option.DNSRouteActionOptions{
					Server: DNSMultiDirectTag,
					AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
						Strategy:       clientDNSStrategy(hopt.DirectDnsDomainStrategy, hopt.IPv6Mode),
						IPv6ByOutbound: ipv6ByOutbound(hopt, OutboundDirectTag),
						RewriteTTL:     &DEFAULT_DNS_TTL,
						BypassIfFailed: false,
					},
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
					Server: DNSMultiDirectTag,
					AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
						Strategy:       clientDNSStrategy(hopt.DirectDnsDomainStrategy, hopt.IPv6Mode),
						IPv6ByOutbound: ipv6ByOutbound(hopt, OutboundDirectTag),
						RewriteTTL:     &DEFAULT_DNS_TTL,
						BypassIfFailed: false,
					},
				},
			},
		})

		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geoip-" + hopt.Region},
			Format: C.RuleSetFormatBinary,
			RemoteOptions: option.RemoteRuleSet{
				URL:            "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/country/geoip-" + hopt.Region + ".srs",
				UpdateInterval: badoption.Duration(5 * time.Hour * 24),
				DownloadDetour: OutboundSelectTag,
			},
		})
		rulesets = append(rulesets, option.RuleSet{
			Type:   C.RuleSetTypeRemote,
			Tag:    badoption.Listable[string]{"geosite-" + hopt.Region},
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
		IPv6Fallback:        ipv6ForClients(hopt.IPv6Mode), // see ipv6ForClients
		AutoDetectInterface: (!C.IsAndroid && !C.IsIos) && (hopt.EnableTun || hopt.EnableTunService),
		DefaultDomainResolver: &option.DomainResolveOptions{
			Server:   DNSMultiDirectTag,
			Strategy: dnsStrategy(hopt.DirectDnsDomainStrategy, hopt.IPv6Mode),
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
				// No query_type matcher: it takes sing-box out of the legacy DNS mode, and the
				// strategy the rules set is fatal outside it (the core stopped at Connecting).
				// The fake IP server answers only A and AAAA and fails the rest, which
				// bypass_if_failed sends on to the remote rule below.
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					// Inbound: inbounds,
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server: DNSFakeTag,
						AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
							Strategy:       clientDNSStrategy(hopt.RemoteDnsDomainStrategy, hopt.IPv6Mode),
							IPv6ByOutbound: ipv6ByOutbound(hopt, OutboundMainDetour),
							RewriteTTL:     &DEFAULT_DNS_TTL,
							DisableCache:   true,
							BypassIfFailed: true,
						},
					},
				},
			})

	}

	dnsRules = append(dnsRules, option.DefaultDNSRule{
		RawDefaultDNSRule: option.RawDefaultDNSRule{},
		DNSRuleAction: option.DNSRuleAction{
			Action: C.RuleActionTypeRoute,
			RouteOptions: option.DNSRouteActionOptions{
				Server: DNSMultiRemoteTag,
				AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
					// no AAAA for the apps unless the IPv6 mode turns IPv6 on, and then none
					// through a proxy found without IPv6 (hiddify-app #2307, #2420)
					Strategy:       clientDNSStrategy(hopt.RemoteDnsDomainStrategy, hopt.IPv6Mode),
					IPv6ByOutbound: ipv6ByOutbound(hopt, OutboundMainDetour),
					RewriteTTL:     &DEFAULT_DNS_TTL,
					BypassIfFailed: false,
				},
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
	// 			Strategy:       dnsStrategy(hopt.RemoteDnsDomainStrategy, hopt.IPv6Mode),
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

// defaultClashApiSecret is generated once per process, so the same settings build the same
// config (hot reload compares the running config with the new one).
var defaultClashApiSecret = sync.OnceValue(func() string {
	return generateRandomString(16)
})

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
