package main

import (
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var commandServer *libbox.CommandServer

type CommandServerHandler struct{}

func (csh *CommandServerHandler) ServiceReload() error {
	log.Trace("[Command Server Handler] Reloading service")
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
	log.Trace("[Command Server Handler] Getting system proxy status")
	return &libbox.SystemProxyStatus{Available: true, Enabled: false}
}

func (csh *CommandServerHandler) SetSystemProxyEnabled(isEnabled bool) error {
	log.Trace("[Command Server Handler] Setting system proxy status")
	return csh.ServiceReload()
}

func startCommandServer() error {
	log.Trace("[Command Server Handler] Starting command server")
	commandServer = libbox.NewCommandServer(&CommandServerHandler{}, 300)
	return commandServer.Start()
}
