package v2

import (
	pb "github.com/hiddify/libcore/hiddifyrpc"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var commandServer *libbox.CommandServer

type CommandServerHandler struct {
	logger log.Logger
}

func (csh *CommandServerHandler) ServiceReload() error {
	csh.logger.Trace("Reloading service")
	SetCoreStatus(pb.CoreState_STARTING, pb.MessageType_EMPTY, "")

	if commandServer != nil {
		commandServer.SetService(nil)
		commandServer = nil
	}
	if Box != nil {
		Box.Close()
		Box = nil
	}
	_, err := StartService(&pb.StartRequest{
		EnableOldCommandServer: true,
		DelayStart:             true,
	})
	return err
}

func (csh *CommandServerHandler) GetSystemProxyStatus() *libbox.SystemProxyStatus {
	csh.logger.Trace("Getting system proxy status")
	return &libbox.SystemProxyStatus{Available: true, Enabled: false}
}

func (csh *CommandServerHandler) SetSystemProxyEnabled(isEnabled bool) error {
	csh.logger.Trace("Setting system proxy status, enabled? ", isEnabled)
	return csh.ServiceReload()
}

func (csh *CommandServerHandler) PostServiceClose() {

}
func startCommandServer(logFactory log.Factory) error {
	logger := logFactory.NewLogger("[Command Server Handler]")
	logger.Trace("Starting command server")
	commandServer = libbox.NewCommandServer(&CommandServerHandler{logger: logger}, 300)
	return commandServer.Start()
}
