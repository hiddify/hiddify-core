package shared

import (
	"fmt"
	"os"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/xmdhs/clash2singbox/convert"
	"github.com/xmdhs/clash2singbox/model/clash"
	"gopkg.in/yaml.v3"
)

func ParseConfig(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
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

	output := defaultTemplate(ConfigTemplateOptions{})
	output, err = convert.Patch(output, sbConfig, "", "", nil)
	if err != nil {
		fmt.Printf("patch error %s", err)
		return nil, err
	}
	return output, nil
}
