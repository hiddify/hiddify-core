package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hiddify/hiddify-core/bridge"
	"github.com/hiddify/hiddify-core/config"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var (
	Box              *libbox.BoxService
	configOptions    *config.ConfigOptions
	activeConfigPath string
	coreLogFactory   log.Factory
	useFlutterBridge bool = true
)

func StopAndAlert(msgType pb.MessageType, message string) {
	SetCoreStatus(pb.CoreState_STOPPED, msgType, message)
	config.DeactivateTunnelService()
	if oldCommandServer != nil {
		oldCommandServer.SetService(nil)
	}
	if Box != nil {
		Box.Close()
		Box = nil
	}
	if oldCommandServer != nil {
		oldCommandServer.Close()
	}
	if useFlutterBridge {
		alert := msgType.String()
		msg, _ := json.Marshal(StatusMessage{Status: convert2OldState(CoreState), Alert: &alert, Message: &message})
		bridge.SendStringToPort(statusPropagationPort, string(msg))
	}
}

func (s *CoreService) Start(ctx context.Context, in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	return Start(in)
}

func Start(in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	defer config.DeferPanicToError("start", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})
	Log(pb.LogLevel_INFO, pb.LogType_CORE, "Starting")
	if CoreState != pb.CoreState_STOPPED {
		Log(pb.LogLevel_INFO, pb.LogType_CORE, "Starting0000")
		Stop()
		// return &pb.CoreInfoResponse{
		// 	CoreState:   CoreState,
		// 	MessageType: pb.MessageType_INSTANCE_NOT_STOPPED,
		// }, fmt.Errorf("instance not stopped")
	}
	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Starting Core")
	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")
	libbox.SetMemoryLimit(!in.DisableMemoryLimit)
	resp, err := StartService(in)
	return resp, err
}
func (s *CoreService) StartService(ctx context.Context, in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	return StartService(in)
}
func StartService(in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Starting Core Service")
	content := in.ConfigContent
	if content == "" {

		activeConfigPath = in.ConfigPath
		fileContent, err := os.ReadFile(activeConfigPath)
		if err != nil {
			Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
			resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_ERROR_READING_CONFIG, err.Error())
			StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
			return &resp, err
		}
		content = string(fileContent)
	}
	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Parsing Config")

	parsedContent, err := parseConfig(content)
	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Parsed")

	if err != nil {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_ERROR_PARSING_CONFIG, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
		return &resp, err
	}
	if !in.EnableRawConfig {
		Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Building config")
		parsedContent_tmp, err := config.BuildConfig(*configOptions, parsedContent)
		if err != nil {
			Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
			resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_ERROR_BUILDING_CONFIG, err.Error())
			StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
			return &resp, err
		}
		parsedContent = *parsedContent_tmp
	}
	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Saving config")
	currentBuildConfigPath := filepath.Join(sWorkingPath, "current-config.json")
	config.SaveCurrentConfig(currentBuildConfigPath, parsedContent)
	if activeConfigPath == "" {
		activeConfigPath = currentBuildConfigPath
	}
	if in.EnableOldCommandServer {
		Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Starting Command Server")
		err = startCommandServer()
		if err != nil {
			Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
			resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_COMMAND_SERVER, err.Error())
			StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
			return &resp, err
		}
	}

	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Stating Service ")
	instance, err := NewService(parsedContent)

	if err != nil {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_CREATE_SERVICE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
		return &resp, err
	}
	Log(pb.LogLevel_DEBUG, pb.LogType_CORE, "Service.. started")
	if in.DelayStart {
		<-time.After(250 * time.Millisecond)
	}

	err = instance.Start()
	if err != nil {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_SERVICE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
		return &resp, err
	}
	Box = instance
	if in.EnableOldCommandServer {
		oldCommandServer.SetService(Box)
	}

	resp := SetCoreStatus(pb.CoreState_STARTED, pb.MessageType_EMPTY, "")
	return &resp, nil

}

func (s *CoreService) Parse(ctx context.Context, in *pb.ParseRequest) (*pb.ParseResponse, error) {
	return Parse(in)
}
func Parse(in *pb.ParseRequest) (*pb.ParseResponse, error) {
	defer config.DeferPanicToError("parse", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CONFIG, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})

	content := in.Content
	if in.TempPath != "" {
		contentBytes, err := os.ReadFile(in.TempPath)
		content = string(contentBytes)
		os.Chdir(filepath.Dir(in.ConfigPath))
		if err != nil {
			return nil, err
		}

	}

	config, err := config.ParseConfigContent(content, true, nil, false)
	if err != nil {
		return &pb.ParseResponse{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}
	if in.ConfigPath != "" {
		err = os.WriteFile(in.ConfigPath, config, 0644)
		if err != nil {
			return &pb.ParseResponse{
				ResponseCode: pb.ResponseCode_FAILED,
				Message:      err.Error(),
			}, err
		}
	}
	return &pb.ParseResponse{
		ResponseCode: pb.ResponseCode_OK,
		Content:      string(config),
		Message:      "",
	}, err
}

func (s *CoreService) ChangeConfigOptions(ctx context.Context, in *pb.ChangeConfigOptionsRequest) (*pb.CoreInfoResponse, error) {
	return ChangeConfigOptions(in)
}

func ChangeConfigOptions(in *pb.ChangeConfigOptionsRequest) (*pb.CoreInfoResponse, error) {
	configOptions = &config.ConfigOptions{}
	err := json.Unmarshal([]byte(in.ConfigOptionsJson), configOptions)
	if err != nil {
		return nil, err
	}
	if configOptions.Warp.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(configOptions.Warp.WireguardConfigStr), &configOptions.Warp.WireguardConfig)
		if err != nil {
			return nil, err
		}
	}
	if configOptions.Warp2.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(configOptions.Warp2.WireguardConfigStr), &configOptions.Warp2.WireguardConfig)
		if err != nil {
			return nil, err
		}
	}
	return &pb.CoreInfoResponse{}, nil
}
func (s *CoreService) GenerateConfig(ctx context.Context, in *pb.GenerateConfigRequest) (*pb.GenerateConfigResponse, error) {
	return GenerateConfig(in)
}
func GenerateConfig(in *pb.GenerateConfigRequest) (*pb.GenerateConfigResponse, error) {
	defer config.DeferPanicToError("generateConfig", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CONFIG, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})
	if configOptions == nil {
		configOptions = config.DefaultConfigOptions()
	}
	config, err := generateConfigFromFile(in.Path, *configOptions)
	if err != nil {
		return nil, err
	}
	return &pb.GenerateConfigResponse{
		ConfigContent: config,
	}, nil
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

func (s *CoreService) Stop(ctx context.Context, empty *pb.Empty) (*pb.CoreInfoResponse, error) {
	return Stop()
}
func Stop() (*pb.CoreInfoResponse, error) {
	defer config.DeferPanicToError("stop", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})

	if CoreState != pb.CoreState_STARTED {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, "Core is not started")
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_STARTED,
			Message:     "instance is not started",
		}, fmt.Errorf("instance not started")
	}
	if Box == nil {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_FOUND,
			Message:     "instance is not found",
		}, fmt.Errorf("instance not found")
	}
	SetCoreStatus(pb.CoreState_STOPPING, pb.MessageType_EMPTY, "")
	config.DeactivateTunnelService()
	if oldCommandServer != nil {
		oldCommandServer.SetService(nil)
	}

	err := Box.Close()
	if err != nil {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_UNEXPECTED_ERROR,
			Message:     "Error while stopping the service.",
		}, fmt.Errorf("Error while stopping the service.")
	}
	Box = nil
	if oldCommandServer != nil {
		err = oldCommandServer.Close()
		if err != nil {
			return &pb.CoreInfoResponse{
				CoreState:   CoreState,
				MessageType: pb.MessageType_UNEXPECTED_ERROR,
				Message:     "Error while Closing the comand server.",
			}, fmt.Errorf("error while Closing the comand server.")
		}
		oldCommandServer = nil
	}
	resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_EMPTY, "")
	return &resp, nil

}
func (s *CoreService) Restart(ctx context.Context, in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	return Restart(in)
}
func Restart(in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	defer config.DeferPanicToError("restart", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})
	log.Debug("[Service] Restarting")

	if CoreState != pb.CoreState_STARTED {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_STARTED,
			Message:     "instance is not started",
		}, fmt.Errorf("instance not started")
	}
	if Box == nil {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_FOUND,
			Message:     "instance is not found",
		}, fmt.Errorf("instance not found")
	}

	resp, err := Stop()
	if err != nil {
		return resp, err
	}

	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")
	<-time.After(250 * time.Millisecond)

	libbox.SetMemoryLimit(!in.DisableMemoryLimit)
	resp, gErr := StartService(in)
	return resp, gErr
}
