package hcore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
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
	Log(LogLevel_DEBUG, LogType_CORE, "Building Config...")
	content := in.ConfigContent

	if content == "" {
		fileContent, err := os.ReadFile(in.ConfigPath)
		if err != nil {
			return nil, err
		}
		content = string(fileContent)
	}
	// Log(LogLevel_DEBUG, LogType_CORE, "Parsing Config... ", in.ConfigPath, " content:", content, "-")
	Log(LogLevel_DEBUG, LogType_CORE, "Parsing Config... ")

	parsedContent, err := readOptions(content)
	Log(LogLevel_DEBUG, LogType_CORE, "Parsed")

	if err != nil {
		return nil, err
	}

	if !in.EnableRawConfig {
		// hcontent, err := json.MarshalIndent(static.HiddifyOptions, "", " ")
		// if err != nil {
		// 	return nil, err
		// }

		// Log(LogLevel_DEBUG, LogType_CORE, "Building config ", string(hcontent))
		// Log(LogLevel_DEBUG, LogType_CORE, "Building config ")
		return config.BuildConfig(*static.HiddifyOptions, parsedContent)
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
	if content == "" {
		path := in.TempPath
		if path == "" {
			path = in.ConfigPath
		}
		contentBytes, err := os.ReadFile(path)
		content = string(contentBytes)
		// os.Chdir(filepath.Dir(in.ConfigPath))
		if err != nil {
			return nil, err
		}

	}

	config, err := config.ParseConfigContent(content, true, static.HiddifyOptions, false)
	if err != nil {
		return &ParseResponse{
			ResponseCode: hcommon.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}
	if in.ConfigPath != "" {
		err = os.WriteFile(in.ConfigPath, config, 0o644)
		if err != nil {
			return &ParseResponse{
				ResponseCode: hcommon.ResponseCode_FAILED,
				Message:      err.Error(),
			}, err
		}
	}
	return &ParseResponse{
		ResponseCode: hcommon.ResponseCode_OK,
		Content:      string(config),
		Message:      "",
	}, err
}

func (s *CoreService) ChangeHiddifySettings(ctx context.Context, in *ChangeHiddifySettingsRequest) (*CoreInfoResponse, error) {
	return ChangeHiddifySettings(in, true)
}

func ChangeHiddifySettings(in *ChangeHiddifySettingsRequest, insert bool) (*CoreInfoResponse, error) {
	static.HiddifyOptions = config.DefaultHiddifyOptions()
	if in.HiddifySettingsJson == "" {
		return &CoreInfoResponse{}, nil
	}
	if insert {
		settings := db.GetTable[hcommon.AppSettings]()
		settings.UpdateInsert(&hcommon.AppSettings{
			Id:    "HiddifySettingsJson",
			Value: in.HiddifySettingsJson,
		})
	}
	err := json.Unmarshal([]byte(in.HiddifySettingsJson), static.HiddifyOptions)
	if err != nil {
		return nil, err
	}
	if static.HiddifyOptions.Warp.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(static.HiddifyOptions.Warp.WireguardConfigStr), &static.HiddifyOptions.Warp.WireguardConfig)
		if err != nil {
			return nil, err
		}
	}
	if static.HiddifyOptions.Warp2.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(static.HiddifyOptions.Warp2.WireguardConfigStr), &static.HiddifyOptions.Warp2.WireguardConfig)
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
	if static.HiddifyOptions == nil {
		static.HiddifyOptions = config.DefaultHiddifyOptions()
	}
	config, err := generateConfigFromFile(in.Path, *static.HiddifyOptions)
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
