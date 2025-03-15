package hcore

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	service_manager "github.com/hiddify/hiddify-core/v2/service_manager"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func (s *CoreService) Start(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return Start(in)
}

func Start(in *StartRequest) (*CoreInfoResponse, error) {
	return StartService(in)
}

func (s *CoreService) StartService(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return StartService(in)
}

func saveLastStartRequest(in *StartRequest) error {
	if in.ConfigContent == "" && in.ConfigPath == "" {
		return nil
	}
	settings := db.GetTable[hcommon.AppSettings]()
	return settings.UpdateInsert(
		&hcommon.AppSettings{
			Id:    "lastStartRequestPath",
			Value: in.ConfigPath,
		},
		&hcommon.AppSettings{
			Id:    "lastStartRequestContent",
			Value: in.ConfigContent,
		},
	)
}

func loadLastStartRequestIfNeeded(in *StartRequest) (*StartRequest, error) {
	if in.ConfigContent != "" || in.ConfigPath != "" {
		return in, nil
	}
	settings := db.GetTable[hcommon.AppSettings]()
	lastPath, err := settings.Get("lastStartRequestPath")
	if err != nil {
		return nil, err
	}
	lastContent, err := settings.Get("lastStartRequestContent")
	if err != nil {
		return nil, err
	}
	return &StartRequest{
		ConfigPath:    lastPath.Value.(string),
		ConfigContent: lastContent.Value.(string),
	}, nil
}

func StartService(in *StartRequest) (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("startmobile", func(recovered_err error) {
		coreResponse, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered_err)
	})
	SetCoreStatus(CoreStates_STARTING, MessageType_EMPTY, "")

	in, err = loadLastStartRequestIfNeeded(in)
	if err != nil {
		return errorWrapper(MessageType_ERROR_BUILDING_CONFIG, err)
	}

	previousStartRequest = in
	options, err := BuildConfig(in)
	if err != nil {
		return errorWrapper(MessageType_ERROR_BUILDING_CONFIG, err)
	}
	saveLastStartRequest(in)

	Log(LogLevel_DEBUG, LogType_CORE, "Main Service pre start")
	if err := service_manager.OnMainServicePreStart(options); err != nil {
		return errorWrapper(MessageType_ERROR_EXTENSION, err)
	}
	currentBuildConfigPath := filepath.Join(sWorkingPath, "data/current-config.json")
	Log(LogLevel_DEBUG, LogType_CORE, "Saving config to ", currentBuildConfigPath)

	config.SaveCurrentConfig(currentBuildConfigPath, *options)
	pout, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return errorWrapper(MessageType_ERROR_BUILDING_CONFIG, err)
	}
	Log(LogLevel_DEBUG, LogType_CORE, string(pout))

	bopts := box.Options{
		Options:           *options,
		PlatformLogWriter: &LogInterface{},
	}
	if globalPlatformInterface != nil {
		bopts.PlatformInterface = libbox.WrapPlatformInterface(globalPlatformInterface)
	}
	instance, err := libbox.NewHService(bopts)
	if err != nil {
		return errorWrapper(MessageType_CREATE_SERVICE, err)
	}
	// for i := 0; i < 10; i++ {
	// 	if hutils.IsPortInUse(options.Inbounds[0].SocksOptions.ListenPort) {
	// 		<-time.After(100 * time.Millisecond)
	// 	}
	// }
	Log(LogLevel_DEBUG, LogType_CORE, "Stating Service with delay ?", in.DelayStart)
	if in.DelayStart {
		<-time.After(250 * time.Millisecond)
	}

	instance.GetInstance().AddPostService("hiddifyMainServiceManager", &hiddifyMainServiceManager{})

	// if err := startCommandServer(instance); err != nil {
	// 	return errorWrapper(MessageType_START_COMMAND_SERVER, err)
	// }

	if err := instance.Start(); err != nil {
		return errorWrapper(MessageType_START_SERVICE, err)
	}
	static.Box = instance

	return SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, ""), nil
}
