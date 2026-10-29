package config

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/hiddify/ray2sing/ray2sing"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/batch"
	SJ "github.com/sagernet/sing/common/json"
	"github.com/xmdhs/clash2singbox/convert"
	clash2singmodel "github.com/xmdhs/clash2singbox/model"
	"github.com/xmdhs/clash2singbox/model/clash"
	"gopkg.in/yaml.v3"
)

//go:embed config.json.template
var configByte []byte

func ReadContent(ctx context.Context, opt *ReadOptions) ([]byte, error) {
	if opt.Content == "" {
		contentBytes, err := os.ReadFile(opt.Path)
		if err != nil {
			return nil, err
		}
		opt.Content = string(contentBytes)
	}
	return []byte(opt.Content), nil
}

func ParseConfig(ctx context.Context, opt *ReadOptions, debug bool, configOpt *HiddifyOptions, fullConfig bool) (*option.Options, error) {
	content, err := ReadContent(ctx, opt)
	if err != nil {
		return nil, err
	}
	return parseConfigContent(ctx, content, debug, nil, false)
}

func ParseConfigBytes(ctx context.Context, opt *ReadOptions, debug bool, configOpt *HiddifyOptions, fullConfig bool) ([]byte, error) {

	options, err := ParseConfig(ctx, opt, debug, configOpt, fullConfig)
	if err != nil {
		return nil, err
	}

	return options.MarshalJSONContext(ctx)

}
func parseConfigContent(ctx context.Context, content []byte, debug bool, configOpt *HiddifyOptions, fullConfig bool) (*option.Options, error) {
	if configOpt == nil {
		configOpt = DefaultHiddifyOptions()
	}

	// The content is JSON, YAML or text. JSON and YAML are read as sing-box / hiddify-core first, then as Clash,
	// then as Xray; text is share links.
	var structured any
	isStructured := json.NewDecoder(SJ.NewCommentFilter(bytes.NewReader(content))).Decode(&structured) == nil
	if !isStructured {
		var yamlValue any
		if yaml.Unmarshal(content, &yamlValue) == nil && isYAMLConfig(yamlValue) {
			structured, isStructured = yamlValue, true
		}
	}
	if !isStructured {
		return parseShareLinks(ctx, content, configOpt)
	}

	options, singboxErr := parseSingboxContent(ctx, structured, configOpt, fullConfig)
	if singboxErr == nil {
		return options, nil
	}
	if options, isClash, err := parseClashContent(ctx, content, configOpt); isClash {
		return options, err
	}
	if links, ok := xrayJSONShareLinks(structured); ok {
		fmt.Printf("Convert using xray json\n")
		v2ray, err := ray2sing.Ray2SingboxOptions(ctx, links, configOpt.UseXrayCoreWhenPossible)
		if err != nil {
			return nil, fmt.Errorf("[XrayJsonParser] %w", err)
		}
		return patchConfigOptions(ctx, v2ray, "XrayJsonParser", configOpt)
	}
	if options, err := parseShareLinks(ctx, content, configOpt); err == nil {
		return options, nil
	}
	return nil, singboxErr
}

func parseShareLinks(ctx context.Context, content []byte, configOpt *HiddifyOptions) (*option.Options, error) {
	v2ray, err := ray2sing.Ray2SingboxOptions(ctx, string(content), configOpt.UseXrayCoreWhenPossible)
	if err != nil {
		return nil, fmt.Errorf("unable to determine config format")
	}
	return patchConfigOptions(ctx, v2ray, "V2rayParser", configOpt)
}

// isYAMLConfig reports YAML that can be a config: a map, or a list of maps. A WireGuard conf
// starting with "[Interface]" is also a YAML list, of one string.
func isYAMLConfig(value any) bool {
	switch value := value.(type) {
	case map[string]any:
		return true
	case []any:
		for _, item := range value {
			if _, ok := item.(map[string]any); !ok {
				return false
			}
		}
		return len(value) > 0
	}
	return false
}

// parseSingboxContent reads a decoded sing-box / hiddify-core config: a config, one outbound or a list of them.
func parseSingboxContent(ctx context.Context, structured any, configOpt *HiddifyOptions, fullConfig bool) (*option.Options, error) {
	fmt.Printf("Convert using singbox\n")
	var jsonObj map[string]interface{} = make(map[string]interface{})
	if tmpJsonObj, ok := structured.(map[string]interface{}); ok {
		if tmpJsonObj["outbounds"] == nil && tmpJsonObj["endpoints"] == nil {
			// a single outbound object (was wrapping the empty result itself)
			jsonObj["outbounds"] = []interface{}{tmpJsonObj}
		} else {
			if fullConfig || (configOpt != nil && configOpt.EnableFullConfig) {
				jsonObj = tmpJsonObj
			} else {
				if tmpJsonObj["outbounds"] != nil {
					jsonObj["outbounds"] = tmpJsonObj["outbounds"]
				}
				if tmpJsonObj["endpoints"] != nil {
					jsonObj["endpoints"] = tmpJsonObj["endpoints"]
				}
			}
		}
	} else if jsonArray, ok := structured.([]interface{}); ok {
		if len(jsonArray) == 0 {
			return nil, fmt.Errorf("[SingboxParser] no outbounds found")
		}
		outbounds := make([]interface{}, 0, len(jsonArray))
		for i, item := range jsonArray {
			itemObj, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("[SingboxParser] outbound at index %d is %T, expected a json object", i, item)
			}
			outbounds = append(outbounds, itemObj)
		}
		jsonObj["outbounds"] = outbounds
	} else {
		return nil, fmt.Errorf("[SingboxParser] incorrect json format: expected a json object or an array of outbound objects, got %T", structured)
	}

	// the legacy WireGuard outbound was removed from sing-box: use the WireGuard endpoint
	outbounds, _ := jsonObj["outbounds"].([]interface{})
	endpoints, _ := jsonObj["endpoints"].([]interface{})
	if outbounds, endpoints = ray2sing.MoveLegacyWireGuardOutbounds(outbounds, endpoints); len(endpoints) > 0 {
		jsonObj["outbounds"] = outbounds
		jsonObj["endpoints"] = endpoints
	}

	newContent, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("[SingboxParser] %w", err)
	}

	return patchConfigStr(ctx, newContent, "SingboxParser", configOpt)
}

// parseClashContent reads a Clash config; isClash is false when content has no Clash proxies.
func parseClashContent(ctx context.Context, content []byte, configOpt *HiddifyOptions) (options *option.Options, isClash bool, err error) {
	clashObj := clash.Clash{}
	if err := yaml.Unmarshal(content, &clashObj); err != nil || clashObj.Proxies == nil {
		return nil, false, nil
	}
	fmt.Printf("Convert using clash\n")
	if len(clashObj.Proxies) == 0 {
		return nil, true, fmt.Errorf("[ClashParser] no outbounds found")
	}
	converted, endpoints, err := convert.Clash2sing(clashObj, clash2singmodel.SINGLATEST)
	if err != nil {
		return nil, true, fmt.Errorf("[ClashParser] converting clash to sing-box error: %w", err)
	}
	output := configByte
	output, err = convert.Patch(output, converted, endpoints, "", "", nil)
	if err != nil {
		return nil, true, fmt.Errorf("[ClashParser] patching clash config error: %w", err)
	}
	options, err = patchConfigStr(ctx, output, "ClashParser", configOpt)
	return options, true, err
}

func patchConfigStr(ctx context.Context, content []byte, name string, configOpt *HiddifyOptions) (*option.Options, error) {
	options := option.Options{}
	err := options.UnmarshalJSONContext(ctx, content)

	if err != nil {
		return nil, fmt.Errorf("[SingboxParser] unmarshal error: %w", err)
	}

	return patchConfigOptions(ctx, &options, name, configOpt)
}
func patchConfigOptions(ctx context.Context, options *option.Options, name string, configOpt *HiddifyOptions) (*option.Options, error) {
	// dns outbounds were removed in sing-box 1.13; drop them instead of keeping invalid placeholders
	options.Outbounds = slices.DeleteFunc(options.Outbounds, isDNSOutbound)
	b, _ := batch.New(ctx, batch.WithConcurrencyNum[*option.Endpoint](2))
	for _, base := range options.Endpoints {
		out := base
		b.Go(base.Tag, func() (*option.Endpoint, error) {
			err := patchWarp(&out, configOpt, false, nil)
			if err != nil {
				return nil, fmt.Errorf("[Warp] patch warp error: %w", err)
			}
			// options.Outbounds[i] = base
			return &out, nil
		})
	}
	if res, err := b.WaitAndGetResult(); err != nil {
		return nil, err
	} else {
		for i, base := range options.Endpoints {
			options.Endpoints[i] = *res[base.Tag].Value
		}
	}

	// fmt.Printf("%s\n", content)
	return validateResult(ctx, options, name)
}

func validateResult(ctx context.Context, options *option.Options, name string) (*option.Options, error) {
	err := libbox.CheckConfigOptions(options)
	if err != nil {
		return nil, fmt.Errorf("[%s] invalid sing-box config: %w", name, err)
	}
	return options, nil
}
