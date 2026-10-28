package hcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/hiddify/hiddify-core/v2/config"
	service_manager "github.com/hiddify/hiddify-core/v2/service_manager"
	"github.com/sagernet/sing-box/experimental/libbox"
)

// HotReload waits for the outcome, unlike Start and Stop: on HOT_RELOAD_FAILED the caller has to
// restart the core to apply the config.
func (s *CoreService) HotReload(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return HotReload(static.BaseContext, in)
}

// HotReload applies the request's config to the running core without restarting it, or starts
// the core when it is stopped. When the config cannot be hot reloaded (a TUN inbound or a
// setting outside outbounds/inbounds/DNS/route changed), HOT_RELOAD_FAILED is returned and the
// core keeps running unchanged: restart it to apply the config.
func HotReload(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	queue.Lock()
	ctx = withStopSignalLocked(ctx)
	// HOT_RELOADING shows at once (like STOPPING), not only once the queue reaches the reload
	setCoreStatusIf(append([]CoreStates{CoreStates_HOT_RELOADING}, running...), CoreStates_HOT_RELOADING, MessageType_EMPTY, "")
	result := enqueueLocked(operationStart, "hotreload", func() (coreResponse *CoreInfoResponse, err error) {
		defer config.DeferPanicToError("hotreload", func(recovered_err error) {
			coreResponse, err = hotReloadError(MessageType_UNEXPECTED_ERROR, recovered_err)
		})
		return hotReloadLocked(ctx, in)
	})
	queue.Unlock()
	return wait(result)
}

// hotReloadLocked is HotReload; static.lock must be held.
func hotReloadLocked(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	if err := contextErr(ctx); err != nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, fmt.Errorf("hot reload cancelled: %w", err))
	}
	switch currentCoreState() {
	case CoreStates_STOPPED:
		return startServiceLocked(ctx, in)
	case CoreStates_WAITING, CoreStates_STARTED, CoreStates_NO_AVAILABLE_PROXIES, CoreStates_HOT_RELOADING:
		// HOT_RELOADING may already be set when the call was accepted (shown at once)
		SetCoreStatus(CoreStates_HOT_RELOADING, MessageType_EMPTY, "")
		response, err := hotReloadService(ctx, in)
		if err != nil {
			// config unchanged: restore whatever the current pings say
			setCoreStatusIf([]CoreStates{CoreStates_HOT_RELOADING}, static.connectedState(), MessageType_EMPTY, "")
			return response, err
		}
		// same as start: WAITING until the lowest-group pings decide STARTED / NO_AVAILABLE_PROXIES
		if response, ok := setCoreStatusIf([]CoreStates{CoreStates_HOT_RELOADING}, CoreStates_WAITING, MessageType_EMPTY, ""); ok {
			startConnectivityWatcher()
			return response, nil
		}
		return &CoreInfoResponse{CoreState: currentCoreState(), MessageType: MessageType_EMPTY}, nil
	default:
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, fmt.Errorf("core is %s", currentCoreState()))
	}
}

// hotReloadService applies the request's config to the running core without restarting it (see
// box.HotReload for what can change). On failure the core keeps running unchanged and
// HOT_RELOAD_FAILED (or ERROR_BUILDING_CONFIG) is returned.
func hotReloadService(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	in, err := loadLastStartRequestIfNeeded(in)
	if err != nil {
		return hotReloadError(MessageType_ERROR_BUILDING_CONFIG, err)
	}
	if static.HiddifyOptions == nil {
		return hotReloadError(MessageType_ERROR_BUILDING_CONFIG, errors.New("HiddifyOptions not initialized"))
	}
	if static.StartedService == nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, errors.New("no running service to hot reload"))
	}
	SetCoreStatus(CoreStates_HOT_RELOADING, MessageType_EMPTY, "")
	options, err := BuildConfig(ctx, in)
	if err != nil {
		return hotReloadError(MessageType_ERROR_BUILDING_CONFIG, err)
	}
	// extensions may change the config at start; apply them the same way
	if err := service_manager.OnMainServicePreStart(options); err != nil {
		return hotReloadError(MessageType_ERROR_EXTENSION, err)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		return hotReloadError(MessageType_ERROR_BUILDING_CONFIG, err)
	}
	if err := contextErr(ctx); err != nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, fmt.Errorf("hot reload cancelled: %w", err))
	}
	tunChanged, err := static.StartedService.IsTunChanged(ctx, options)
	if err != nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, err)
	}
	if tunChanged {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, errors.New("tun inbound has changed, hot reload can not be applied"))
	}
	// last point to give up: once applied, the reload is kept and recorded below
	if err := contextErr(ctx); err != nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, fmt.Errorf("hot reload cancelled: %w", err))
	}
	result, err := static.StartedService.HotReload(serviceContext(ctx), options)
	if err != nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, err)
	}
	static.previousStartRequest = in
	saveLastStartRequest(in)
	config.SaveCurrentConfig(ctx, filepath.Join(sWorkingPath, "data/current-config.json"), *options)
	Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("hot reload: %+v", result))
	return &CoreInfoResponse{CoreState: CoreStates_HOT_RELOADING, MessageType: MessageType_EMPTY}, nil
}

// hotReloadError reports a failed hot reload; the core keeps running unchanged.
func hotReloadError(messageType MessageType, err error) (*CoreInfoResponse, error) {
	Log(LogLevel_ERROR, LogType_CORE, "hot reload: ", err.Error())
	state := currentCoreState()
	if state == CoreStates_HOT_RELOADING {
		state = static.connectedState()
	}
	return &CoreInfoResponse{
		CoreState:   state,
		MessageType: messageType,
		Message:     err.Error(),
	}, err
}
