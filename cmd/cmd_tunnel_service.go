package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcore/buildid"
	"github.com/hiddify/hiddify-core/v2/hcore/tunnelservice"

	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Use:   "tunnel run/start/stop/install/uninstall/activate/deactivate/exit/connect/disconnect/build-id",
	Short: "Tunnel Service run/start/stop/install/uninstall/activate/deactivate/exit/connect/disconnect/build-id",
	ValidArgs: []string{
		"run", "start", "stop", "install", "uninstall",
		"activate", "deactivate", "exit", "connect", "disconnect", "build-id",
	},
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]
		switch arg {
		case "build-id":
			// The app compares this to the running service's build.id stamp. Only Go embeds the id.
			fmt.Println(buildid.Get())
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
		case "install":
			// pkexec clears most of the environment; pass the owner as the second argument.
			if len(args) >= 2 && args[1] != "" {
				_ = os.Setenv("HIDDIFY_TUNNEL_OWNER", args[1])
			}
			code, out := tunnelservice.StartTunnelService("install")
			fmt.Printf("exitCode:%d msg=%s", code, out)
			if code != 0 {
				os.Exit(code)
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
