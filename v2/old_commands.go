package v2

import (
	"github.com/sagernet/sing-box/experimental/libbox"
)

var (
	oldStatusClient        *libbox.CommandClient
	oldGroupClient         *libbox.CommandClient
	oldGroupInfoOnlyClient *libbox.CommandClient
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
				StatusInterval: 1000000000, //1000ms debounce
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
				StatusInterval: 300000000, //300ms debounce
			},
		)
		return oldGroupClient.Connect()
	case libbox.CommandGroupInfoOnly:
		oldGroupInfoOnlyClient = libbox.NewCommandClient(
			&OldCommandClientHandler{
				port:   port,
				logger: coreLogFactory.NewLogger("[GroupInfoOnly Command Client]"),
			},
			&libbox.CommandClientOptions{
				Command:        libbox.CommandGroupInfoOnly,
				StatusInterval: 300000000, //300ms debounce
			},
		)
		return oldGroupInfoOnlyClient.Connect()
	}
	return nil
}

func StopCommand(command int32) error {
	switch command {
	case libbox.CommandStatus:
		err := oldStatusClient.Disconnect()
		oldStatusClient = nil
		return err
	case libbox.CommandGroup:
		err := oldGroupClient.Disconnect()
		oldGroupClient = nil
		return err
	case libbox.CommandGroupInfoOnly:
		err := oldGroupInfoOnlyClient.Disconnect()
		oldGroupInfoOnlyClient = nil
		return err
	}
	return nil
}
