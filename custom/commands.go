package main

import "github.com/sagernet/sing-box/experimental/libbox"

var (
	statusCommand *libbox.CommandClient
)

func StartCommand(command int32, port int64) error {
	switch command {
	case libbox.CommandStatus:
		statusCommand = libbox.NewCommandClient(
			&CommandClientHandler{port: port},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandStatus,
				StatusInterval: 1000000000,
			},
		)
		return statusCommand.Connect()
	}
	return nil
}

func StopCommand(command int32) error {
	switch command {
	case libbox.CommandStatus:
		err := statusCommand.Disconnect()
		statusCommand = nil
		return err
	}
	return nil
}
