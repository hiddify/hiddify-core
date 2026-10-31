package config

import (
	"fmt"
	"strconv"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
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

type ChainWarpOptions struct {
	LicenseKey string `json:"license-key,omitempty"`
	CleanIP    string `json:"clean-ip,omitempty"`
	CleanPort  uint16 `json:"clean-port,omitempty"`
	Noise      string `json:"noise,omitempty"`
	NoiseSize  string `json:"noise-size,omitempty"`
	NoiseDelay string `json:"noise-delay,omitempty"`
	NoiseMode  string `json:"noise-mode,omitempty"`
}

type ChainPsiphonOptions struct {
	Region           string `json:"region,omitempty"`
	ConduitPairingID string `json:"conduit-pairing-id,omitempty"`
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
	for i := range *outbounds {
		chainDialer((*outbounds)[i].Options, (*outbounds)[i].Tag, tag)
	}
	for i := range *endpoints {
		chainDialer((*endpoints)[i].Options, (*endpoints)[i].Tag, tag)
	}
	return nil
}

// addChainHop adds the hop of one mode, named tag.
func addChainHop(status string, hop ChainHopOptions, mode, tag, detour string, outbounds *[]option.Outbound, endpoints *[]option.Endpoint) error {
	switch mode {
	case ChainModeWarp, ChainModeWarpAwg, ChainModeWarpMasque, ChainModeWarpAuto:
		*endpoints = append(*endpoints, chainWarpEndpoint(tag, status, hop.Warp, detour, mode))
	case ChainModePsiphon:
		*outbounds = append(*outbounds, chainPsiphonOutbound(tag, hop.Psiphon, detour))
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
				Tolerance:                 100,
				ToleranceRatio:            0.1,
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
func chainDialer(options any, outboundTag, hopTag string) {
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
	wrapper.ReplaceDialerOptions(dialer)
}

// chainWarpModes are the tunnels of the warp endpoint (its mode option) for the WARP chain modes.
var chainWarpModes = map[string]string{
	ChainModeWarp:       "wg",
	ChainModeWarpAwg:    "awg",
	ChainModeWarpMasque: "masque",
	ChainModeWarpAuto:   "auto", // races WireGuard, AmneziaWG and MASQUE and keeps the fastest
}

// chainWarpEndpoint is a warp endpoint running the tunnel of mode. awg and auto take the AmneziaWG
// settings of the core's default noise preset; behind a detour (TCP-only, it cannot carry QUIC)
// MASQUE runs over HTTP/2.
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
	if warpMode != "masque" { // in masque mode server is the MASQUE peer, not the WireGuard one
		if warp.CleanIP != "" {
			options.Server = warp.CleanIP
		}
		options.ServerPort = warp.CleanPort
	}
	if warpMode == "masque" || warpMode == "auto" {
		options.MASQUE = &option.WARPMASQUEOptions{Enabled: true}
		if detour != "" {
			options.MASQUE.VHTTP = "h2"
		}
	}
	// Only shrink MTU when packets ride inside another proxy (extra security / explicit detour).
	if detour != "" {
		switch warpMode {
		case "wg", "awg":
			options.MTU = chainNestedWireGuardMTU
		case "auto":
			// the MASQUE leg refuses an MTU below 1280, and the legs share the MTU
			options.MTU = chainNestedMASQUEMTU
		}
		// Nested Extra Security: no scanner / search_noise — probing CIDRs through a leaf
		// is slow and delays the first handshake forever.
	} else if options.Server == "" {
		// Unblocker dials directly: scan IPs and noise (skipped when CleanIP/Server is set).
		options.Scanner = chainUnblockerScanner()
	}
	return option.Endpoint{Type: C.TypeWARP, Tag: tag, Options: options}
}

// chainUnblockerScanner finds a working Cloudflare endpoint and a noise preset that gets
// through. Extra Security must never use this (nested CIDR probes hang through a leaf).
func chainUnblockerScanner() *option.WARPScannerOptions {
	return &option.WARPScannerOptions{Enabled: true, SearchNoise: true}
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

func chainPsiphonOutbound(tag string, psiphon ChainPsiphonOptions, detour string) option.Outbound {
	region := psiphon.Region
	if region == psiphonRegionAuto {
		region = ""
	}
	// "hiddify": the Psiphon config embedded at build time (falls back to the defaults without it)
	options := &option.PsiphonOutboundOptions{Config: "hiddify", EgressRegion: region}
	options.Detour = detour
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
