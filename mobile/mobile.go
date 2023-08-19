package mobile

import (
	"encoding/json"
	"os"

	"github.com/hiddify/libcore/shared"
	_ "github.com/sagernet/gomobile/event/key"
	"github.com/sagernet/sing-box/option"
)

func Parse(path string) error {
	return shared.ParseConfig(path)
}

func ApplyOverrides(path string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var options option.Options
	err = options.UnmarshalJSON(fileContent)
	if err != nil {
		return "", err
	}
	overrides := shared.ConfigOverrides{ExcludeTunInbound: false, IncludeMixedInbound: false, IncludeLogOutput: false, LogLevel: "", IncludeLogTimestamp: false, ClashApiPort: 9090}
	options = shared.ApplyOverrides(options, overrides)
	config, err := json.Marshal(options)
	return string(config), err
}
