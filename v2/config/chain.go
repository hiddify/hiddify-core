package config

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
	"github.com/sagernet/wireguard-go/hiddify"
)

// Chain hops ("extra security" / "unblocker") configured in the app settings.
// Traffic order: app -> extra security -> main profile -> unblocker -> internet.
const (
	ChainStatusOff           = "off"
	ChainStatusExtraSecurity = "extra_security"
	ChainStatusUnblocker     = "unblocker"

	ChainModeWarp       = "warp"        // Cloudflare WARP over WireGuard
	ChainModeWarpMasque = "warp_masque" // Cloudflare WARP over MASQUE (HTTP/2 when behind a detour)
	ChainModeWarpAwg    = "warp_awg"    // Cloudflare WARP over AmneziaWG
	ChainModeWarpAuto   = "warp_auto"   // race WireGuard / AmneziaWG / MASQUE; keep the fastest
	ChainModePsiphon    = "psiphon"
	ChainModeProfile    = "profile"
	// every WARP mode and Psiphon at once, in a selector the user can move on the fly, with a
	// lowest-delay balancer of them as its default
	ChainModeSelectable = "selectable"
	chainModeLowest     = "lowest" // the balancer of the selectable mode

	// Visible name prefixes (mode is appended by ChainHopTag). §hide§ keeps them out of
	// the user selector while still exposed via proxy info / chain-hops.
	ChainExtraSecurityPrefix = "🔒 Extra Security"
	ChainUnblockerPrefix     = "🔓 Unblocker"

	// Mode-less defaults (tests / PredefinedOutboundTags). Live hops use ChainHopTag(status, mode).
	ChainExtraSecurityTag = ChainExtraSecurityPrefix + " §hide§"
	ChainUnblockerTag     = ChainUnblockerPrefix + " §hide§"

	psiphonRegionAuto = "AUTO"

	// Nested WireGuard/AWG behind another proxy. QUIC/IPv6 paths need ~1240; 1280 still
	// drops large packets on some nested tunnels. MASQUE keeps its own minimum (1280).
	chainNestedWireGuardMTU = 1240
	// The lowest MTU MASQUE accepts; auto mode uses it behind a detour, for all its legs.
	chainNestedMASQUEMTU = 1280
)

type ChainOptions struct {
	ChainStatus   string          `json:"chain-status,omitempty"`
	ExtraSecurity ChainHopOptions `json:"extra-security,omitempty"`
	Unblocker     ChainHopOptions `json:"unblocker,omitempty"`
}

type ChainHopOptions struct {
	Mode    string              `json:"mode,omitempty"`
	Warp    ChainWarpOptions    `json:"warp,omitempty"`
	Psiphon ChainPsiphonOptions `json:"psiphon,omitempty"`
	Profile ChainProfileOptions `json:"profile,omitempty"`
}

// Where the WARP peer of a hop comes from.
const (
	// WarpEndpointProfile uses the endpoint Cloudflare assigned to the device.
	WarpEndpointProfile = "profile"
	// WarpEndpointScan finds one with the scanner, which remembers what answered per network.
	WarpEndpointScan = "scan"
	// WarpEndpointCustom uses CleanIP and CleanPort as given.
	WarpEndpointCustom = "custom"
)

type ChainWarpOptions struct {
	LicenseKey string `json:"license-key,omitempty"`

	// Endpoint is where the peer comes from: WarpEndpointProfile, Scan or Custom. Empty is read
	// from CleanIP, which carried the same choice as the words auto/default/random before the
	// source was its own field.
	Endpoint string `json:"endpoint,omitempty"`
	// CleanIP and CleanPort are the peer, for the custom source. A literal address or hostname,
	// never a keyword: an old auto/default/random value now means a scan.
	CleanIP   string `json:"clean-ip,omitempty"`
	CleanPort uint16 `json:"clean-port,omitempty"`

	// Scanner tunes the endpoint scan of the scan source; nil is the scanner's default.
	Scanner *ChainWarpScannerOptions `json:"scanner,omitempty"`
	// Masque is the tunnel of the masque and auto modes.
	Masque *ChainWarpMasqueOptions `json:"masque,omitempty"`
	// Awg is the tunnel of the awg and auto modes; nil keeps the core's AmneziaWG preset.
	Awg *ChainWarpAwgOptions `json:"awg,omitempty"`
	// MTU of the tunnel; 0 keeps the mode's default.
	MTU uint32 `json:"mtu,omitempty"`

	Noise      string `json:"noise,omitempty"`
	NoiseSize  string `json:"noise-size,omitempty"`
	NoiseDelay string `json:"noise-delay,omitempty"`
	NoiseMode  string `json:"noise-mode,omitempty"`
}

// ChainWarpScannerOptions is the endpoint scan of a hop, over the scanner's own defaults. Zero
// values keep the default, so a form sends only what the user changed.
type ChainWarpScannerOptions struct {
	// SearchNoise tries the noise presets when no endpoint answers with the configured noise, and
	// keeps the one that works for the network. Nil keeps it on, as for the unblocker hop.
	SearchNoise *bool `json:"search-noise,omitempty"`
	// MaxRTT is the slowest accepted handshake, in milliseconds (1000).
	MaxRTT int `json:"max-rtt,omitempty"`
	// Timeout is how long one probe waits for its handshake, in milliseconds (2000).
	Timeout int `json:"timeout,omitempty"`
	// Count is the most addresses probed in one scan (64).
	Count int `json:"count,omitempty"`
	// Concurrency is how many probes run at once (8).
	Concurrency int `json:"concurrency,omitempty"`
	// Ports to probe; the protocol's own ports when empty.
	Ports []uint16 `json:"ports,omitempty"`
	// CIDRs to draw addresses from; Cloudflare's WARP ranges when empty.
	CIDRs []string `json:"cidrs,omitempty"`
	// NoisePresets to try, in order; all of them when empty.
	NoisePresets []string `json:"noise-presets,omitempty"`
	// ChainSearchOptions: how long and how hard the search goes on, on the unblocker hop.
	ChainSearchOptions
}

// ChainSearchOptions tune the search of a scanner (WARP endpoints, Psiphon CDN edges): how long it
// goes on before it offers a deep search, and the size of the address blocks it rates. They apply
// to the unblocker hop only: behind a detour the hop does not search. Zero values keep the
// scanner's defaults.
type ChainSearchOptions struct {
	// SearchTimeout is how long a search goes on before it offers a deep search, in seconds
	// (180). A deep search has no limit, or a long one: -1 is a deep search from the start, a
	// value above 180 one with that limit.
	SearchTimeout int `json:"search-timeout,omitempty"`
	// DeepConcurrency is the probes a deep search runs at once (twice the scanner's concurrency).
	DeepConcurrency int `json:"deep-concurrency,omitempty"`
	// BlockBitsIPv4 and BlockBitsIPv6 size the address blocks rated per network (24 and 48).
	BlockBitsIPv4 int `json:"block-bits-ipv4,omitempty"`
	BlockBitsIPv6 int `json:"block-bits-ipv6,omitempty"`
}

// edgeSearch is the scanner's search options of the hop; none behind a detour.
func (o ChainSearchOptions) edgeSearch(chainStatus string) option.EdgeSearchOptions {
	if chainStatus != ChainStatusUnblocker {
		return option.EdgeSearchOptions{}
	}
	return option.EdgeSearchOptions{
		SearchTimeout:   badoption.Duration(time.Duration(o.SearchTimeout) * time.Second),
		DeepConcurrency: o.DeepConcurrency,
		BlockBitsIPv4:   o.BlockBitsIPv4,
		BlockBitsIPv6:   o.BlockBitsIPv6,
	}
}

// ChainWarpMasqueOptions is the MASQUE tunnel of the masque and auto modes.
type ChainWarpMasqueOptions struct {
	// VHTTP is the HTTP version: "auto" (HTTP/3, falling back to HTTP/2), "h3" or "h2". A hop
	// behind a detour runs h2 whatever this says, because a detour cannot carry QUIC.
	VHTTP string `json:"vhttp,omitempty"`
	// ServerName is the TLS server name; Cloudflare's MASQUE name when empty.
	ServerName string `json:"server-name,omitempty"`
	UseIPv6    bool   `json:"use-ipv6,omitempty"`
	// UDPTimeout, UDPKeepalivePeriod and ReconnectDelay are in seconds.
	UDPTimeout           int    `json:"udp-timeout,omitempty"`
	UDPKeepalivePeriod   int    `json:"udp-keepalive,omitempty"`
	UDPInitialPacketSize uint16 `json:"udp-initial-packet-size,omitempty"`
	ReconnectDelay       int    `json:"reconnect-delay,omitempty"`

	TLSInsecure bool `json:"tls-insecure,omitempty"`
	TLSFragment bool `json:"tls-fragment,omitempty"`
	// TLSFragmentFallbackDelay is in milliseconds.
	TLSFragmentFallbackDelay int  `json:"tls-fragment-fallback-delay,omitempty"`
	TLSRecordFragment        bool `json:"tls-record-fragment,omitempty"`

	ECHEnabled         bool   `json:"ech-enabled,omitempty"`
	ECHConfig          string `json:"ech-config,omitempty"`
	ECHQueryServerName string `json:"ech-query-server-name,omitempty"`
}

// ChainWarpAwgOptions is the AmneziaWG tunnel of the awg and auto modes. All zero keeps the
// core's preset, so an untouched form does not send AmneziaWG without its obfuscation.
type ChainWarpAwgOptions struct {
	Jc   int    `json:"jc,omitempty"`
	Jmin int    `json:"jmin,omitempty"`
	Jmax int    `json:"jmax,omitempty"`
	S1   int    `json:"s1,omitempty"`
	S2   int    `json:"s2,omitempty"`
	S3   int    `json:"s3,omitempty"`
	S4   int    `json:"s4,omitempty"`
	H1   string `json:"h1,omitempty"`
	H2   string `json:"h2,omitempty"`
	H3   string `json:"h3,omitempty"`
	H4   string `json:"h4,omitempty"`
	I1   string `json:"i1,omitempty"`
	I2   string `json:"i2,omitempty"`
	I3   string `json:"i3,omitempty"`
	I4   string `json:"i4,omitempty"`
	I5   string `json:"i5,omitempty"`
}

type ChainPsiphonOptions struct {
	Region           string `json:"region,omitempty"`
	ConduitPairingID string `json:"conduit-pairing-id,omitempty"`
	// Scanner tunes the CDN edge scanner of the unblocker hop; behind a detour (extra security)
	// Psiphon does not scan. Nil is the scanner's default.
	Scanner *ChainPsiphonScannerOptions `json:"scanner,omitempty"`
}

// ChainPsiphonScannerOptions is the CDN edge scan of the unblocker hop, over the scanner's own
// defaults. Zero values keep the default.
type ChainPsiphonScannerOptions struct {
	// Enabled turns the scanner off with false; nil keeps it on.
	Enabled *bool `json:"enabled,omitempty"`
	// Sets limits the built-in CDN sets (fastly, psiphon-akamai, cloudflare, ...); all when empty.
	Sets []string `json:"sets,omitempty"`
	// CIDRs are IPv4 edges to scan instead of the built-in sets, with ServerNames.
	CIDRs       []string `json:"cidrs,omitempty"`
	ServerNames []string `json:"server-names,omitempty"`
	// Candidates is the answering edges of each CDN handed to Psiphon (2).
	Candidates int `json:"candidates,omitempty"`
	// Count is the most edges probed in one round (256).
	Count int `json:"count,omitempty"`
	// Concurrency is the probes run at once (16 on phones, 48 on desktops).
	Concurrency int `json:"concurrency,omitempty"`
	// Timeout is how long one probe waits, and MaxRTT the slowest accepted handshake, in
	// milliseconds (3000, 2000).
	Timeout int `json:"timeout,omitempty"`
	MaxRTT  int `json:"max-rtt,omitempty"`
	ChainSearchOptions
}

type ChainProfileOptions struct {
	ID *string `json:"id,omitempty"`
}

// ChainModeDisplayName is the user-facing WARP/Psiphon label for a chain mode.
func ChainModeDisplayName(mode string) string {
	switch mode {
	case ChainModeWarp:
		return "WARP · WireGuard"
	case ChainModeWarpMasque:
		return "WARP · MASQUE"
	case ChainModeWarpAwg:
		return "WARP · AmneziaWG"
	case ChainModeWarpAuto:
		return "WARP · Auto"
	case ChainModePsiphon:
		return "Psiphon"
	case ChainModeProfile:
		return "Profile"
	case ChainModeSelectable:
		return "Selectable"
	case chainModeLowest:
		return "Lowest"
	default:
		return mode
	}
}

// ChainHopTag names the hop with its mode, e.g. "🔒 Extra Security · WARP · WireGuard §hide§".
func ChainHopTag(status, mode string) string {
	name := ChainModeDisplayName(mode)
	switch status {
	case ChainStatusExtraSecurity:
		return ChainExtraSecurityPrefix + " · " + name + " §hide§"
	case ChainStatusUnblocker:
		return ChainUnblockerPrefix + " · " + name + " §hide§"
	default:
		return name + " §hide§"
	}
}

// ChainTagDisplay is the tag without §hide§ (for UI lists).
func ChainTagDisplay(tag string) string {
	return strings.TrimSpace(strings.Split(tag, "§")[0])
}

func IsChainExtraSecurityTag(tag string) bool {
	return strings.HasPrefix(ChainTagDisplay(tag), ChainExtraSecurityPrefix)
}

func IsChainUnblockerTag(tag string) bool {
	return strings.HasPrefix(ChainTagDisplay(tag), ChainUnblockerPrefix)
}

// setChainHop adds the active chain hop and routes the main profile through it:
//   - extra security: the hop dials through the main selector and all traffic enters the hop
//   - unblocker: the hop dials directly and every main outbound/endpoint dials through the hop
func setChainHop(opt *HiddifyOptions, outbounds *[]option.Outbound, endpoints *[]option.Endpoint) error {
	var (
		hop    ChainHopOptions
		tag    string
		detour string
	)
	switch opt.ChainStatus {
	case "", ChainStatusOff:
		return nil
	case ChainStatusExtraSecurity:
		hop = opt.ExtraSecurity
		tag = ChainHopTag(ChainStatusExtraSecurity, hop.Mode)
		detour = OutboundSelectTag // the hop follows the proxy chosen in select
	case ChainStatusUnblocker:
		hop = opt.Unblocker
		tag = ChainHopTag(ChainStatusUnblocker, hop.Mode)
		detour = dialDetour(OutboundDirectTag)
	default:
		return fmt.Errorf("unknown chain status: %s", opt.ChainStatus)
	}

	if hop.Mode == ChainModeSelectable {
		chainSelectableHop(opt.ChainStatus, hop, tag, detour, outbounds, endpoints)
	} else if err := addChainHop(opt.ChainStatus, hop, hop.Mode, tag, detour, outbounds, endpoints); err != nil {
		return err
	}

	if opt.ChainStatus == ChainStatusExtraSecurity {
		OutboundMainDetour = tag
		return nil
	}
	// the resolver the proxies use without the hop (route.default_domain_resolver)
	resolver := &option.DomainResolveOptions{
		Server:   DNSMultiDirectTag,
		Strategy: dnsStrategy(opt.DirectDnsDomainStrategy, opt.IPv6Mode),
	}
	for i := range *outbounds {
		chainDialer((*outbounds)[i].Options, (*outbounds)[i].Tag, tag, resolver)
	}
	for i := range *endpoints {
		chainDialer((*endpoints)[i].Options, (*endpoints)[i].Tag, tag, resolver)
	}
	return nil
}

// addChainHop adds the hop of one mode, named tag.
func addChainHop(status string, hop ChainHopOptions, mode, tag, detour string, outbounds *[]option.Outbound, endpoints *[]option.Endpoint) error {
	switch mode {
	case ChainModeWarp, ChainModeWarpAwg, ChainModeWarpMasque, ChainModeWarpAuto:
		*endpoints = append(*endpoints, chainWarpEndpoint(tag, status, hop.Warp, detour, mode))
	case ChainModePsiphon:
		*outbounds = append(*outbounds, chainPsiphonOutbound(tag, status, hop.Psiphon, detour))
	case ChainModeProfile:
		// refuse instead of silently connecting without the requested hop
		return fmt.Errorf("%s: profile mode is not supported yet", status)
	default:
		return fmt.Errorf("%s: unknown mode: %s", status, mode)
	}
	return nil
}

// chainSelectableModes are the hops of the selectable mode, in the order the selector lists them.
var chainSelectableModes = []string{ChainModeWarpAuto, ChainModeWarp, ChainModeWarpAwg, ChainModeWarpMasque, ChainModePsiphon}

// chainSelectableHop adds a hop of each mode, a lowest-delay balancer of them, and the selector
// named tag over the balancer (its default) and every hop: the app moves the hop with
// SelectOutbound while connected. Each WARP hop has its own device.
func chainSelectableHop(status string, hop ChainHopOptions, tag, detour string, outbounds *[]option.Outbound, endpoints *[]option.Endpoint) {
	members := make([]string, 0, len(chainSelectableModes))
	for _, mode := range chainSelectableModes {
		memberTag := ChainHopTag(status, mode)
		_ = addChainHop(status, hop, mode, memberTag, detour, outbounds, endpoints) // the modes are known
		members = append(members, memberTag)
	}
	lowest := ChainHopTag(status, chainModeLowest)
	*outbounds = append(*outbounds,
		option.Outbound{
			Type: C.TypeBalancer,
			Tag:  lowest,
			Options: &option.BalancerOutboundOptions{
				Outbounds:                 members,
				Strategy:                  "lowest-delay",
				DelayAcceptableRatio:      2,
				Tolerance:                 BalancerTolerance,
				ToleranceRatio:            BalancerToleranceRatio,
				InterruptExistConnections: true,
			},
		},
		option.Outbound{
			Type: C.TypeSelector,
			Tag:  tag,
			Options: &option.SelectorOutboundOptions{
				Outbounds:                 append([]string{lowest}, members...),
				Default:                   lowest,
				InterruptExistConnections: true,
			},
		},
	)
}

// chainDialer makes a main outbound/endpoint dial through the unblocker hop, unless it already
// dials through another detour (it is then not the first hop of its own chain) or is the hop itself.
//
// It also resolves the proxy's server name itself, with resolver, unless the proxy names its own.
// A dialer with a detour hands the name to the detour, and the WARP hop looks it up through the
// DNS rules: the final one is the remote DNS, which goes through select, the very proxies waiting
// for the name. Every proxy given by a domain (all the CDN ones) then timed out while the hop
// answered its own test.
func chainDialer(options any, outboundTag, hopTag string, resolver *option.DomainResolveOptions) {
	if outboundTag == hopTag || IsChainUnblockerTag(outboundTag) { // the hops of the selectable mode too
		return
	}
	wrapper, ok := options.(option.DialerOptionsWrapper)
	if !ok {
		return
	}
	dialer := wrapper.TakeDialerOptions()
	if dialer.Detour != "" {
		return
	}
	dialer.Detour = hopTag
	if dialer.DomainResolver == nil || dialer.DomainResolver.Server == "" {
		dialer.DomainResolver = resolver
	}
	wrapper.ReplaceDialerOptions(dialer)
}

// chainWarpModes are the tunnels of the warp endpoint (its mode option) for the WARP chain modes.
var chainWarpModes = map[string]string{
	ChainModeWarp:       "wg",
	ChainModeWarpAwg:    "awg",
	ChainModeWarpMasque: "masque",
	ChainModeWarpAuto:   "auto", // races WireGuard, AmneziaWG and MASQUE and keeps the fastest
}

// chainWarpEndpoint is a warp endpoint running the tunnel of mode, with the peer the hop asks for
// (see chainWarpEndpointSource). Behind a detour (TCP-only, it cannot carry QUIC) MASQUE runs over
// HTTP/2 and the tunnel keeps a smaller MTU.
func chainWarpEndpoint(tag, chainStatus string, warp ChainWarpOptions, detour, mode string) option.Endpoint {
	warpMode := chainWarpModes[mode]
	options := &option.WARPEndpointOptions{
		// A stable key per hop and mode: the endpoint takes its own device back from the pool.
		UniqueIdentifier: "chain-" + chainStatus + "-" + mode,
		Profile: option.WARPProfile{
			License: warp.LicenseKey,
			// Enroll without Profile.Detour so select cannot block registration before
			// leaves are ready; tunnel traffic still uses DialerOptions.Detour.
		},
		Noise: chainWarpNoise(warp),
		Mode:  warpMode,
	}
	options.Detour = detour
	switch chainWarpEndpointSource(chainStatus, warp, detour, warpMode) {
	case WarpEndpointCustom:
		// In masque mode the endpoint reads this as the MASQUE peer, not the WireGuard one; in
		// auto mode it is the WireGuard peer of every leg that runs WireGuard.
		options.Server = strings.TrimSpace(warp.CleanIP)
		options.ServerPort = warp.CleanPort
	case WarpEndpointScan:
		options.Scanner = chainWarpScanner(chainStatus, warp.Scanner)
	}
	if warp.MTU != 0 {
		options.MTU = warp.MTU
	}
	if warpMode == "masque" || warpMode == "auto" {
		options.MASQUE = chainWarpMasque(warp.Masque, detour)
		if warp.Masque != nil && warp.Masque.UDPTimeout > 0 {
			options.UDPTimeout = badoption.Duration(time.Duration(warp.Masque.UDPTimeout) * time.Second)
		}
	}
	if warpMode == "awg" || warpMode == "auto" {
		options.AWG = chainWarpAwg(warp.Awg)
	}
	// Only shrink MTU when packets ride inside another proxy (extra security / explicit detour),
	// and only when the hop does not set one itself.
	if detour != "" && warp.MTU == 0 {
		switch warpMode {
		case "wg", "awg":
			options.MTU = chainNestedWireGuardMTU
		case "auto":
			// the MASQUE leg refuses an MTU below 1280, and the legs share the MTU
			options.MTU = chainNestedMASQUEMTU
		}
	}
	return option.Endpoint{Type: C.TypeWARP, Tag: tag, Options: options}
}

// chainWarpEndpointSource is where the WARP peer of the hop comes from. Older settings carried the
// choice in CleanIP as the words auto/default/random (a peer picked at random) and auto4/auto6 (a
// family); they now mean a scan, which picks by what answers instead of at random. With nothing
// set, the unblocker scans, as it always did (it dials the internet itself, so the scan is what
// finds a peer that gets through), and extra security takes the endpoint Cloudflare assigned. A
// hop behind a detour never scans: probing CIDRs through a leaf only measures the detour, and
// delays the first handshake, so it uses the assigned endpoint plus the endpoint's own failover.
// The address is the WireGuard peer: in masque mode the endpoint reads server as the MASQUE peer
// instead, so a custom address does not apply there and the hop's default source does.
func chainWarpEndpointSource(chainStatus string, warp ChainWarpOptions, detour, warpMode string) string {
	byDefault := WarpEndpointProfile
	if chainStatus == ChainStatusUnblocker {
		byDefault = WarpEndpointScan
	}
	source := strings.ToLower(strings.TrimSpace(warp.Endpoint))
	if source == "" {
		switch strings.ToLower(strings.TrimSpace(warp.CleanIP)) {
		case "":
			source = byDefault
		case "auto", "auto4", "auto6", "default", "random":
			source = WarpEndpointScan
		default:
			source = WarpEndpointCustom
		}
	}
	switch source {
	case WarpEndpointProfile, WarpEndpointScan, WarpEndpointCustom:
	case "auto", "auto4", "auto6", "default", "random":
		source = WarpEndpointScan
	default:
		source = byDefault // a source this build does not know
	}
	if source == WarpEndpointCustom && (warpMode == "masque" || strings.TrimSpace(warp.CleanIP) == "") {
		source = byDefault
	}
	if source == WarpEndpointScan && detour != "" {
		return WarpEndpointProfile
	}
	return source
}

// chainWarpScanner finds a working Cloudflare endpoint and a noise preset that gets through, on
// the scanner's defaults for whatever the hop leaves unset.
func chainWarpScanner(chainStatus string, scanner *ChainWarpScannerOptions) *option.WARPScannerOptions {
	options := &option.WARPScannerOptions{Enabled: true, SearchNoise: true}
	if scanner == nil {
		return options
	}
	options.EdgeSearchOptions = scanner.edgeSearch(chainStatus)
	if scanner.SearchNoise != nil {
		options.SearchNoise = *scanner.SearchNoise
	}
	options.MaxRTT = badoption.Duration(time.Duration(scanner.MaxRTT) * time.Millisecond)
	options.Timeout = badoption.Duration(time.Duration(scanner.Timeout) * time.Millisecond)
	options.Count = scanner.Count
	options.Concurrency = scanner.Concurrency
	options.Ports = scanner.Ports
	for _, cidr := range scanner.CIDRs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err != nil {
			continue // a range that does not parse is dropped, not fatal
		}
		options.CIDRs = append(options.CIDRs, prefix)
	}
	options.NoisePresets = scanner.NoisePresets
	return options
}

// chainWarpMasque builds the MASQUE tunnel of the hop. A detour cannot carry QUIC, so the tunnel
// runs HTTP/2 whatever the hop asks for.
func chainWarpMasque(masque *ChainWarpMasqueOptions, detour string) *option.WARPMASQUEOptions {
	options := &option.WARPMASQUEOptions{Enabled: true, VHTTP: "auto"}
	if masque == nil {
		if detour != "" {
			options.VHTTP = "h2"
		}
		return options
	}
	options.VHTTP = strings.TrimSpace(masque.VHTTP)
	options.ServerName = strings.TrimSpace(masque.ServerName)
	options.UseIPv6 = masque.UseIPv6
	options.UDPKeepalivePeriod = badoption.Duration(time.Duration(masque.UDPKeepalivePeriod) * time.Second)
	options.UDPInitialPacketSize = masque.UDPInitialPacketSize
	options.ReconnectDelay = badoption.Duration(time.Duration(masque.ReconnectDelay) * time.Second)
	options.TLS = chainWarpMasqueTLS(masque)
	if detour != "" {
		options.VHTTP = "h2"
	}
	return options
}

// chainWarpMasqueTLS maps the transport knobs onto the tunnel's TLS options.
func chainWarpMasqueTLS(masque *ChainWarpMasqueOptions) option.MASQUEOutboundTLSOptions {
	tlsOptions := option.MASQUEOutboundTLSOptions{
		Insecure:              masque.TLSInsecure,
		Fragment:              masque.TLSFragment,
		FragmentFallbackDelay: badoption.Duration(time.Duration(masque.TLSFragmentFallbackDelay) * time.Millisecond),
		RecordFragment:        masque.TLSRecordFragment,
	}
	if masque.ECHEnabled {
		tlsOptions.ECH = &option.OutboundECHOptions{
			Enabled:         true,
			QueryServerName: strings.TrimSpace(masque.ECHQueryServerName),
		}
		if config := strings.TrimSpace(masque.ECHConfig); config != "" {
			tlsOptions.ECH.Config = badoption.Listable[string]{config}
		}
	}
	return tlsOptions
}

// chainWarpAwg builds the AmneziaWG settings of the hop. All zero keeps the core's preset, which
// is what makes the tunnel look like something other than AmneziaWG.
func chainWarpAwg(awg *ChainWarpAwgOptions) *option.AwgOptions {
	if awg == nil || *awg == (ChainWarpAwgOptions{}) {
		return nil
	}
	return &option.AwgOptions{
		Jc:   awg.Jc,
		Jmin: awg.Jmin,
		Jmax: awg.Jmax,
		S1:   awg.S1,
		S2:   awg.S2,
		S3:   awg.S3,
		S4:   awg.S4,
		H1:   awg.H1,
		H2:   awg.H2,
		H3:   awg.H3,
		H4:   awg.H4,
		I1:   awg.I1,
		I2:   awg.I2,
		I3:   awg.I3,
		I4:   awg.I4,
		I5:   awg.I5,
	}
}

func chainWarpNoise(warp ChainWarpOptions) hiddify.NoiseOptions {
	parse := func(s string) hiddify.Range {
		var r hiddify.Range
		if s != "" {
			_ = r.UnmarshalJSON([]byte(strconv.Quote(s))) // invalid input leaves the range empty
		}
		return r
	}
	fake := hiddify.FakePacketOptions{
		Count: parse(warp.Noise),
		Size:  parse(warp.NoiseSize),
		Delay: parse(warp.NoiseDelay),
		Mode:  warp.NoiseMode,
	}
	fake.Enabled = fake.Count != (hiddify.Range{})
	if _, isPreset := hiddify.NoisePresetByName(warp.NoiseMode); isPreset {
		fake.Enabled = true // a preset brings its own count
	}
	return hiddify.NoiseOptions{FakePacket: fake}
}

func chainPsiphonOutbound(tag, chainStatus string, psiphon ChainPsiphonOptions, detour string) option.Outbound {
	region := psiphon.Region
	if region == psiphonRegionAuto {
		region = ""
	}
	// "hiddify": the Psiphon config embedded at build time (falls back to the defaults without it)
	options := &option.PsiphonOutboundOptions{Config: "hiddify", EgressRegion: region}
	options.Detour = detour
	if chainStatus == ChainStatusUnblocker { // behind a detour the core does not scan
		options.Scanner = chainPsiphonScanner(chainStatus, psiphon.Scanner)
	}
	return option.Outbound{Type: C.TypePsiphon, Tag: tag, Options: options}
}

// chainUsesPsiphon reports whether the active chain hop is Psiphon; Psiphon runs as one instance
// per process, so the profile's own Psiphon outbounds are dropped then.
func chainUsesPsiphon(opt *HiddifyOptions) bool {
	switch opt.ChainStatus {
	case ChainStatusExtraSecurity:
		return opt.ExtraSecurity.Mode == ChainModePsiphon || opt.ExtraSecurity.Mode == ChainModeSelectable
	case ChainStatusUnblocker:
		return opt.Unblocker.Mode == ChainModePsiphon || opt.Unblocker.Mode == ChainModeSelectable
	}
	return false
}

// redirectRemovedPsiphon points detours to a dropped Psiphon outbound at the kept one; without a
// kept one (a Psiphon chain hop replaces them) the detour is cleared: the outbound dials directly,
// or through the unblocker hop, which setChainHop adds to outbounds without a detour.
func redirectRemovedPsiphon(outbounds []option.Outbound, endpoints []option.Endpoint, removed map[string]bool, kept string) {
	if len(removed) == 0 {
		return
	}
	redirect := func(options any) {
		wrapper, ok := options.(option.DialerOptionsWrapper)
		if !ok {
			return
		}
		dialer := wrapper.TakeDialerOptions()
		if !removed[dialer.Detour] {
			return
		}
		dialer.Detour = kept
		wrapper.ReplaceDialerOptions(dialer)
	}
	for i := range outbounds {
		redirect(outbounds[i].Options)
	}
	for i := range endpoints {
		redirect(endpoints[i].Options)
	}
}

// chainPsiphonScanner is the CDN edge scanner of the unblocker hop; nil keeps the core's default
// (on, as the hop has no detour).
func chainPsiphonScanner(chainStatus string, scanner *ChainPsiphonScannerOptions) *option.PsiphonScannerOptions {
	if scanner == nil {
		return nil
	}
	options := &option.PsiphonScannerOptions{
		Enabled:           scanner.Enabled,
		Sets:              scanner.Sets,
		ServerNames:       scanner.ServerNames,
		Candidates:        scanner.Candidates,
		Count:             scanner.Count,
		Concurrency:       scanner.Concurrency,
		Timeout:           badoption.Duration(time.Duration(scanner.Timeout) * time.Millisecond),
		MaxRTT:            badoption.Duration(time.Duration(scanner.MaxRTT) * time.Millisecond),
		EdgeSearchOptions: scanner.edgeSearch(chainStatus),
	}
	for _, cidr := range scanner.CIDRs {
		if prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr)); err == nil && prefix.Addr().Is4() {
			options.CIDRs = append(options.CIDRs, prefix)
		}
	}
	return options
}
