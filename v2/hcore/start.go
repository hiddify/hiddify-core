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

// Start queues a start of the core and answers at once with STARTING (or ALREADY_STARTED when
// the core is started or will be); the outcome is reported through CoreInfoListener.
func (s *CoreService) Start(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return startInBackground(in), nil
}

func (s *CoreService) StartService(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return startInBackground(in), nil
}

func startInBackground(in *StartRequest) *CoreInfoResponse {
	queue.Lock()
	defer queue.Unlock()
	if startingLocked() {
		return &CoreInfoResponse{
			CoreState:   currentCoreState(),
			MessageType: MessageType_ALREADY_STARTED,
			Message:     "instance already started",
		}
	}
	queueStartLocked(static.BaseContext, in)
	return &CoreInfoResponse{CoreState: CoreStates_STARTING, MessageType: MessageType_EMPTY}
}

func Start(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return StartService(ctx, in)
}

// StartService starts the core after the operations queued before it, and returns the outcome.
func StartService(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	queue.Lock()
	result := queueStartLocked(ctx, in)
	queue.Unlock()
	return wait(result)
}

// queueStartLocked queues a start that a stop called after it cancels; queue must be locked.
func queueStartLocked(ctx context.Context, in *StartRequest) <-chan operationResult {
	ctx = withStopSignalLocked(ctx)
	return enqueueLocked(operationStart, "start", func() (*CoreInfoResponse, error) {
		if err := contextErr(ctx); err != nil {
			return cancelledStart(err)
		}
		return startServiceLocked(ctx, in)
	})
}

// cancelledStart reports a start given up because its context is done; the core stays STOPPED.
// For a stop, nothing is published: the stop that follows reports STOPPING and STOPPED.
func cancelledStart(err error) (*CoreInfoResponse, error) {
	Log(LogLevel_WARNING, LogType_CORE, "start cancelled: ", err.Error())
	if errors.Is(err, errStopped) {
		setCoreState(CoreStates_STOPPED)
		return &CoreInfoResponse{CoreState: CoreStates_STOPPED, MessageType: MessageType_INSTANCE_NOT_STARTED, Message: "start cancelled"}, err
	}
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

// startServiceLocked starts the core; static.lock must be held.
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
	instance, err := NewService(serviceContext(ctx), *options)
	if err != nil {
		return errorWrapper(MessageType_START_SERVICE, err)
	}
	if err := contextErr(ctx); err != nil {
		// cancelled while the service was starting: stop it again
		if closeErr := closeService(instance); closeErr != nil {
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
