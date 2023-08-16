package shared

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/xmdhs/clash2singbox/convert"
	"github.com/xmdhs/clash2singbox/model/clash"
	"gopkg.in/yaml.v3"
)

func ConvertToSingbox(path string, options ConfigTemplateOptions) ([]byte, error) {
	clashConfig := clash.Clash{}
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(fileContent, &clashConfig)
	if err != nil {
		fmt.Printf("unmarshal error %s", err)
		return nil, err
	}
	sbConfig, err := convert.Clash2sing(clashConfig)
	if err != nil {
		fmt.Printf("convert error %s", err)
		return nil, err
	}

	output := defaultTemplate(options)
	output, err = convert.Patch(output, sbConfig, "", "", nil)
	if err != nil {
		fmt.Printf("patch error %s", err)
		return output, err
	}

	return output, nil
}
