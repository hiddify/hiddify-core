package mobile

import (
	"encoding/json"
	"os"

	"github.com/hiddify/libcore/config"
	_ "github.com/sagernet/gomobile"
	"github.com/sagernet/sing-box/option"
)

func Parse(path string, tempPath string, debug bool) error {
	config, err := config.ParseConfig(tempPath, debug)
	if err != nil {
		return err
	}
	return os.WriteFile(path, config, 0777)
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
	configOptions := &config.ConfigOptions{}
	err = json.Unmarshal([]byte(configOptionsJson), configOptions)
	if err != nil {
		return "", nil
	}
	return config.BuildConfigJson(*configOptions, options)
}
