package global

import (
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var (
	statusClient *libbox.CommandClient
	groupClient  *libbox.CommandClient
)

func StartCommand(command int32, port int64, logFactory log.Factory) error {
	switch command {
	case libbox.CommandStatus:
		statusClient = libbox.NewCommandClient(
			&CommandClientHandler{
				port:   port,
				logger: logFactory.NewLogger("[Status Command Client]"),
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandStatus,
				StatusInterval: 1000000000,
			},
		)
		return statusClient.Connect()
	case libbox.CommandGroup:
		groupClient = libbox.NewCommandClient(
			&CommandClientHandler{
				port:   port,
				logger: logFactory.NewLogger("[Group Command Client]"),
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroup,
				StatusInterval: 1000000000,
			},
		)
		return groupClient.Connect()
	}
	return nil
}

func StopCommand(command int32) error {
	switch command {
	case libbox.CommandStatus:
		err := statusClient.Disconnect()
		statusClient = nil
		return err
	case libbox.CommandGroup:
		err := groupClient.Disconnect()
		groupClient = nil
		return err
	}
	return nil
}
