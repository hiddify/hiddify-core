package hcore

import (
	"context"
	"fmt"
	"time"

	box "github.com/sagernet/sing-box"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/experimental/clashapi"
	"github.com/sagernet/sing-box/experimental/clashapi/trafficontrol"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/service"
)

// NewService creates the service and starts it. Once [stop] closes, the start in progress is
// interrupted, so a stop does not wait for a slow start (a download, a handshake) to finish; nil when
// nothing stops it.
func NewService(ctx context.Context, options option.Options, stop <-chan struct{}) (*daemon.StartedService, error) {

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
	watch := newStopwatch("start/service")
	err := libbox.CheckConfigOptions(&options)
	if err != nil {
		return nil, err
	}
	watch.step("checking the config")
	instance := daemon.NewStartedService(bopts)
	if stop != nil {
		started := make(chan struct{})
		defer close(started)
		go func() {
			select {
			case <-stop:
			case <-started:
				return
			}
			// the instance only becomes interruptible once it is created, so keep asking until the
			// start has given up
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for {
				instance.InterruptStart()
				select {
				case <-started:
					return
				case <-ticker.C:
				}
			}
		}()
	}

	// for i := 0; i < 10; i++ {
	// 	if hutils.IsPortInUse(options.Inbounds[0].SocksOptions.ListenPort) {
	// 		<-time.After(100 * time.Millisecond)
	// 	}
	// }

	content, err := json.MarshalContext(ctx, options)
	if err != nil {
		closeService(instance)
		return nil, err
	}
	watch.step("encoding the config")
	if err := instance.StartOrReloadService(ctx, string(content), nil); err != nil {
		closeService(instance)
		return nil, err
	}
	watch.step("starting the inbounds, outbounds and services")

	// instance.GetInstance().AddPostService("hiddifyMainServiceManager", &hiddifyMainServiceManager{})

	// if err := startCommandServer(instance); err != nil {
	// 	return errorWrapper(MessageType_START_COMMAND_SERVER, err)
	// }

	return instance, nil
}

// closeService stops ss and ends its observer goroutines. CloseService alone
// stops only the box and leaves five goroutines running for every service built.
func closeService(ss *daemon.StartedService) error {
	watch := newStopwatch("stop/service")
	// a close that takes long is the thing to look at: show what every goroutine is waiting for
	slow := time.AfterFunc(3*time.Second, func() {
		path := fmt.Sprint(sWorkingPath, "/data/goroutine-stop-slow.log")
		reportTiming("stop/service: closing takes more than 3s, goroutines dumped to " + path)
		dumpGoroutinesToFile(path)
	})
	defer slow.Stop()
	err := ss.CloseService()
	watch.step("closing the box (inbounds, outbounds, services)")
	ss.Close()
	watch.step("ending the observers")
	return err
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
	return ss.Instance()

}

func (h *HiddifyInstance) Context() context.Context {
	ins := h.Instance()
	if ins == nil {
		return nil
	}
	return ins.Context()
}

func (h *HiddifyInstance) TrafficManager() *trafficontrol.Manager {
	if ctx := h.Context(); ctx != nil {
		if s, ok := service.FromContext[adapter.ClashServer](ctx).(*clashapi.Server); ok {
			return s.TrafficManager()
		}
	}
	return nil
}
