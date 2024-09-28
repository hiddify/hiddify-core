package sdk

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"

	"github.com/hiddify/hiddify-core/config"
	v2 "github.com/hiddify/hiddify-core/v2"
	"github.com/sagernet/sing-box/option"
)

func RunInstance(hiddifySettings *config.HiddifyOptions, singconfig *option.Options) (*v2.HiddifyService, error) {
	return v2.RunInstance(hiddifySettings, singconfig)
}

func ParseConfig(hiddifySettings *config.HiddifyOptions, configStr string) (*option.Options, error) {
	if hiddifySettings == nil {
		hiddifySettings = config.DefaultHiddifyOptions()
	}
	if strings.HasPrefix(configStr, "http://") || strings.HasPrefix(configStr, "https://") {
		client := &http.Client{}
		configPath := strings.Split(configStr, "\n")[0]
		// Create a new request
		req, err := http.NewRequest("GET", configPath, nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			return nil, err
		}
		req.Header.Set("User-Agent", "HiddifyNext/2.3.1 ("+runtime.GOOS+") like ClashMeta v2ray sing-box")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error making GET request:", err)
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read config body: %w", err)
		}
		configStr = string(body)
	}
	return config.ParseConfigContentToOptions(configStr, true, hiddifySettings, false)
}
