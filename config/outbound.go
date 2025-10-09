package config

import (
    "fmt"

    C "github.com/sagernet/sing-box/constant"
    "github.com/sagernet/sing-box/option"
)

type outboundMap map[string]interface{}

func patchOutboundMux(base option.Outbound, configOpt HiddifyOptions, obj outboundMap) outboundMap {
    if configOpt.Mux.Enable {
        multiplex := option.OutboundMultiplexOptions{
            Enabled:    true,
            Padding:    configOpt.Mux.Padding,
            MaxStreams: configOpt.Mux.MaxStreams,
            Protocol:   configOpt.Mux.Protocol,
        }
        obj["multiplex"] = multiplex
    }
    return obj
}

func patchOutboundTLSTricks(base option.Outbound, configOpt HiddifyOptions, obj outboundMap) outboundMap {
    if base.Type == C.TypeSelector || base.Type == C.TypeURLTest || base.Type == C.TypeBlock || base.Type == C.TypeDNS {
        return obj
    }
    if isOutboundReality(base, obj) {
        return obj
    }

    // Xray-specific fragment is no longer supported here under v1.13.
    // Determine TLS and transport from JSON object for v1.13
    var tlsEnabled bool
    if tlsMap, ok := obj["tls"].(map[string]any); ok {
        if enabled, ok := tlsMap["enabled"].(bool); ok {
            tlsEnabled = enabled
        }
    }
    transportType := ""
    if trMap, ok := obj["transport"].(map[string]any); ok {
        if t, ok := trMap["type"].(string); ok {
            transportType = t
        }
    }
    if !tlsEnabled {
        return obj
    }
    if transportType != C.V2RayTransportTypeWebsocket && transportType != C.V2RayTransportTypeGRPC && transportType != C.V2RayTransportTypeHTTPUpgrade {
        return obj
    }

    if outtls, ok := obj["tls"].(map[string]interface{}); ok {
        obj = patchOutboundFragment(base, configOpt, obj)
        var tlsTricks map[string]any
        if existing, ok := outtls["tls_tricks"].(map[string]any); ok {
            tlsTricks = existing
        } else {
            tlsTricks = map[string]any{}
        }
        if configOpt.TLSTricks.MixedSNICase {
            tlsTricks["mixed_case_sni"] = true
        }

        if configOpt.TLSTricks.EnablePadding {
            tlsTricks["padding_mode"] = "random"
            tlsTricks["padding_size"] = configOpt.TLSTricks.PaddingSize
            outtls["utls"] = map[string]interface{}{
                "enabled":     true,
                "fingerprint": "custom",
            }
        }

        outtls["tls_tricks"] = tlsTricks
        // if tlsTricks.MixedCaseSNI || tlsTricks.PaddingMode != "" {
        //     // } else {
        //     //  tls["tls_tricks"] = nil
        // }
    }
    return obj
}

func patchOutboundFragment(base option.Outbound, configOpt HiddifyOptions, obj outboundMap) outboundMap {
    if configOpt.TLSTricks.EnableFragment {
        obj["tcp_fast_open"] = false
        obj["tls_fragment"] = map[string]any{
            "enabled": configOpt.TLSTricks.EnableFragment,
            "size":    configOpt.TLSTricks.FragmentSize,
            "sleep":   configOpt.TLSTricks.FragmentSleep,
        }
    }

    return obj
}

func isOutboundReality(base option.Outbound, obj outboundMap) bool {
    // this function checks reality status ONLY FOR VLESS.
    // Some other protocols can also use reality, but it's discouraged as stated in the reality document
    if base.Type != C.TypeVLESS {
        return false
    }
    if tlsMap, ok := obj["tls"].(map[string]any); ok {
        if reality, ok := tlsMap["reality"].(map[string]any); ok {
            if enabled, ok := reality["enabled"].(bool); ok {
                return enabled
            }
        }
    }
    return false
}

func patchOutbound(base option.Outbound, configOpt HiddifyOptions, staticIpsDns map[string][]string) (*option.Outbound, string, error) {
    formatErr := func(err error) error {
        return fmt.Errorf("error patching outbound[%s][%s]: %w", base.Tag, base.Type, err)
    }
    if err := patchWarp(&base, &configOpt, true, staticIpsDns); err != nil {
        return nil, "", formatErr(err)
    }
    // Apply MUX options in typed way for common outbounds
    if configOpt.Mux.Enable {
        mux := option.OutboundMultiplexOptions{
            Enabled:    true,
            Padding:    configOpt.Mux.Padding,
            MaxStreams: configOpt.Mux.MaxStreams,
            Protocol:   configOpt.Mux.Protocol,
        }
        switch base.Type {
        case C.TypeVMess:
            if o, ok := base.Options.(option.VMessOutboundOptions); ok {
                o.Multiplex = &mux
                base.Options = o
            }
        case C.TypeVLESS:
            if o, ok := base.Options.(option.VLESSOutboundOptions); ok {
                o.Multiplex = &mux
                base.Options = o
            }
        case C.TypeTrojan:
            if o, ok := base.Options.(option.TrojanOutboundOptions); ok {
                o.Multiplex = &mux
                base.Options = o
            }
        }
    }

    // Apply TLS tweaks in typed way when TLS is already present and enabled
    applyTLSTweaks := func(tls *option.OutboundTLSOptions) {
        if tls == nil || !tls.Enabled {
            return
        }
        if configOpt.TLSTricks.EnableFragment {
            tls.Fragment = true
        }
        if configOpt.TLSTricks.MixedSNICase || configOpt.TLSTricks.EnablePadding {
            // No direct mixed-case or padding size fields in v1.13; enable uTLS with custom fp as closest behavior
            if tls.UTLS == nil {
                tls.UTLS = &option.OutboundUTLSOptions{}
            }
            tls.UTLS.Enabled = true
            if tls.UTLS.Fingerprint == "" {
                tls.UTLS.Fingerprint = "custom"
            }
        }
        if configOpt.TLSTricks.EnableECH {
            if tls.ECH == nil {
                tls.ECH = &option.OutboundECHOptions{}
            }
            tls.ECH.Enabled = true
            // Prefer inline config if provided; otherwise use path
            if configOpt.TLSTricks.ECHConfig != "" {
                tls.ECH.Config = []string{configOpt.TLSTricks.ECHConfig}
                tls.ECH.ConfigPath = ""
            } else if configOpt.TLSTricks.ECHConfigPath != "" {
                tls.ECH.Config = nil
                tls.ECH.ConfigPath = configOpt.TLSTricks.ECHConfigPath
            }
        }
        if configOpt.TLSTricks.EnableReality {
            if tls.Reality == nil {
                tls.Reality = &option.OutboundRealityOptions{}
            }
            tls.Reality.Enabled = true
            if configOpt.TLSTricks.RealityPublicKey != "" {
                tls.Reality.PublicKey = configOpt.TLSTricks.RealityPublicKey
            }
            if configOpt.TLSTricks.RealityShortID != "" {
                tls.Reality.ShortID = configOpt.TLSTricks.RealityShortID
            }
        }
    }
    switch base.Type {
    case C.TypeVMess:
        if o, ok := base.Options.(option.VMessOutboundOptions); ok {
            if o.OutboundTLSOptionsContainer.TLS != nil {
                applyTLSTweaks(o.OutboundTLSOptionsContainer.TLS)
                base.Options = o
            }
        }
    case C.TypeVLESS:
        if o, ok := base.Options.(option.VLESSOutboundOptions); ok {
            if o.OutboundTLSOptionsContainer.TLS != nil {
                applyTLSTweaks(o.OutboundTLSOptionsContainer.TLS)
                base.Options = o
            }
        }
    case C.TypeTrojan:
        if o, ok := base.Options.(option.TrojanOutboundOptions); ok {
            if o.OutboundTLSOptionsContainer.TLS != nil {
                applyTLSTweaks(o.OutboundTLSOptionsContainer.TLS)
                base.Options = o
            }
        }
    }
    return &base, "", nil
}
// 		}
// 	}
// 	return ""
// }
