package hcore

import (
	"context"
	"fmt"
	"io"
	"os"
	runtimeDebug "runtime/debug"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
	"github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hutils"
	"github.com/hiddify/hiddify-core/v2/service_manager"

	B "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
	"github.com/sagernet/sing/service/pause"
)

var (
	sWorkingPath          string
	sTempPath             string
	sUserID               int
	sGroupID              int
	statusPropagationPort int64
)

func InitHiddifyService() error {
	return service_manager.StartServices()
}

func (s *CoreService) Setup(ctx context.Context, req *SetupRequest) (*hcommon.Response, error) {
	if grpcServer[req.Mode] != nil {
		return &hcommon.Response{Code: hcommon.ResponseCode_OK, Message: ""}, nil
	}
	err := Setup(req, nil)
	code := hcommon.ResponseCode_OK
	if err != nil {
		code = hcommon.ResponseCode_FAILED
	}
	return &hcommon.Response{Code: code, Message: err.Error()}, err
}

func Setup(params *SetupRequest, platformInterface libbox.PlatformInterface) error {
	defer config.DeferPanicToError("setup", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
		<-time.After(5 * time.Second)
	})
	static.globalPlatformInterface = platformInterface
	if grpcServer[params.Mode] != nil {
		Log(LogLevel_WARNING, LogType_CORE, "grpcServer already started")
		return nil
	}
	tcpConn := true // runtime.GOOS == "windows" // TODO add TVOS
	libbox.Setup(
		&libbox.SetupOptions{
			BasePath:        params.BasePath,
			WorkingPath:     params.WorkingDir,
			TempPath:        params.TempDir,
			IsTVOS:          !tcpConn,
			FixAndroidStack: true,
		})

	hutils.RedirectStderr(fmt.Sprint(params.WorkingDir, "/data/stderr", params.Mode, ".log"))

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("libbox.Setup success %s %s %s %v", params.BasePath, params.WorkingDir, params.TempDir, tcpConn))

	sWorkingPath = params.WorkingDir
	os.Chdir(sWorkingPath)
	sTempPath = params.TempDir
	sUserID = os.Getuid()
	sGroupID = os.Getgid()

	var defaultWriter io.Writer
	if !params.Debug {
		defaultWriter = io.Discard
	}
	factory, err := log.New(
		log.Options{
			DefaultWriter: defaultWriter,
			BaseTime:      time.Now(),
			Observable:    true,
			// Options: option.LogOptions{
			// 	Disabled: false,
			// 	Level:    "trace",
			// 	Output:   "stdout",
			// },
		})
	static.CoreLogFactory = factory

	if err != nil {
		return E.Cause(err, "create logger")
	}

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("StartGrpcServerByMode %s %d\n", params.Listen, params.Mode))
	switch params.Mode {
	case SetupMode_OLD:
		statusPropagationPort = int64(params.FlutterStatusPort)
	// case SetupMode_GRPC_BACKGROUND_INSECURE:
	default:
		_, err := StartGrpcServerByMode(params.Listen, params.Mode)
		if err != nil {
			return err
		}
	}
	settings := db.GetTable[hcommon.AppSettings]()
	val, err := settings.Get("HiddifySettingsJson")
	if val == nil || err != nil {
		// if params.Mode == SetupMode_GRPC_BACKGROUND_INSECURE {
		_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: ""})
		if err != nil {
			Log(LogLevel_DEBUG, LogType_CORE, E.Cause(err, "ChangeHiddifySettings").Error())
		}
	} else {
		// settings := db.GetTable[hcommon.AppSettings]()
		_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: val.Value.(string)})
		if err != nil {
			Log(LogLevel_DEBUG, LogType_CORE, E.Cause(err, "ChangeHiddifySettings").Error())
		}

	}
	return InitHiddifyService()
}

func NewService(options option.Options) (*libbox.BoxService, error) {
	runtimeDebug.FreeOSMemory()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = filemanager.WithDefault(ctx, sWorkingPath, sTempPath, sUserID, sGroupID)
	urlTestHistoryStorage := urltest.NewHistoryStorage()
	ctx = service.ContextWithPtr(ctx, urlTestHistoryStorage)
	instance, err := B.New(B.Options{
		Context:           ctx,
		Options:           options,
		PlatformLogWriter: &LogInterface{},
	})
	if err != nil {
		cancel()
		return nil, E.Cause(err, "create service")
	}
	runtimeDebug.FreeOSMemory()
	service := libbox.NewBoxService(
		ctx,
		cancel,
		instance,
		service.FromContext[pause.Manager](ctx),
		urlTestHistoryStorage,
	)
	return &service, nil
}

func readOptions(configContent string) (option.Options, error) {
	var options option.Options
	err := options.UnmarshalJSON([]byte(configContent))
	if err != nil {
		return option.Options{}, E.Cause(err, "decode config")
	}
	return options, nil
}
