package shared

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/er888kh/go-subconverter/converter"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/xmdhs/clash2singbox/convert"
	"github.com/xmdhs/clash2singbox/model/clash"
	"gopkg.in/yaml.v3"
)

//go:embed config.json.template
var configByte []byte

func ParseConfig(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	clash_conf, err := parseV2rayFormat(content)
	if err == nil {
		content = clash_conf
	}

	config, err := parseClash(content)
	if err != nil {
		config = content
	}
	err = libbox.CheckConfig(string(config))
	if err != nil {
		return err
	}
	err = os.WriteFile(path, config, 0777)
	if err != nil {
		return err
	}
	return nil
}
func parseV2rayFormat(content []byte) ([]byte, error) {
	clash_conf, err := converter.GenerateProxies(string(content), "meta")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil, err
	}
	fmt.Printf("v2ray to clash: %v\n", clash_conf)
	return []byte(clash_conf), nil
}
func parseClash(content []byte) ([]byte, error) {
	clashConfig := clash.Clash{}
	err := yaml.Unmarshal(content, &clashConfig)
	if err != nil {
		fmt.Printf("unmarshal error %s", err)
		return nil, err
	}
	sbConfig, err := convert.Clash2sing(clashConfig)
	if err != nil {
		fmt.Printf("convert error %s", err)
		return nil, err
	}

	output := configByte
	output, err = convert.Patch(output, sbConfig, "", "", nil)
	if err != nil {
		fmt.Printf("patch error %s", err)
		return nil, err
	}
	return output, nil
}
