package hcore

import (
	"context"

	box "github.com/sagernet/sing-box"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

func NewService(ctx context.Context, options option.Options) (*daemon.StartedService, error) {

	// ctx = filemanager.WithDefault(ctx, sWorkingPath, sTempPath, sUserID, sGroupID)
	logInterface := LogInterface{}
	bopts := daemon.ServiceOptions{
		Context:     ctx,
		Debug:       static.debug,
		LogMaxLines: 100,
		// Options:           *options,
		Handler: &logInterface,
		ExtraServices: []adapter.LifecycleService{
			&hiddifyMainServiceManager{},
		},
	}
	err := libbox.CheckConfigOptions(&options)
	if err != nil {
		return nil, err
	}
	instance := daemon.NewStartedService(bopts)

	// for i := 0; i < 10; i++ {
	// 	if hutils.IsPortInUse(options.Inbounds[0].SocksOptions.ListenPort) {
	// 		<-time.After(100 * time.Millisecond)
	// 	}
	// }

	if err := instance.StartOrReloadServiceOptions(options); err != nil {
		return nil, err
	}

	// instance.GetInstance().AddPostService("hiddifyMainServiceManager", &hiddifyMainServiceManager{})

	// if err := startCommandServer(instance); err != nil {
	// 	return errorWrapper(MessageType_START_COMMAND_SERVER, err)
	// }

	return instance, nil
}

func (h *HiddifyInstance) UrlTestHistory() *urltest.HistoryStorage {

	ins := h.Instance()
	if ins == nil {
		return nil
	}
	return ins.UrlTestHistory()
}

func (h *HiddifyInstance) Box() *box.Box {
	ins := h.Instance()
	if ins == nil {
		return nil
	}
	return ins.Box()
}

func (h *HiddifyInstance) Instance() *daemon.Instance {
	ss := h.StartedService
	if ss == nil {
		return nil
	}
	return h.StartedService.Instance()

}

func (h *HiddifyInstance) Context() context.Context {
	ins := h.Instance()
	if ins == nil {
		return nil
	}
	return ins.Context()
}
