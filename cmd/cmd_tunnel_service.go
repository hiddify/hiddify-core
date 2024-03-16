package cmd

import (
	"fmt"

	v2 "github.com/hiddify/libcore/v2"

	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Use:       "tunnel run/start/stop/install/uninstall",
	Short:     "Tunnel Service run/start/stop/install/uninstall",
	ValidArgs: []string{"run", "start", "stop", "install", "uninstall"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]
		code, out := v2.StartTunnelService(arg)
		fmt.Printf("exitCode:%d msg=%s", code, out)
	},
}
