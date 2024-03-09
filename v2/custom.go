package v2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hiddify/libcore/config"
	pb "github.com/hiddify/libcore/hiddifyrpc"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

var Box *libbox.BoxService
var configOptions *config.ConfigOptions
var activeConfigPath *string
var logFactory *log.Factory

func StopAndAlert(msgType pb.MessageType, message string) {
	SetCoreStatus(pb.CoreState_STOPPED, msgType, message)
	config.DeactivateTunnelService()
	// if commandServer != nil {
	// 	commandServer.SetService(nil)
	// }
	if Box != nil {
		Box.Close()
		Box = nil
	}
	// if commandServer != nil {
	// 	commandServer.Close()
	// }
}

func (s *server) Start(ctx context.Context, in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
	defer config.DeferPanicToError("start", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})

	if CoreState != pb.CoreState_STOPPED {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_INSTANCE_NOT_STOPPED,
		}, fmt.Errorf("instance not stopped")
	}
	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")

	libbox.SetMemoryLimit(!in.DisableMemoryLimit)
	resp, err := s.StartService(ctx, in)
	return resp, err
}

// Implement the StartService method
func (s *server) StartService(ctx context.Context, in *pb.StartRequest) (*pb.CoreInfoResponse, error) {

	content := in.ConfigContent
	if content != "" {
		fileContent, err := os.ReadFile(*activeConfigPath)
		if err != nil {
			resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_ERROR_READING_CONFIG, err.Error())
			return &resp, err
		}
		content = string(fileContent)
	}

	parsedContent, err := parseConfig(content)
	if err != nil {
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_ERROR_PARSING_CONFIG, err.Error())
		return &resp, err
	}
	var patchedOptions *option.Options
	patchedOptions, err = config.BuildConfig(*configOptions, parsedContent)
	if err != nil {
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_ERROR_BUILDING_CONFIG, err.Error())
		return &resp, err
	}

	config.SaveCurrentConfig(filepath.Join(sWorkingPath, "current-config.json"), *patchedOptions)

	// err = startCommandServer(*logFactory)
	// if err != nil {
	// 	resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_COMMAND_SERVER, err.Error())
	// 	return &resp, err
	// }

	instance, err := NewService(*patchedOptions)
	if err != nil {
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_CREATE_SERVICE, err.Error())
		return &resp, err
	}

	if in.DelayStart {
		<-time.After(250 * time.Millisecond)
	}

	err = instance.Start()
	if err != nil {
		resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_START_SERVICE, err.Error())
		return &resp, err
	}
	Box = instance
	// commandServer.SetService(box)

	resp := SetCoreStatus(pb.CoreState_STARTED, pb.MessageType_EMPTY, "")
	return &resp, nil

}

func (s *server) Parse(ctx context.Context, in *pb.ParseRequest) (*pb.ParseResponse, error) {
	defer config.DeferPanicToError("parse", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CONFIG, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})

	config, err := config.ParseConfigContent(in.Content, true)
	if err != nil {
		return &pb.ParseResponse{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}
	return &pb.ParseResponse{
		ResponseCode: pb.ResponseCode_OK,
		Content:      string(config),
		Message:      "",
	}, err
}

// func (s *server) ChangeConfigOptions(ctx context.Context, in *pb.ChangeConfigOptionsRequest) (*pb.CoreInfoResponse, error) {
// 	// Implement your change config options logic
// 	// Return a CoreInfoResponse
// }

// func (s *server) GenerateConfig(ctx context.Context, in *pb.GenerateConfigRequest) (*pb.GenerateConfigResponse, error) {
// 	defer config.DeferPanicToError("generateConfig", func(err error) {
// 		Log(pb.LogLevel_FATAL, pb.LogType_CONFIG, err.Error())
// 		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
// 	})

// 	config, err := generateConfigFromFile(C.GoString(path), *configOptions)
// 	if err != nil {
// 		return C.CString("error" + err.Error())
// 	}
// 	return C.CString(config)
// }

// Implement the Stop method
func (s *server) Stop(ctx context.Context, empty *pb.Empty) (*pb.CoreInfoResponse, error) {
	defer config.DeferPanicToError("stop", func(err error) {
		Log(pb.LogLevel_FATAL, pb.LogType_CORE, err.Error())
		StopAndAlert(pb.MessageType_UNEXPECTED_ERROR, err.Error())
	})
	config.DeactivateTunnelService()
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
	// commandServer.SetService(nil)

	err := Box.Close()
	if err != nil {
		return &pb.CoreInfoResponse{
			CoreState:   CoreState,
			MessageType: pb.MessageType_UNEXPECTED_ERROR,
			Message:     "Error while stopping the service.",
		}, fmt.Errorf("Error while stopping the service.")
	}
	Box = nil
	// err = commandServer.Close()
	// if err != nil {
	// 	return &pb.CoreInfoResponse{
	// 		CoreState:   CoreState,
	// 		MessageType: pb.MessageType_UNEXPECTED_ERROR,
	// 		Message:     "Error while Closing the comand server.",
	// 	}, fmt.Errorf("Error while Closing the comand server.")

	// }
	// commandServer = nil
	resp := SetCoreStatus(pb.CoreState_STOPPED, pb.MessageType_EMPTY, "")
	return &resp, nil

}

func (s *server) Restart(ctx context.Context, in *pb.StartRequest) (*pb.CoreInfoResponse, error) {
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

	resp, err := s.Stop(ctx, &pb.Empty{})
	if err != nil {
		return resp, err
	}

	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")
	<-time.After(250 * time.Millisecond)

	libbox.SetMemoryLimit(!in.DisableMemoryLimit)
	resp, gErr := s.StartService(ctx, in)
	return resp, gErr
}
