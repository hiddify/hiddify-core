package hcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	service_manager "github.com/hiddify/hiddify-core/v2/service_manager"
	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"
)

func (s *CoreService) Start(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return Start(withRequestContext(static.BaseContext, ctx), in)
}

func Start(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return StartService(ctx, in)
}

func (s *CoreService) StartService(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return StartService(withRequestContext(static.BaseContext, ctx), in)
}

type requestContextKey struct{}

// withRequestContext attaches the caller's request context to base. base stays the context the
// core runs with, so the core keeps running after the request returns; the request context is
// only checked by contextErr, to give up on a start, restart or reload nobody waits for anymore.
func withRequestContext(base, request context.Context) context.Context {
	return context.WithValue(base, requestContextKey{}, request)
}

// contextErr reports why ctx, or the request context attached to it, is done; nil while both
// are alive.
func contextErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if request, ok := ctx.Value(requestContextKey{}).(context.Context); ok {
		if err := request.Err(); err != nil {
			return err
		}
	}
	return nil
}

// sleepContext waits for d, returning early once ctx or its request context is done.
func sleepContext(ctx context.Context, d time.Duration) {
	var requestDone <-chan struct{}
	if request, ok := ctx.Value(requestContextKey{}).(context.Context); ok {
		requestDone = request.Done()
	}
	select {
	case <-ctx.Done():
	case <-requestDone:
	case <-time.After(d):
	}
}

// cancelledStart reports a start given up because its context is done; the core stays STOPPED.
func cancelledStart(err error) (*CoreInfoResponse, error) {
	Log(LogLevel_WARNING, LogType_CORE, "start cancelled: ", err.Error())
	return SetCoreStatus(CoreStates_STOPPED, MessageType_INSTANCE_NOT_STARTED, "start cancelled"), err
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
		&hcommon.AppSettings{
			Id:    "lastStartRequestName",
			Value: in.ConfigName,
		},
	)
}

func loadLastStartRequestIfNeeded(in *StartRequest) (*StartRequest, error) {
	if in != nil && (in.ConfigContent != "" || in.ConfigPath != "") {
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

	lastName, err := settings.Get("lastStartRequestName")
	if err != nil {
		return nil, err
	}
	return &StartRequest{
		ConfigPath:    lastPath.Value.(string),
		ConfigContent: lastContent.Value.(string),
		ConfigName:    lastName.Value.(string),
	}, nil
}

func StartService(ctx context.Context, in *StartRequest) (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("startmobile", func(recovered_err error) {
		coreResponse, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered_err)
	})
	static.lock.Lock()
	defer static.lock.Unlock()
	if err := contextErr(ctx); err != nil {
		return cancelledStart(err)
	}
	return startServiceLocked(ctx, in)
}

// startServiceLocked is StartService with static.lock held.
func startServiceLocked(ctx context.Context, in *StartRequest) (coreResponse *CoreInfoResponse, err error) {
	if static.CoreState != CoreStates_STOPPED {
		// return errorWrapper(MessageType_ALREADY_STARTED, fmt.Errorf("instance already started"))
		return &CoreInfoResponse{
			CoreState:   static.CoreState,
			MessageType: MessageType_ALREADY_STARTED,
			Message:     "instance already started",
		}, nil
	}
	SetCoreStatus(CoreStates_STARTING, MessageType_EMPTY, "")

	in, err = loadLastStartRequestIfNeeded(in)
	if err != nil {
		return errorWrapper(MessageType_ERROR_BUILDING_CONFIG, err)
	}

	static.previousStartRequest = in

	if static.HiddifyOptions == nil {
		return errorWrapper(
			MessageType_ERROR_BUILDING_CONFIG,
			errors.New("HiddifyOptions not initialized"),
		)
	}

	options, err := BuildConfig(ctx, in)
	if err != nil {
		return errorWrapper(MessageType_ERROR_BUILDING_CONFIG, err)
	}
	if err := contextErr(ctx); err != nil {
		return cancelledStart(err)
	}
	saveLastStartRequest(in)

	Log(LogLevel_DEBUG, LogType_CORE, "Main Service pre start")
	if err := service_manager.OnMainServicePreStart(options); err != nil {
		return errorWrapper(MessageType_ERROR_EXTENSION, err)
	}
	if err := contextErr(ctx); err != nil {
		return cancelledStart(err)
	}
	currentBuildConfigPath := filepath.Join(sWorkingPath, "data/current-config.json")
	Log(LogLevel_DEBUG, LogType_CORE, "Saving config to ", currentBuildConfigPath)

	config.SaveCurrentConfig(ctx, currentBuildConfigPath, *options)
	if static.debug {
		pout, err := options.MarshalJSONContext(ctx)
		if err != nil {
			return errorWrapper(MessageType_ERROR_BUILDING_CONFIG, err)
		}
		Log(LogLevel_INFO, LogType_CORE, "Current Config is:\n", string(pout))
	}
	ctx = libbox.FromContext(ctx, static.globalPlatformInterface)
	if static.globalPlatformInterface != nil {
		platformWrapper := libbox.WrapPlatformInterface(static.globalPlatformInterface)
		service.MustRegister[adapter.PlatformInterface](ctx, platformWrapper)
		// } else {
		// 	service.MustRegister[adapter.PlatformInterface](ctx, (*adapter.PlatformInterface)nil)
	}
	Log(LogLevel_DEBUG, LogType_CORE, "Stating Service with delay ?", in.DelayStart)
	if in.DelayStart {
		sleepContext(ctx, 1000*time.Millisecond)
	}
	if err := contextErr(ctx); err != nil {
		return cancelledStart(err)
	}
	libbox.SetMemoryLimit(C.IsIos || !in.DisableMemoryLimit)
	instance, err := NewService(ctx, *options)
	if err != nil {
		return errorWrapper(MessageType_START_SERVICE, err)
	}
	if err := contextErr(ctx); err != nil {
		// cancelled while the service was starting: stop it again
		if closeErr := instance.CloseService(); closeErr != nil {
			Log(LogLevel_ERROR, LogType_CORE, "closing cancelled service: ", closeErr.Error())
		}
		return cancelledStart(err)
	}
	static.StartedService = instance
	if static.debug {
		dumpGoroutinesToFile(fmt.Sprint(sWorkingPath, "/data/goroutine-start.log"))
	}
	for inb := range options.Inbounds {
		if opts, ok := options.Inbounds[inb].Options.(option.SocksInboundOptions); ok {
			static.ListenPort = opts.ListenPort
		}
	}

	return SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, ""), nil
}
