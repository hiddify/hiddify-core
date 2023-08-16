package main

import (
	"context"
	"os"

	B "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
)

var (
	sBasePath    string
	sWorkingPath string
	sTempPath    string
	sUserID      int
	sGroupID     int
)

type BoxService struct {
	ctx      context.Context
	cancel   context.CancelFunc
	instance *B.Box
}

func Setup(basePath string, workingPath string, tempPath string) {
	sBasePath = basePath
	sWorkingPath = workingPath
	sTempPath = tempPath
	sUserID = os.Getuid()
	sGroupID = os.Getgid()
}

func NewService(configContent string) (*BoxService, error) {
	options, err := parseConfig(configContent)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx = filemanager.WithDefault(ctx, sWorkingPath, sTempPath, sUserID, sGroupID)
	ctx = service.ContextWithPtr(ctx, urltest.NewHistoryStorage())
	instance, err := B.New(B.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		cancel()
		return nil, E.Cause(err, "create service")
	}
	return &BoxService{
		ctx:      ctx,
		cancel:   cancel,
		instance: instance,
	}, nil
}

func (s *BoxService) Start() error {
	return s.instance.Start()
}

func (s *BoxService) Close() error {
	s.cancel()
	return s.instance.Close()
}

func parseConfig(configContent string) (option.Options, error) {
	var options option.Options
	err := options.UnmarshalJSON([]byte(configContent))
	if err != nil {
		return option.Options{}, E.Cause(err, "decode config")
	}
	return options, nil
}
