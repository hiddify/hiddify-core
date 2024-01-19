package config

import (
	"encoding/json"
	"fmt"
	"net"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

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

	var obj map[string]interface{}
	err = json.Unmarshal(jsonData, &obj)
	if err != nil {
		return nil, "", formatErr(err)
	}

	if server, ok := obj["server"].(string); ok {
		if server != "" && net.ParseIP(server) == nil {
			serverDomain = fmt.Sprintf("full:%s", server)
		}
	}

	if !(base.Type == C.TypeSelector || base.Type == C.TypeURLTest || base.Type == C.TypeBlock || base.Type == C.TypeDNS) {
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
