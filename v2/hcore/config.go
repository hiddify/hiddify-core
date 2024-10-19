package hcore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	common "github.com/hiddify/hiddify-core/v2/common"
	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
	hutils "github.com/hiddify/hiddify-core/v2/hutils"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

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

func removeTunnelIfNeeded(options *option.Options) (tuninb *option.TunInboundOptions) {
	if hutils.TunAllowed() {
		return nil
	}

	// Create a new slice to hold the remaining inbounds
	newInbounds := make([]option.Inbound, 0, len(options.Inbounds))

	for _, inb := range options.Inbounds {
		if inb.Type == C.TypeTun {
			tuninb = &inb.TunOptions
		} else {
			newInbounds = append(newInbounds, inb)
		}
	}

	options.Inbounds = newInbounds
	return tuninb
}
