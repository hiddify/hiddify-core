package config

import (
	"encoding/json"
	"fmt"
	"strings"

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
		// } else {
		// 	delete(obj, "multiplex")
	}
	return obj
}

func patchOutboundTLSTricks(base option.Outbound, configOpt HiddifyOptions, obj outboundMap) outboundMap {
	if base.Type == C.TypeSelector || base.Type == C.TypeURLTest || base.Type == C.TypeBlock || base.Type == C.TypeDNS {
		return obj
	}
	if isOutboundReality(base) {
		return obj
	}

	var tls *option.OutboundTLSOptions
	var transport *option.V2RayTransportOptions
	if base.VLESSOptions.OutboundTLSOptionsContainer.TLS != nil {
		tls = base.VLESSOptions.OutboundTLSOptionsContainer.TLS
		transport = base.VLESSOptions.Transport
	} else if base.TrojanOptions.OutboundTLSOptionsContainer.TLS != nil {
		tls = base.TrojanOptions.OutboundTLSOptionsContainer.TLS
		transport = base.TrojanOptions.Transport
	} else if base.VMessOptions.OutboundTLSOptionsContainer.TLS != nil {
		tls = base.VMessOptions.OutboundTLSOptionsContainer.TLS
		transport = base.VMessOptions.Transport
	}

	if base.Type == C.TypeDirect {
		return patchOutboundFragment(base, configOpt, obj)
	}

	if tls == nil || !tls.Enabled || transport == nil {
		return obj
	}

	if transport.Type != C.V2RayTransportTypeWebsocket && transport.Type != C.V2RayTransportTypeGRPC && transport.Type != C.V2RayTransportTypeHTTPUpgrade {
		return obj
	}

	if outtls, ok := obj["tls"].(map[string]interface{}); ok {
		obj = patchOutboundFragment(base, configOpt, obj)
		tlsTricks := tls.TLSTricks
		if tlsTricks == nil {
			tlsTricks = &option.TLSTricksOptions{}
		}
		tlsTricks.MixedCaseSNI = tlsTricks.MixedCaseSNI || configOpt.TLSTricks.MixedSNICase

		if configOpt.TLSTricks.EnablePadding {
			tlsTricks.PaddingMode = "random"
			tlsTricks.PaddingSize = configOpt.TLSTricks.PaddingSize
			// fmt.Printf("--------------------%+v----%+v", tlsTricks.PaddingSize, configOpt)
			outtls["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": "custom",
			}
		}

		outtls["tls_tricks"] = tlsTricks
		// if tlsTricks.MixedCaseSNI || tlsTricks.PaddingMode != "" {
		// 	// } else {
		// 	// 	tls["tls_tricks"] = nil
		// }
		// fmt.Printf("-------%+v------------- ", tlsTricks)
	}
	return obj
}

func patchOutboundFragment(base option.Outbound, configOpt HiddifyOptions, obj outboundMap) outboundMap {
	if configOpt.TLSTricks.EnableFragment {
		obj["tcp_fast_open"] = false
		obj["tls_fragment"] = option.TLSFragmentOptions{
			Enabled: configOpt.TLSTricks.EnableFragment,
			Size:    configOpt.TLSTricks.FragmentSize,
			Sleep:   configOpt.TLSTricks.FragmentSleep,
		}

	}

	return obj
}

func isOutboundReality(base option.Outbound) bool {
	// this function checks reality status ONLY FOR VLESS.
	// Some other protocols can also use reality, but it's discouraged as stated in the reality document
	if base.Type != C.TypeVLESS {
		return false
	}
	if base.VLESSOptions.OutboundTLSOptionsContainer.TLS == nil {
		return false
	}
	if base.VLESSOptions.OutboundTLSOptionsContainer.TLS.Reality == nil {
		return false
	}
	return base.VLESSOptions.OutboundTLSOptionsContainer.TLS.Reality.Enabled
}

func patchOutbound(base option.Outbound, configOpt HiddifyOptions, staticIpsDns map[string][]string) (*option.Outbound, error) {
	formatErr := func(err error) error {
		return fmt.Errorf("error patching outbound[%s][%s]: %w", base.Tag, base.Type, err)
	}
	err := patchWarp(&base, &configOpt, true, staticIpsDns)
	if err != nil {
		return nil, formatErr(err)
	}
	var outbound option.Outbound

	jsonData, err := base.MarshalJSON()
	if err != nil {
		return nil, formatErr(err)
	}

	var obj outboundMap
	err = json.Unmarshal(jsonData, &obj)
	if err != nil {
		return nil, formatErr(err)
	}

	obj = patchOutboundTLSTricks(base, configOpt, obj)

	switch base.Type {
	case C.TypeVMess, C.TypeVLESS, C.TypeTrojan, C.TypeShadowsocks:
		obj = patchOutboundMux(base, configOpt, obj)
	}
	obj = patchOutboundXray(base, configOpt, obj)
	modifiedJson, err := json.Marshal(obj)
	if err != nil {
		return nil, formatErr(err)
	}

	err = outbound.UnmarshalJSON(modifiedJson)
	if err != nil {
		return nil, formatErr(err)
	}

	return &outbound, nil
}

func patchOutboundXray(base option.Outbound, configOpt HiddifyOptions, obj outboundMap) outboundMap {
	if base.Type == C.TypeXray {
		// Handle alternative key "xray_outbound_raw"
		if rawConfig, exists := obj["xray_outbound_raw"]; exists && rawConfig != nil && rawConfig != "" {
			obj["xconfig"] = rawConfig
		}

		// Ensure "xconfig" exists and properly structured
		xconfig, ok := obj["xconfig"].(map[string]any)
		if !ok {
			return obj // Return early if the structure is invalid
		}

		// Ensure "outbounds" key exists within "xconfig"
		if _, exists := xconfig["outbounds"]; !exists {
			xconfig = map[string]any{"outbounds": xconfig}
		}

		if configOpt.TLSTricks.EnableFragment {
			// TODO
			// if obj["xray_fragment"] == nil || obj["xray_fragment"].(map[string]any)["packets"] == "" {
			// 	obj["xray_fragment"] = map[string]any{
			// 		"packets":  "tlshello",
			// 		"length":   configOpt.TLSTricks.FragmentSize,
			// 		"interval": configOpt.TLSTricks.FragmentSleep,
			// 	}
			// }
		}

		dnsConfig, ok := xconfig["dns"].(map[string]any)
		if !ok {
			dnsConfig = map[string]any{}
		}

		// Ensure "servers" key exists and is a slice
		servers, ok := dnsConfig["servers"].([]any)
		if !ok {
			servers = []any{}
		}

		// // Ensure "servers" key exists and is a slice
		// hosts, ok := dnsConfig["hosts"].([]any)
		// if !ok {
		// 	hosts = []any{}
		// }
		// for _, host := range base.DNSOptions. {
		// 	hosts = append(hosts, host)
		// }

		// Append remote DNS address
		servers = append(servers, configOpt.DNSOptions.RemoteDnsAddress)

		// Append direct DNS address if not "local"
		if configOpt.DNSOptions.DirectDnsAddress != "local" {
			dnsAdd := configOpt.DNSOptions.DirectDnsAddress
			if strings.Contains(dnsAdd, "://") {
				dnsAdd = strings.Replace(dnsAdd, "://", "+local://", 1)
			} else {
				dnsAdd = "udp+local://" + dnsAdd
			}
			servers = append(servers, dnsAdd)
		}

		// Append fallback DNS server
		if configOpt.DNSOptions.DirectDnsAddress != "1.1.1.1" {
			servers = append(servers, "udp+local://1.1.1.1")
		}

		// Update "servers" key in "dns"
		dnsConfig["servers"] = servers
		dnsConfig["disableFallback"] = false
		xconfig["dns"] = dnsConfig
		obj["xconfig"] = xconfig
		obj["xdebug"] = configOpt.LogLevel == "debug" || configOpt.LogLevel == "trace"
	}

	return obj
}

// func (o outboundMap) transportType() string {
// 	if transport, ok := o["transport"].(map[string]interface{}); ok {
// 		if transportType, ok := transport["type"].(string); ok {
// 			return transportType
// 		}
// 	}
// 	return ""
// }
