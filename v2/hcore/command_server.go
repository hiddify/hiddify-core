package hcore

// import (
// 	"github.com/sagernet/sing-box/experimental/libbox"
// )

// var commandServer *libbox.CommandServer

// type CommandServerHandler struct{}

// func (csh *CommandServerHandler) ServiceReload() error {
// 	Log(LogLevel_DEBUG, LogType_CORE, "Reloading service")

// 	Stop()
// 	_, err := StartService(&StartRequest{
// 		EnableOldCommandServer: true,
// 		DelayStart:             true,
// 	}, nil)
// 	return err
// }

// func (csh *CommandServerHandler) GetSystemProxyStatus() *libbox.SystemProxyStatus {
// 	Log(LogLevel_DEBUG, LogType_CORE, "Getting system proxy status")
// 	return &libbox.SystemProxyStatus{Available: true, Enabled: false}
// }

// func (csh *CommandServerHandler) SetSystemProxyEnabled(isEnabled bool) error {
// 	Log(LogLevel_DEBUG, LogType_CORE, "Setting system proxy status, enabled? ", isEnabled)
// 	return csh.ServiceReload()
// }

// func (csh *CommandServerHandler) PostServiceClose() {
// 	if commandServer != nil {
// 		commandServer.Close()
// 		commandServer.SetService(nil)
// 	}
// 	commandServer = nil
// }

// func startCommandServer(sbox *libbox.BoxService) error {
// 	commandServer = libbox.NewCommandServer(&CommandServerHandler{}, 300)
// 	commandServer.SetService(sbox)
// 	return commandServer.Start()
// }
