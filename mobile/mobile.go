package mobile

import (
	"encoding/json"
	"os"

	"github.com/hiddify/libcore/shared"
	_ "github.com/sagernet/gomobile/event/key"
	"github.com/sagernet/sing-box/option"
)

func Parse(path string, tempPath string, debug bool) error {
	return shared.ParseConfig(path, tempPath, debug)
}

func BuildConfig(path string, configOptionsJson string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var options option.Options
	err = options.UnmarshalJSON(fileContent)
	if err != nil {
		return "", err
	}
	configOptions := &shared.ConfigOptions{}
	err = json.Unmarshal([]byte(configOptionsJson), configOptions)
	if err != nil {
		return "", nil
	}
	return shared.BuildConfigJson(*configOptions, options)
}
