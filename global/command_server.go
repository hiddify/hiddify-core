package global

import (
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var commandServer *libbox.CommandServer

type CommandServerHandler struct {
	logger log.Logger
}

func (csh *CommandServerHandler) ServiceReload() error {
	csh.logger.Trace("Reloading service")
	propagateStatus(Starting)
	if commandServer != nil {
		commandServer.SetService(nil)
		commandServer = nil
	}
	if box != nil {
		box.Close()
		box = nil
	}
	return startService(true)
}

func (csh *CommandServerHandler) GetSystemProxyStatus() *libbox.SystemProxyStatus {
	csh.logger.Trace("Getting system proxy status")
	return &libbox.SystemProxyStatus{Available: true, Enabled: false}
}

func (csh *CommandServerHandler) SetSystemProxyEnabled(isEnabled bool) error {
	csh.logger.Trace("Setting system proxy status, enabled? ", isEnabled)
	return csh.ServiceReload()
}

func startCommandServer(logFactory log.Factory) error {
	logger := logFactory.NewLogger("[Command Server Handler]")
	logger.Trace("Starting command server")
	commandServer = libbox.NewCommandServer(&CommandServerHandler{logger: logger}, 300)
	return commandServer.Start()
}
