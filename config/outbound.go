package config

import (
	"encoding/json"
	"fmt"
	"net"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

type outboundMap map[string]interface{}

func patchOutboundMux(base option.Outbound, configOpt ConfigOptions, obj outboundMap) outboundMap {
	if configOpt.EnableMux {
		multiplex := option.OutboundMultiplexOptions{
			Enabled:    true,
			Padding:    configOpt.MuxPadding,
			MaxStreams: configOpt.MaxStreams,
			Protocol:   configOpt.MuxProtocol,
		}
		obj["multiplex"] = multiplex
	} else {
		delete(obj, "multiplex")
	}
	return obj
}

func patchOutboundTLSTricks(base option.Outbound, configOpt ConfigOptions, obj outboundMap) outboundMap {

	obj = patchOutboundFragment(base, configOpt, obj)
	if tls, ok := obj["tls"].(map[string]interface{}); ok {
		tlsTricks := option.TLSTricksOptions{
			MixedCaseSNI: configOpt.TLSTricks.EnableMixedSNICase,
		}

		if configOpt.TLSTricks.EnablePadding {
			tlsTricks.PaddingMode = "random"
			tlsTricks.PaddingSize = configOpt.TLSTricks.PaddingSize
		}

		if tlsTricks.MixedCaseSNI || tlsTricks.PaddingMode != "" {
			tls["tls_tricks"] = tlsTricks
		} else {
			tls["tls_tricks"] = nil
		}
	}
	return obj
}

func patchOutboundFragment(base option.Outbound, configOpt ConfigOptions, obj outboundMap) outboundMap {
	if configOpt.EnableFragment {
		tlsFragment := option.TLSFragmentOptions{
			Enabled: configOpt.TLSTricks.EnableFragment,
			Size:    configOpt.TLSTricks.FragmentSize,
			Sleep:   configOpt.TLSTricks.FragmentSleep,
		}
		obj["tls_fragment"] = tlsFragment
	} else {
		obj["tls_fragment"] = nil
	}
	return obj
}

func isOutboundReality(base option.Outbound) bool {
	// this function checks reality status ONLY FOR VLESS.
	// Some other protocols can also use reality, but it's discouraged as stated in the reality document
	isReality := false
	switch base.Type {
	case C.TypeVLESS:
		if base.VLESSOptions.TLS.Reality != nil {
			isReality = base.VLESSOptions.TLS.Reality.Enabled
		}
	}
	return isReality
}

func patchOutbound(base option.Outbound, configOpt ConfigOptions) (*option.Outbound, string, error) {
	var serverDomain string
	var outbound option.Outbound

	formatErr := func(err error) error {
		return fmt.Errorf("error patching outbound[%s][%s]: %w", base.Tag, base.Type, err)
	}

	jsonData, err := base.MarshalJSON()
	if err != nil {
		return nil, "", formatErr(err)
	}

	var obj outboundMap
	err = json.Unmarshal(jsonData, &obj)
	if err != nil {
		return nil, "", formatErr(err)
	}

	if server, ok := obj["server"].(string); ok {
		if server != "" && net.ParseIP(server) == nil {
			serverDomain = fmt.Sprintf("full:%s", server)
		}
	}
	if !(base.Type == C.TypeSelector || base.Type == C.TypeURLTest || base.Type == C.TypeBlock || base.Type == C.TypeDNS || isOutboundReality(base)) {
		obj = patchOutboundTLSTricks(base, configOpt, obj)
	}
	switch base.Type {
	case C.TypeVMess, C.TypeVLESS, C.TypeTrojan, C.TypeShadowsocks:
		obj = patchOutboundMux(base, configOpt, obj)
	}

	modifiedJson, err := json.Marshal(obj)
	if err != nil {
		return nil, "", formatErr(err)
	}

	err = outbound.UnmarshalJSON(modifiedJson)
	if err != nil {
		return nil, "", formatErr(err)
	}

	return &outbound, serverDomain, nil
}

// func (o outboundMap) transportType() string {
// 	if transport, ok := o["transport"].(map[string]interface{}); ok {
// 		if transportType, ok := transport["type"].(string); ok {
// 			return transportType
// 		}
// 	}
// 	return ""
// }
