package global

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiddify/libcore/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	err = os.WriteFile(path, config, 0777)
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

	config.SaveCurrentConfig(sWorkingPath, *patchedOptions)

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
	options, err := parseConfig(content)
	if err != nil {
		return stopAndAlert(EmptyConfiguration, err)
	}
	configOptions = &config.ConfigOptions{}
	patchedOptions, err := config.BuildConfig(*configOptions, options)

	options = *patchedOptions

	err = config.SaveCurrentConfig(sWorkingPath, options)
	if err != nil {
		return err
	}

	err = startCommandServer(*logFactory)
	if err != nil {
		return stopAndAlert(StartCommandServer, err)
	}

	instance, err := NewService(options)
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

func StopService() error {
	if status != Started {
		return nil
	}
	if box == nil {
		return errors.New("instance not found")
	}

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

func SetupC(baseDir string, workDir string, tempDir string, debug bool) error {
	err := os.MkdirAll("./bin", 600)
	if err != nil {
		return err
	}
	err = os.MkdirAll("./work", 600)
	if err != nil {
		return err
	}
	err = os.MkdirAll("./temp", 600)
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
		ipv6 = "      \"inet6_address\": \"fdfe:dcba:9876::1/126\",\n"
	} else {
		ipv6 = ""
	}
	base := "{\n  \"inbounds\": [\n    {\n      \"type\": \"tun\",\n      \"tag\": \"tun-in\",\n      \"interface_name\": \"tun0\",\n      \"inet4_address\": \"172.19.0.1/30\",\n" + ipv6 + "      \"mtu\": 9000,\n      \"auto_route\": true,\n      \"strict_route\": " + fmt.Sprintf("%t", StrictRoute) + ",\n      \"endpoint_independent_nat\": " + fmt.Sprintf("%t", EndpointIndependentNat) + ",\n      \"stack\": \"" + Stack + "\"\n    }],\n  \"outbounds\": [\n    {\n      \"type\": \"socks\",\n      \"tag\": \"socks-out\",\n      \"server\": \"127.0.0.1\",\n      \"server_port\": " + fmt.Sprintf("%d", ServerPort) + ",\n      \"version\": \"5\"\n    }\n  ]\n}\n"
	return base
}

func WriteParameters(Ipv6 bool, ServerPort int, StrictRoute bool, EndpointIndependentNat bool, Stack string) error {
	parameters := fmt.Sprintf("%t,%d,%t,%t,%s", Ipv6, ServerPort, StrictRoute, EndpointIndependentNat, Stack)
	err := os.WriteFile("bin/parameters.config", []byte(parameters), 600)
	if err != nil {
		return err
	}
	return nil
}
func ReadParameters() (bool, int, bool, bool, string, error) {
	Data, err := os.ReadFile("bin/parameters.config")
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
