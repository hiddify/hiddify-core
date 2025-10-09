package v2

import (
	"github.com/sagernet/sing-box/experimental/libbox"
)

var (
	oldStatusClient   *libbox.CommandClient
	oldGroupClient   *libbox.CommandClient
)

func StartCommand(command int32, port int64) error {
	switch command {
	case libbox.CommandStatus:
		oldStatusClient = libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[Status Command Client]"),
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandStatus,
				StatusInterval: 1000000000, // 1000ms debounce
			},
		)
		return oldStatusClient.Connect()
	case libbox.CommandGroup:
		oldGroupClient = libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[Group Command Client]"),
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroup,
				StatusInterval: 300000000, // 300ms debounce
			},
		)
		return oldGroupClient.Connect()
	default:
		return nil
	}
}

func StopCommand(command int32) error {
	switch command {
	case libbox.CommandStatus:
		if oldStatusClient != nil {
			err := oldStatusClient.Disconnect()
			oldStatusClient = nil
			return err
		}
	case libbox.CommandGroup:
		if oldGroupClient != nil {
			err := oldGroupClient.Disconnect()
			oldGroupClient = nil
			return err
		}
	}
	return nil
}
