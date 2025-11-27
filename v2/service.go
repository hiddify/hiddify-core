package v2

import (
	json "github.com/goccy/go-json"
	"io"
	runtimeDebug "runtime/debug"
	"time"

	"github.com/hiddify/hiddify-core/config"
	"github.com/hiddify/hiddify-core/v2/service_manager"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

var (
    sWorkingPath          string
    sTempPath             string
    statusPropagationPort int64
)

func InitHiddifyService() error {
	return service_manager.StartServices()
}

func Setup(basePath string, workingPath string, tempPath string, statusPort int64, debug bool) error {
    statusPropagationPort = int64(statusPort)
    _ = libbox.Setup(&libbox.SetupOptions{
        BasePath:    basePath,
        WorkingPath: workingPath,
        TempPath:    tempPath,
        // Username: "", IsTVOS: false, FixAndroidStack: false
    })
    sWorkingPath = workingPath
    sTempPath = tempPath

    var defaultWriter io.Writer
    if !debug {
        defaultWriter = io.Discard
    }
    factory, err := log.New(log.Options{
        DefaultWriter: defaultWriter,
        BaseTime:      time.Now(),
        Observable:    true,
    })
    coreLogFactory = factory

    if err != nil {
        return E.Cause(err, "create logger")
    }
    return InitHiddifyService()
}

func NewService(options option.Options) (*libbox.BoxService, error) {
    runtimeDebug.FreeOSMemory()
    // Convert options to JSON for libbox.NewService API
    cfg, err := config.ToJson(options)
    if err != nil {
        return nil, E.Cause(err, "encode config")
    }
    bs, err := libbox.NewService(cfg, &platformStub{})
    if err != nil {
        return nil, E.Cause(err, "create service")
    }
    runtimeDebug.FreeOSMemory()
    return bs, nil
}

func readOptions(configContent string) (option.Options, error) {
    var options option.Options
    if err := json.Unmarshal([]byte(configContent), &options); err != nil {
        return option.Options{}, E.Cause(err, "decode config")
    }
    return options, nil
}
