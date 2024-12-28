package mobile

import (
	hcore "github.com/hiddify/hiddify-core/v2/hcore"

	_ "github.com/sagernet/gomobile"

	"github.com/sagernet/sing-box/experimental/libbox"
)

func Setup(baseDir string, workingDir string, tempDir string, mode int, listen string, secret string, debug bool, platformInterface libbox.PlatformInterface) error {
	return hcore.Setup(&hcore.SetupRequest{
		BasePath:          baseDir,
		WorkingDir:        workingDir,
		TempDir:           tempDir,
		FlutterStatusPort: 0,
		Listen:            listen,
		Debug:             debug,
		Mode:              hcore.SetupMode(mode),
		Secret:            secret,
	}, platformInterface)

	// return hcore.Start(17078)
}

func BuildConfig(configPath string) (string, error) {
	return hcore.BuildConfigJson(&hcore.StartRequest{
		ConfigPath: configPath,
	})
}

// func Start(configPath string, configContent string, platformInterface libbox.PlatformInterface) (*hcore.CoreInfoResponse, error) {
// 	state, err := hcore.StartWithPlatformInterface(&hcore.StartRequest{
// 		ConfigContent: configContent,
// 		ConfigPath:    configPath,
// 	}, platformInterface)
// 	return state, err
// }

func Start(configPath string, configContent string) error {
	_, err := hcore.StartService(&hcore.StartRequest{
		ConfigPath:    configPath,
		ConfigContent: configContent,
	})
	return err
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
