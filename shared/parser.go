package shared

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/hiddify/ray2sing/ray2sing"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/xmdhs/clash2singbox/convert"
	"github.com/xmdhs/clash2singbox/model/clash"
	"gopkg.in/yaml.v3"
)

//go:embed config.json.template
var configByte []byte

func ParseConfig(path string, tempPath string, debug bool) error {
	content, err := os.ReadFile(tempPath)
	if err != nil {
		return err
	}
	config, err := parseV2rayFormat(content)
	if err != nil {
		config = content
		config, err = parseClash(content)
		if err != nil {
			config = content
		}
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
	singconf, err := ray2sing.Ray2Singbox(string(content))
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil, err
	}
	return []byte(singconf), nil
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
