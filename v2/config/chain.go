package config

import (
	"fmt"
	"strconv"

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

	ChainModeWarp    = "warp"
	ChainModePsiphon = "psiphon"
	ChainModeProfile = "profile"

	ChainExtraSecurityTag = "🔒 Extra Security §hide§"
	ChainUnblockerTag     = "🔓 Unblocker §hide§"

	psiphonRegionAuto = "AUTO"
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
		hop, tag, detour = opt.ExtraSecurity, ChainExtraSecurityTag, OutboundSelectTag
	case ChainStatusUnblocker:
		hop, tag, detour = opt.Unblocker, ChainUnblockerTag, dialDetour(OutboundDirectTag)
	default:
		return fmt.Errorf("unknown chain status: %s", opt.ChainStatus)
	}

	switch hop.Mode {
	case ChainModeWarp:
		*endpoints = append(*endpoints, chainWarpEndpoint(tag, opt.ChainStatus, hop.Warp, detour))
	case ChainModePsiphon:
		*outbounds = append(*outbounds, chainPsiphonOutbound(tag, hop.Psiphon, detour))
	case ChainModeProfile:
		// refuse instead of silently connecting without the requested hop
		return fmt.Errorf("%s: profile mode is not supported yet", opt.ChainStatus)
	default:
		return fmt.Errorf("%s: unknown mode: %s", opt.ChainStatus, hop.Mode)
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

// chainDialer makes a main outbound/endpoint dial through the unblocker hop, unless it already
// dials through another detour (it is then not the first hop of its own chain) or is the hop itself.
func chainDialer(options any, outboundTag, hopTag string) {
	if outboundTag == hopTag || outboundTag == WARPConfigTag {
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

func chainWarpEndpoint(tag, chainStatus string, warp ChainWarpOptions, detour string) option.Endpoint {
	options := &option.WARPEndpointOptions{
		UniqueIdentifier: "chain-" + chainStatus,
		Profile: option.WARPProfile{
			License: warp.LicenseKey,
			Detour:  detour, // register the WARP account along the same path
		},
		Noise: chainWarpNoise(warp),
		MTU:   1280,
	}
	options.Detour = detour
	if warp.CleanIP != "" {
		options.Server = warp.CleanIP
	}
	options.ServerPort = warp.CleanPort
	return option.Endpoint{Type: C.TypeWARP, Tag: tag, Options: options}
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
