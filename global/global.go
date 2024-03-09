package global

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hiddify/libcore/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

var box *libbox.BoxService
var configOptions *config.ConfigOptions
var activeConfigPath *string
var logFactory *log.Factory

func setup(baseDir string, workingDir string, tempDir string, statusPort int64, debug bool) error {
	Setup(baseDir, workingDir, tempDir)
	statusPropagationPort = statusPort

	var defaultWriter io.Writer
	if !debug {
		defaultWriter = io.Discard
	}
	factory, err := log.New(
		log.Options{
			DefaultWriter: defaultWriter,
			BaseTime:      time.Now(),
			Observable:    false,
		})
	if err != nil {
		return err
	}
	logFactory = &factory
	return nil
}

func parse(path string, tempPath string, debug bool) error {
	config, err := config.ParseConfig(tempPath, debug)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, config, 0644)
	if err != nil {
		return err
	}
	return nil
}

func changeConfigOptions(configOptionsJson string) error {
	configOptions = &config.ConfigOptions{}
	err := json.Unmarshal([]byte(configOptionsJson), configOptions)

	if err != nil {
		return err
	}
	if configOptions.Warp.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(configOptions.Warp.WireguardConfigStr), &configOptions.Warp.WireguardConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateConfig(path string) (string, error) {
	config, err := generateConfigFromFile(path, *configOptions)
	if err != nil {
		return "", err
	}
	return config, nil
}

func generateConfigFromFile(path string, configOpt config.ConfigOptions) (string, error) {
	os.Chdir(filepath.Dir(path))
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	options, err := parseConfig(string(content))
	if err != nil {
		return "", err
	}
	config, err := config.BuildConfigJson(configOpt, options)
	if err != nil {
		return "", err
	}
	return config, nil
}

func start(configPath string, disableMemoryLimit bool) error {
	if status != Stopped {
		return nil
	}
	propagateStatus(Starting)

	activeConfigPath = &configPath

	libbox.SetMemoryLimit(!disableMemoryLimit)
	err := startService(false)
	if err != nil {
		return err
	}
	return nil
}

func startService(delayStart bool) error {
	content, err := os.ReadFile(*activeConfigPath)
	if err != nil {
		return stopAndAlert(EmptyConfiguration, err)
	}
	options, err := parseConfig(string(content))
	if err != nil {
		return stopAndAlert(EmptyConfiguration, err)
	}
	os.Chdir(filepath.Dir(*activeConfigPath))
	var patchedOptions *option.Options
	patchedOptions, err = config.BuildConfig(*configOptions, options)
	if err != nil {
		return fmt.Errorf("error building config: %w", err)
	}

	config.SaveCurrentConfig(filepath.Join(sWorkingPath, "tunnel-current-config.json"), *patchedOptions)

	err = startCommandServer(*logFactory)
	if err != nil {
		return stopAndAlert(StartCommandServer, err)
	}

	instance, err := NewService(*patchedOptions)
	if err != nil {
		return stopAndAlert(CreateService, err)
	}

	if delayStart {
		time.Sleep(250 * time.Millisecond)
	}

	err = instance.Start()
	if err != nil {
		return stopAndAlert(StartService, err)
	}
	box = instance
	commandServer.SetService(box)

	propagateStatus(Started)
	return nil
}

func stop() error {
	if status != Started {
		return nil
	}
	if box == nil {
		return errors.New("instance not found")
	}
	config.DeactivateTunnelService()
	propagateStatus(Stopping)

	commandServer.SetService(nil)
	err := box.Close()
	if err != nil {
		return err
	}
	box = nil

	err = commandServer.Close()
	if err != nil {
		return err
	}
	commandServer = nil
	propagateStatus(Stopped)

	return nil
}

func restart(configPath string, disableMemoryLimit bool) error {
	log.Debug("[Service] Restarting")

	if status != Started {
		return nil
	}
	if box == nil {
		return errors.New("instance not found")
	}

	err := stop()
	if err != nil {
		return err
	}

	propagateStatus(Starting)

	time.Sleep(250 * time.Millisecond)

	activeConfigPath = &configPath
	libbox.SetMemoryLimit(!disableMemoryLimit)
	gErr := startService(false)
	if gErr != nil {
		return gErr
	}
	return nil
}

func startCommandClient(command int, port int64) error {
	err := StartCommand(int32(command), port, *logFactory)
	if err != nil {
		return err
	}
	return nil
}

func stopCommandClient(command int) error {
	err := StopCommand(int32(command))
	if err != nil {
		return err
	}
	return nil
}

func selectOutbound(groupTag string, outboundTag string) error {
	err := libbox.NewStandaloneCommandClient().SelectOutbound(groupTag, outboundTag)
	if err != nil {
		return err
	}
	return nil
}

func urlTest(groupTag string) error {
	err := libbox.NewStandaloneCommandClient().URLTest(groupTag)
	if err != nil {
		return err
	}
	return nil
}

func StartServiceC(delayStart bool, content string) error {
	if box != nil {
		return errors.New("instance already started")
	}
	options, err := parseConfig(content)
	// if err != nil {
	// 	return stopAndAlert(EmptyConfiguration, err)
	// }
	// configOptions = &config.ConfigOptions{}
	// patchedOptions, err := config.BuildConfig(*configOptions, options)

	// options = *patchedOptions

	// config.SaveCurrentConfig(filepath.Join(sWorkingPath, "custom-current-config.json"), options)
	// if err != nil {
	// 	fmt.Printf("Error in saving config: %v\n", err)
	// 	return err
	// }

	// err = startCommandServer(*logFactory)
	// if err != nil {
	// 	return err
	// }

	instance, err := NewService(options)
	if err != nil {
		// 	return stopAndAlert(CreateService, err)
		return err
	}

	// if delayStart {
	// 	time.Sleep(250 * time.Millisecond)
	// }

	err = instance.Start()
	if err != nil {
		// return stopAndAlert(StartService, err)
		fmt.Printf("String Service Error: %v\n", err)
		return err
	}
	box = instance
	// commandServer.SetService(box)

	status = Started
	return nil
}
func StopServiceC() error {
	// if status != Started {
	// 	return errors.New("instance not started")
	// }
	config.DeactivateTunnelService()
	if box == nil {
		return errors.New("instance not found")
	}
	// propagateStatus(Stopping)
	err := box.Close()
	// commandServer.SetService(nil)
	if err != nil {
		return err
	}
	box = nil

	// err = commandServer.Close()
	// if err != nil {
	// 	return err
	// }
	commandServer = nil
	status = Stopped

	return nil
}

func SetupC(baseDir string, workDir string, tempDir string, debug bool) error {
	err := os.MkdirAll(baseDir, 0644)
	if err != nil {
		return err
	}
	err = os.MkdirAll(workDir, 0644)
	if err != nil {
		return err
	}
	err = os.MkdirAll(tempDir, 0644)
	if err != nil {
		return err
	}
	Setup(baseDir, workDir, tempDir)
	var defaultWriter io.Writer
	if !debug {
		defaultWriter = io.Discard
	}
	factory, err := log.New(
		log.Options{
			DefaultWriter: defaultWriter,
			BaseTime:      time.Now(),
			Observable:    false,
		})
	if err != nil {
		return err
	}
	logFactory = &factory
	return nil
}

func MakeConfig(Ipv6 bool, ServerPort int, StrictRoute bool, EndpointIndependentNat bool, Stack string) string {
	var ipv6 string
	if Ipv6 {
		ipv6 = `      "inet6_address": "fdfe:dcba:9876::1/126",`
	} else {
		ipv6 = ""
	}
	base := `{
		"inbounds": [
		  {
			"type": "tun",
			"tag": "tun-in",
			"interface_name": "HiddifyTunnel",
			"inet4_address": "172.19.0.1/30",
			` + ipv6 + `
			"mtu": 9000,
			"auto_route": true,
			"strict_route": ` + fmt.Sprintf("%t", StrictRoute) + `,
			"endpoint_independent_nat": ` + fmt.Sprintf("%t", EndpointIndependentNat) + `,
			"stack": "` + Stack + `"
		  }
		],
		"outbounds": [
		  {
			"type": "socks",
			"tag": "socks-out",
			"server": "127.0.0.1",
			"server_port": ` + fmt.Sprintf("%d", ServerPort) + `,
			"version": "5"
		  }
		]
	  }`

	return base
}

func WriteParameters(Ipv6 bool, ServerPort int, StrictRoute bool, EndpointIndependentNat bool, Stack string) error {
	parameters := fmt.Sprintf("%t,%d,%t,%t,%s", Ipv6, ServerPort, StrictRoute, EndpointIndependentNat, Stack)
	err := os.WriteFile("parameters.config", []byte(parameters), 0644)
	if err != nil {
		return err
	}
	return nil
}
func ReadParameters() (bool, int, bool, bool, string, error) {
	Data, err := os.ReadFile("parameters.config")
	if err != nil {
		return false, 0, false, false, "", err
	}
	DataSlice := strings.Split(string(Data), ",")
	Ipv6, _ := strconv.ParseBool(DataSlice[0])
	ServerPort, _ := strconv.Atoi(DataSlice[1])
	StrictRoute, _ := strconv.ParseBool(DataSlice[2])
	EndpointIndependentNat, _ := strconv.ParseBool(DataSlice[3])
	stack := DataSlice[4]
	return Ipv6, ServerPort, StrictRoute, EndpointIndependentNat, stack, nil
}
