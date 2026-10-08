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

func (s *CoreService) HotReload(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return HotReload(static.BaseContext, in)
}

// HotReload applies the request's config to the running core without restarting it, or starts
// the core when it is stopped. When the config cannot be hot reloaded (a TUN inbound or a
// setting outside outbounds/inbounds/DNS/route changed), HOT_RELOAD_FAILED is returned and the
// core keeps running unchanged: restart it to apply the config.
func HotReload(ctx context.Context, in *StartRequest) (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("hotreload", func(recovered_err error) {
		coreResponse, err = hotReloadError(MessageType_UNEXPECTED_ERROR, recovered_err)
	})
	static.lock.Lock()
	defer static.lock.Unlock()
	switch static.CoreState {
	case CoreStates_STOPPED:
		return startServiceLocked(ctx, in)
	case CoreStates_STARTED:
		SetCoreStatus(CoreStates_HOT_RELOADING, MessageType_EMPTY, "")
		// back to STARTED whatever happens: the core keeps running unchanged on a failure
		defer func() {
			if static.CoreState == CoreStates_HOT_RELOADING {
				SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, "")
			}
		}()
		return hotReloadService(ctx, in)
	default:
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, fmt.Errorf("core is %s", static.CoreState))
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
	tunChanged, err := static.StartedService.IsTunChanged(ctx, options)
	if err != nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, err)
	}
	if tunChanged {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, errors.New("tun inbound has changed, hot reload can not be applied"))
	}
	result, err := static.StartedService.HotReload(ctx, options)
	if err != nil {
		return hotReloadError(MessageType_HOT_RELOAD_FAILED, err)
	}
	static.previousStartRequest = in
	saveLastStartRequest(in)
	config.SaveCurrentConfig(ctx, filepath.Join(sWorkingPath, "data/current-config.json"), *options)
	Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("hot reload: %+v", result))
	return SetCoreStatus(CoreStates_STARTED, MessageType_EMPTY, ""), nil
}

// hotReloadError reports a failed hot reload; the core keeps running unchanged (STARTED).
func hotReloadError(messageType MessageType, err error) (*CoreInfoResponse, error) {
	Log(LogLevel_ERROR, LogType_CORE, "hot reload: ", err.Error())
	state := static.CoreState
	if state == CoreStates_HOT_RELOADING {
		state = CoreStates_STARTED
	}
	return &CoreInfoResponse{
		CoreState:   state,
		MessageType: messageType,
		Message:     err.Error(),
	}, err
}
