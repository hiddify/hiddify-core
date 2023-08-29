package main

import "github.com/sagernet/sing-box/experimental/libbox"

var (
	statusCommand *libbox.CommandClient
	groupCommand  *libbox.CommandClient
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
	case libbox.CommandGroup:
		groupCommand = libbox.NewCommandClient(
			&CommandClientHandler{port: port},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroup,
				StatusInterval: 1000000000,
			},
		)
		return groupCommand.Connect()
	}
	return nil
}

func StopCommand(command int32) error {
	switch command {
	case libbox.CommandStatus:
		err := statusCommand.Disconnect()
		statusCommand = nil
		return err
	case libbox.CommandGroup:
		err := groupCommand.Disconnect()
		groupCommand = nil
		return err
	}
	return nil
}
