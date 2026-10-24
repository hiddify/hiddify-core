package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcore/tunnelservice"

	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Use:   "tunnel run/start/stop/install/uninstall/activate/deactivate/exit/connect/disconnect",
	Short: "Tunnel Service run/start/stop/install/uninstall/activate/deactivate/exit/connect/disconnect",
	ValidArgs: []string{
		"run", "start", "stop", "install", "uninstall",
		"activate", "deactivate", "exit", "connect", "disconnect",
	},
	Args: cobra.RangeArgs(1, 2),
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
		case "connect":
			if len(args) != 2 {
				fmt.Fprintln(os.Stderr, "usage: tunnel connect <request.json>")
				os.Exit(2)
			}
			if err := tunnelservice.ConnectFromFile(args[1]); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		case "disconnect":
			if err := tunnelservice.DisconnectTunnel(); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		default:
			code, out := tunnelservice.StartTunnelService(arg)
			fmt.Printf("exitCode:%d msg=%s", code, out)
			if code != 0 {
				os.Exit(code)
			}
		}
	},
}
