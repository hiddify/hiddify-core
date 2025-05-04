package cmd

import (
	"fmt"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcore/tunnelservice"

	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Use:       "tunnel run/start/stop/install/uninstall/activate/deactivate/exit",
	Short:     "Tunnel Service run/start/stop/install/uninstall/activate/deactivate/exit",
	ValidArgs: []string{"run", "start", "stop", "install", "uninstall", "activate", "deactivate", "exit"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]
		switch arg {
		case "activate":
			tunnelservice.ActivateTunnelService(&tunnelservice.TunnelStartRequest{
				Ipv6:       true,
				ServerPort: 12334,
				Stack:      "gvisor",
			})

			<-time.After(1 * time.Second)

		case "deactivate":
			tunnelservice.DeactivateTunnelServiceForce()
		case "exit":
			tunnelservice.ExitTunnelService()
		default:
			code, out := tunnelservice.StartTunnelService(arg)
			fmt.Printf("exitCode:%d msg=%s", code, out)
		}
	},
}
