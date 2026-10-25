package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcore/buildid"
	"github.com/hiddify/hiddify-core/v2/hcore/adminbgservice"

	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Aliases: []string{"tunnel"},
	Use:   "admin_bgservice run/start/stop/install/uninstall/activate/deactivate/exit/connect/disconnect/build-id",
	Short: "Admin background service run/start/stop/install/uninstall/activate/deactivate/exit/connect/disconnect/build-id",
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
			adminbgservice.ActivateAdminBgService(&adminbgservice.AdminBgStartRequest{
				Ipv6:       true,
				ServerPort: 12334,
				Stack:      "gvisor",
			})
			<-time.After(1 * time.Second)
		case "deactivate":
			adminbgservice.DeactivateAdminBgServiceForce()
		case "exit":
			adminbgservice.ExitAdminBgService()
		case "connect":
			if len(args) != 2 {
				fmt.Fprintln(os.Stderr, "usage: admin_bgservice connect <request.json>")
				os.Exit(2)
			}
			if err := adminbgservice.ConnectFromFile(args[1]); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		case "disconnect":
			if err := adminbgservice.DisconnectAdminBg(); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		case "install":
			// pkexec clears most of the environment; pass the owner as the second argument.
			if len(args) >= 2 && args[1] != "" {
				_ = os.Setenv("HIDDIFY_ADMIN_BG_OWNER", args[1])
			}
			code, out := adminbgservice.StartAdminBgService("install")
			fmt.Printf("exitCode:%d msg=%s", code, out)
			if code != 0 {
				os.Exit(code)
			}
		default:
			code, out := adminbgservice.StartAdminBgService(arg)
			fmt.Printf("exitCode:%d msg=%s", code, out)
			if code != 0 {
				os.Exit(code)
			}
		}
	},
}
