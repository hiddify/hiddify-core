package hcore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hiddify/hiddify-core/bridge"
	"github.com/hiddify/hiddify-core/config"
	common "github.com/hiddify/hiddify-core/v2/common"
	"github.com/hiddify/hiddify-core/v2/db"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

var (
	Box            *libbox.BoxService
	HiddifyOptions *config.HiddifyOptions
	// activeConfigPath string
	coreLogFactory   log.Factory
	useFlutterBridge bool = true
)

func StopAndAlert(msgType MessageType, message string) {
	SetCoreStatus(CoreStates_STOPPED, msgType, message)
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

func (s *CoreService) Start(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return Start(in)
}

func Start(in *StartRequest) (*CoreInfoResponse, error) {
	defer config.DeferPanicToError("start", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	})
	Log(LogLevel_INFO, LogType_CORE, "Starting")
	if CoreState != CoreStates_STOPPED {
		Log(LogLevel_INFO, LogType_CORE, "Starting0000")
		Stop()
		// return &CoreInfoResponse{
		// 	CoreState:   CoreState,
		// 	MessageType: MessageType_INSTANCE_NOT_STOPPED,
		// }, fmt.Errorf("instance not stopped")
	}
	Log(LogLevel_DEBUG, LogType_CORE, "Starting Core")
	SetCoreStatus(CoreStates_STARTING, MessageType_EMPTY, "")
	libbox.SetMemoryLimit(!in.DisableMemoryLimit)
	resp, err := StartService(in)
	return resp, err
}

func (s *CoreService) StartService(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return StartService(in)
}

func errorWrapper(state MessageType, err error) (*CoreInfoResponse, error) {
	Log(LogLevel_FATAL, LogType_CORE, err.Error())
	StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	return SetCoreStatus(CoreStates_STOPPED, state, err.Error()), err
}

func StartWithPlatformInterface(in *StartRequest, platformInterface libbox.PlatformInterface) (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("start", func(recovered_err error) {
		coreResponse, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered_err)
		<-time.After(5 * time.Second)
	})

	Log(LogLevel_DEBUG, LogType_CORE, "Starting Core Service")
	json, err := BuildConfigJson(in)
	if err != nil {
		return errorWrapper(MessageType_ERROR_BUILDING_CONFIG, err)
	}
	Log(LogLevel_DEBUG, LogType_CORE, "Saving config")
	// currentBuildConfigPath := filepath.Join(sWorkingPath, "current-config.json")
	// config.SaveCurrentConfig(currentBuildConfigPath, *parsedContent)
	// activeConfigPath = currentBuildConfigPath

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("Starting Service json %++v, platformInterface %++v", json, platformInterface))
	instance, err := libbox.NewService(json, &HiddifyPlatformInterface{
		platform: platformInterface,
	})
	if err != nil {
		return errorWrapper(MessageType_CREATE_SERVICE, err)
	}

	Log(LogLevel_DEBUG, LogType_CORE, "Stating Service with delay ?", in.DelayStart)
	if in.DelayStart {
		<-time.After(250 * time.Millisecond)
	}

	err = instance.Start()
	if err != nil {
		return errorWrapper(MessageType_START_SERVICE, err)
	}
	Box = instance
	if in.EnableOldCommandServer {
		Log(LogLevel_DEBUG, LogType_CORE, "Starting Command Server")
		if err := startCommandServer(); err != nil {
			return errorWrapper(MessageType_START_COMMAND_SERVER, err)
		}
		oldCommandServer.SetService(Box)
	}

	return SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, ""), nil
}

func BuildConfigJson(in *StartRequest) (string, error) {
	Log(LogLevel_DEBUG, LogType_CORE, "Stating Service ")

	parsedContent, err := BuildConfig(in)
	if err != nil {
		return "", err
	}
	return config.ToJson(*parsedContent)
}

func BuildConfig(in *StartRequest) (*option.Options, error) {
	content := in.ConfigContent
	if content == "" {
		fileContent, err := os.ReadFile(in.ConfigPath)
		if err != nil {
			return nil, err
		}
		content = string(fileContent)
	}

	Log(LogLevel_DEBUG, LogType_CORE, "Parsing Config")

	parsedContent, err := readOptions(content)
	Log(LogLevel_DEBUG, LogType_CORE, "Parsed")

	if err != nil {
		return nil, err
	}

	if !in.EnableRawConfig {
		Log(LogLevel_DEBUG, LogType_CORE, "Building config "+fmt.Sprintf("%++v", HiddifyOptions))
		return config.BuildConfig(*HiddifyOptions, parsedContent)

	}

	return &parsedContent, nil
}

func StartService(in *StartRequest) (*CoreInfoResponse, error) {
	Log(LogLevel_DEBUG, LogType_CORE, "Starting Core Service")
	content := in.ConfigContent
	if content == "" {

		fileContent, err := os.ReadFile(in.ConfigPath)
		if err != nil {
			Log(LogLevel_FATAL, LogType_CORE, err.Error())
			resp := SetCoreStatus(CoreStates_STOPPED, MessageType_ERROR_READING_CONFIG, err.Error())
			StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
			return resp, err
		}
		content = string(fileContent)
	}
	Log(LogLevel_DEBUG, LogType_CORE, "Parsing Config")

	parsedContent, err := readOptions(content)
	Log(LogLevel_DEBUG, LogType_CORE, "Parsed")

	if err != nil {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		resp := SetCoreStatus(CoreStates_STOPPED, MessageType_ERROR_PARSING_CONFIG, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
		return resp, err
	}
	if !in.EnableRawConfig {
		Log(LogLevel_DEBUG, LogType_CORE, "Building config")
		parsedContent_tmp, err := config.BuildConfig(*HiddifyOptions, parsedContent)
		if err != nil {
			Log(LogLevel_FATAL, LogType_CORE, err.Error())
			resp := SetCoreStatus(CoreStates_STOPPED, MessageType_ERROR_BUILDING_CONFIG, err.Error())
			StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
			return resp, err
		}
		parsedContent = *parsedContent_tmp
	}
	Log(LogLevel_DEBUG, LogType_CORE, "Saving config")
	currentBuildConfigPath := filepath.Join(sWorkingPath, "current-config.json")
	config.SaveCurrentConfig(currentBuildConfigPath, parsedContent)
	// if activeConfigPath == "" {
	// 	activeConfigPath = currentBuildConfigPath
	// }
	if in.EnableOldCommandServer {
		Log(LogLevel_DEBUG, LogType_CORE, "Starting Command Server")
		err = startCommandServer()
		if err != nil {
			Log(LogLevel_FATAL, LogType_CORE, err.Error())
			resp := SetCoreStatus(CoreStates_STOPPED, MessageType_START_COMMAND_SERVER, err.Error())
			StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
			return resp, err
		}
	}

	Log(LogLevel_DEBUG, LogType_CORE, "Stating Service ")
	instance, err := NewService(parsedContent)
	if err != nil {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		resp := SetCoreStatus(CoreStates_STOPPED, MessageType_CREATE_SERVICE, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
		return resp, err
	}
	Log(LogLevel_DEBUG, LogType_CORE, "Service.. started")
	if in.DelayStart {
		<-time.After(250 * time.Millisecond)
	}

	err = instance.Start()
	if err != nil {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		resp := SetCoreStatus(CoreStates_STOPPED, MessageType_START_SERVICE, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
		return resp, err
	}
	Box = instance
	if in.EnableOldCommandServer {
		oldCommandServer.SetService(Box)
	}

	resp := SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, "")
	return resp, nil
}

func (s *CoreService) Parse(ctx context.Context, in *ParseRequest) (*ParseResponse, error) {
	return Parse(in)
}

func Parse(in *ParseRequest) (*ParseResponse, error) {
	defer config.DeferPanicToError("parse", func(err error) {
		Log(LogLevel_FATAL, LogType_CONFIG, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
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

	config, err := config.ParseConfigContent(content, true, HiddifyOptions, false)
	if err != nil {
		return &ParseResponse{
			ResponseCode: common.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}
	if in.ConfigPath != "" {
		err = os.WriteFile(in.ConfigPath, config, 0o644)
		if err != nil {
			return &ParseResponse{
				ResponseCode: common.ResponseCode_FAILED,
				Message:      err.Error(),
			}, err
		}
	}
	return &ParseResponse{
		ResponseCode: common.ResponseCode_OK,
		Content:      string(config),
		Message:      "",
	}, err
}

func (s *CoreService) ChangeHiddifySettings(ctx context.Context, in *ChangeHiddifySettingsRequest) (*CoreInfoResponse, error) {
	return ChangeHiddifySettings(in)
}

func ChangeHiddifySettings(in *ChangeHiddifySettingsRequest) (*CoreInfoResponse, error) {
	HiddifyOptions = config.DefaultHiddifyOptions()
	if in.HiddifySettingsJson == "" {
		return &CoreInfoResponse{}, nil
	}
	settings := db.GetTable[common.AppSettings]()
	settings.UpdateInsert(&common.AppSettings{
		Id:    "HiddifySettingsJson",
		Value: in.HiddifySettingsJson,
	})
	err := json.Unmarshal([]byte(in.HiddifySettingsJson), HiddifyOptions)
	if err != nil {
		return nil, err
	}
	if HiddifyOptions.Warp.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(HiddifyOptions.Warp.WireguardConfigStr), &HiddifyOptions.Warp.WireguardConfig)
		if err != nil {
			return nil, err
		}
	}
	if HiddifyOptions.Warp2.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(HiddifyOptions.Warp2.WireguardConfigStr), &HiddifyOptions.Warp2.WireguardConfig)
		if err != nil {
			return nil, err
		}
	}
	return &CoreInfoResponse{}, nil
}

func (s *CoreService) GenerateConfig(ctx context.Context, in *GenerateConfigRequest) (*GenerateConfigResponse, error) {
	return GenerateConfig(in)
}

func GenerateConfig(in *GenerateConfigRequest) (*GenerateConfigResponse, error) {
	defer config.DeferPanicToError("generateConfig", func(err error) {
		Log(LogLevel_FATAL, LogType_CONFIG, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	})
	if HiddifyOptions == nil {
		HiddifyOptions = config.DefaultHiddifyOptions()
	}
	config, err := generateConfigFromFile(in.Path, *HiddifyOptions)
	if err != nil {
		return nil, err
	}
	return &GenerateConfigResponse{
		ConfigContent: config,
	}, nil
}

func generateConfigFromFile(path string, configOpt config.HiddifyOptions) (string, error) {
	os.Chdir(filepath.Dir(path))
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	options, err := readOptions(string(content))
	if err != nil {
		return "", err
	}
	config, err := config.BuildConfigJson(configOpt, options)
	if err != nil {
		return "", err
	}
	return config, nil
}

func (s *CoreService) Stop(ctx context.Context, empty *common.Empty) (*CoreInfoResponse, error) {
	return Stop()
}

func Stop() (*CoreInfoResponse, error) {
	defer config.DeferPanicToError("stop", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	})

	if CoreState != CoreStates_STARTED {
		Log(LogLevel_FATAL, LogType_CORE, "Core is not started")
		return &CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: MessageType_INSTANCE_NOT_STARTED,
			Message:     "instance is not started",
		}, fmt.Errorf("instance not started")
	}
	if Box == nil {
		return &CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: MessageType_INSTANCE_NOT_FOUND,
			Message:     "instance is not found",
		}, fmt.Errorf("instance not found")
	}
	SetCoreStatus(CoreStates_STOPPING, MessageType_EMPTY, "")
	config.DeactivateTunnelService()
	if oldCommandServer != nil {
		oldCommandServer.SetService(nil)
	}

	err := Box.Close()
	if err != nil {
		return &CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: MessageType_UNEXPECTED_ERROR,
			Message:     "Error while stopping the service.",
		}, fmt.Errorf("error while stopping the service")
	}
	Box = nil
	if oldCommandServer != nil {
		err = oldCommandServer.Close()
		if err != nil {
			return &CoreInfoResponse{
				CoreState:   CoreState,
				MessageType: MessageType_UNEXPECTED_ERROR,
				Message:     "Error while Closing the comand server.",
			}, fmt.Errorf("error while Closing the comand server")
		}
		oldCommandServer = nil
	}
	resp := SetCoreStatus(CoreStates_STOPPED, MessageType_EMPTY, "")
	return resp, nil
}

func (s *CoreService) Restart(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return Restart(in)
}

func Restart(in *StartRequest) (*CoreInfoResponse, error) {
	defer config.DeferPanicToError("restart", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	})
	log.Debug("[Service] Restarting")

	if CoreState != CoreStates_STARTED {
		return &CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: MessageType_INSTANCE_NOT_STARTED,
			Message:     "instance is not started",
		}, fmt.Errorf("instance not started")
	}
	if Box == nil {
		return &CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: MessageType_INSTANCE_NOT_FOUND,
			Message:     "instance is not found",
		}, fmt.Errorf("instance not found")
	}

	resp, err := Stop()
	if err != nil {
		return resp, err
	}

	SetCoreStatus(CoreStates_STARTING, MessageType_EMPTY, "")
	<-time.After(250 * time.Millisecond)

	libbox.SetMemoryLimit(!in.DisableMemoryLimit)
	resp, gErr := StartService(in)
	return resp, gErr
}

func Close() error {
	defer config.DeferPanicToError("close", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		StopAndAlert(MessageType_UNEXPECTED_ERROR, err.Error())
	})
	log.Debug("[Service] Closing")

	_, err := Stop()
	CloseGrpcServer()
	if err != nil {
		return err
	}
	return nil
}

// func (s *CoreService) Status(ctx context.Context, empty *common.Empty) (*CoreInfoResponse, error) {
// 	return Status()
// }
