package config

import (
	"fmt"
	"io"

	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
	E "github.com/sagernet/sing/common/exceptions"
)

const (
	serviceURL    = "http://localhost:18020"
	startEndpoint = "/start"
	stopEndpoint  = "/stop"
)

func isSupportedOS() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "linux"
}
func ActivateTunnelService(opt ConfigOptions) (bool, error) {
	if !isSupportedOS() {
		return false, E.New("Unsupported OS: " + runtime.GOOS)
	}
	
	go startTunnelRequest(opt, true)
	return true, nil
}

func DeactivateTunnelService() (bool, error) {
	if !isSupportedOS() {
		return true, nil
	}

	go stopTunnelRequest()
	return true, nil
}

func startTunnelRequest(opt ConfigOptions, installService bool) (bool, error) {
	params := map[string]interface{}{
		"Ipv6":                   opt.IPv6Mode == option.DomainStrategy(dns.DomainStrategyUseIPv4),
		"ServerPort":             opt.InboundOptions.MixedPort,
		"StrictRoute":            opt.InboundOptions.StrictRoute,
		"EndpointIndependentNat": true,
		"Stack":                  opt.InboundOptions.TUNStack,
	}

	values := url.Values{}
	for key, value := range params {
		values.Add(key, fmt.Sprint(value))
	}

	url := fmt.Sprintf("%s%s?%s", serviceURL, startEndpoint, values.Encode())
	fmt.Printf("URL: %s\n", url)
	response, err := http.Get(url)
	if err != nil {
		if installService {
			return runTunnelService(opt)
		}
		return false, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	fmt.Printf("Response Code: %d %s. Response Body: %s Error:%v\n", response.StatusCode, response.Status, body, err)
	if err != nil || response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Unexpected Status Code: %d %s. Response Body: %s error:%v", response.StatusCode, response.Status, body, err)
	}

	return true, nil
}

func stopTunnelRequest() (bool, error) {
	response, err := http.Get(serviceURL + stopEndpoint)
	if err != nil {
		return false, fmt.Errorf("HTTP Request Error: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	fmt.Printf("Response Code: %d %s. Response Body: %s Error:%v\n", response.StatusCode, response.Status, body, err)
	if err != nil || response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Unexpected Status Code: %d %s. Response Body: %s error:%v", response.StatusCode, response.Status, body, err)
	}

	return true, nil
}

func runTunnelService(opt ConfigOptions) (bool, error) {
	executablePath := getTunnelServicePath()
	fmt.Printf("Executable path is %s",executablePath)
	out, err := ExecuteCmd(executablePath, "install")
	fmt.Println("Shell command executed:", out, err)
	return startTunnelRequest(opt, false)
}

func getTunnelServicePath() string {
	var fullPath string
	exePath, _ := os.Executable()
	binFolder := filepath.Dir(exePath)
	switch runtime.GOOS {
	case "windows":
		fullPath = "HiddifyService.exe"
	case "darwin":
		fallthrough
	default:
		fullPath = "HiddifyService"
	}

	abspath,_:=filepath.Abs(filepath.Join(binFolder, fullPath))
	return abspath
}
