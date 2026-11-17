package mobile

import (
	hcore "github.com/hiddify/hiddify-core/v2/hcore"

	_ "net/http/pprof"

	_ "github.com/sagernet/gomobile"
	"github.com/sagernet/sing-box/experimental/libbox"
)

type SetupOptions struct {
	BasePath        string
	WorkingDir      string
	TempDir         string
	Listen          string
	Secret          string
	Debug           bool
	Mode            int
	FixAndroidStack bool
	// MemoryLimit is what the core may use on this device, in bytes; 0 for none of its own.
	MemoryLimit int64
	// LowMemory is set on a device short of memory (a low-RAM TV): relaying trades speed for memory.
	LowMemory bool
}

func Setup(opt *SetupOptions, platformInterface libbox.PlatformInterface) error {
	hcore.SetDeviceMemory(opt.MemoryLimit, opt.LowMemory)
	return hcore.Setup(&hcore.SetupRequest{
		BasePath:          opt.BasePath,
		WorkingDir:        opt.WorkingDir,
		TempDir:           opt.TempDir,
		FlutterStatusPort: 0,
		Listen:            opt.Listen,
		Debug:             opt.Debug,
		Mode:              hcore.SetupMode(opt.Mode),
		Secret:            opt.Secret,
		FixAndroidStack:   opt.FixAndroidStack,
	}, platformInterface)

	// return hcore.Start(17078)
}

// func Start(configPath string, configContent string, platformInterface libbox.PlatformInterface) (*hcore.CoreInfoResponse, error) {
// 	state, err := hcore.StartWithPlatformInterface(&hcore.StartRequest{
// 		ConfigContent: configContent,
// 		ConfigPath:    configPath,
// 	}, platformInterface)
// 	return state, err
// }

func Start(configPath string, configContent string) error {
	_, err := hcore.StartService(libbox.BaseContext(nil), &hcore.StartRequest{
		ConfigPath:    configPath,
		ConfigContent: configContent,
	})
	return err
}

// CoreStoppedListener is told when the core stops by itself, with the reason (a start, stop or
// reload that failed): the platform's service takes itself down, as the app does when it runs.
type CoreStoppedListener interface {
	OnCoreStopped(messageType int32, message string)
}

// SetCoreStoppedListener sets the listener told when the core stops by itself; nil removes it.
// It is called on a thread of its own, never inside a call into the core.
func SetCoreStoppedListener(listener CoreStoppedListener) {
	if listener == nil {
		hcore.SetCoreStoppedHandler(nil)
		return
	}
	hcore.SetCoreStoppedHandler(func(msgType hcore.MessageType, message string) {
		listener.OnCoreStopped(int32(msgType), message)
	})
}

func Stop() error {
	_, err := hcore.Stop()
	return err
}

func GetServerPublicKey() []byte {
	return hcore.GetGrpcServerPublicKey()
}

func AddGrpcClientPublicKey(clientPublicKey []byte) error {
	return hcore.AddGrpcClientPublicKey(clientPublicKey)
}

func Close(mode int) {
	hcore.Close(hcore.SetupMode(mode))
}

func Test() string {
	return "Hello from mobile"
}

func Pause() {
	hcore.Pause()
}

func Wake() {
	hcore.Wake()
}
