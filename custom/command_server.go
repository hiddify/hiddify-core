package main

import "github.com/sagernet/sing-box/experimental/libbox"

var commandServer *libbox.CommandServer

type CommandServerHandler struct{}

func (csh *CommandServerHandler) ServiceReload() error {
	return nil
}

func startCommandServer() error {
	commandServer = libbox.NewCommandServer(&CommandServerHandler{}, 300)
	return commandServer.Start()
}
