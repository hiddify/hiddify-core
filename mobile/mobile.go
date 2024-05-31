package mobile

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/config"
	_ "github.com/sagernet/gomobile"
	"github.com/sagernet/sing-box/option"
)

func Setup() error {
	// return config.StartGRPCServer(7078)
	return nil
}

func Parse(path string, tempPath string, debug bool) error {
	config, err := config.ParseConfig(tempPath, debug)
	if err != nil {
		return err
	}
	return os.WriteFile(path, config, 0644)
}

func BuildConfig(path string, configOptionsJson string) (string, error) {
	os.Chdir(filepath.Dir(path))
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
	if configOptions.Warp.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(configOptions.Warp.WireguardConfigStr), &configOptions.Warp.WireguardConfig)
		if err != nil {
			return "", err
		}
	}

	if configOptions.Warp2.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(configOptions.Warp2.WireguardConfigStr), &configOptions.Warp2.WireguardConfig)
		if err != nil {
			return "", err
		}
	}

	return config.BuildConfigJson(*configOptions, options)
}

func GenerateWarpConfig(licenseKey string, accountId string, accessToken string) (string, error) {
	return config.GenerateWarpAccount(licenseKey, accountId, accessToken)
}
