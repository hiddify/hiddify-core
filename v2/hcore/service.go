package hcore

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	runtimeDebug "runtime/debug"
	"time"

	common "github.com/hiddify/hiddify-core/v2/common"
	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/db"
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

type SetupMode int

const (
	OLD                      SetupMode = 0
	GRPC_NORMAL              SetupMode = 1
	GRPC_BACKGROUND          SetupMode = 2
	GRPC_NORMAL_INSECURE     SetupMode = 3
	GRPC_BACKGROUND_INSECURE SetupMode = 4
)

type SetupParameters struct {
	BasePath          string
	WorkingDir        string
	TempDir           string
	FlutterStatusPort int64
	Listen            string
	Secret            string
	Debug             bool
	Mode              SetupMode
}

func Setup(params SetupParameters) error {
	defer config.DeferPanicToError("setup", func(err error) {
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
	})
	tcpConn := runtime.GOOS == "windows" // TODO add TVOS
	libbox.Setup(params.BasePath, params.WorkingDir, params.TempDir, tcpConn)
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
	coreLogFactory = factory

	if err != nil {
		return E.Cause(err, "create logger")
	}

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("StartGrpcServerByMode %s %d\n", params.Listen, params.Mode))
	switch params.Mode {
	case OLD:
		statusPropagationPort = int64(params.FlutterStatusPort)
	default:
		_, err := StartGrpcServerByMode(params.Listen, params.Mode)
		if err != nil {
			return err
		}
	}
	settings := db.GetTable[common.AppSettings]()
	val, err := settings.Get("HiddifySettingsJson")
	if val == nil || err != nil {
		_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: ""})
		if err != nil {
			Log(LogLevel_DEBUG, LogType_CORE, E.Cause(err, "ChangeHiddifySettings").Error())
		}
	} else {
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
