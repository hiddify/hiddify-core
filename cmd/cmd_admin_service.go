package main

import (
	"fmt"

	"github.com/hiddify/libcore/admin_service"

	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Use:   "admin-service",
	Short: "Sign box service start/stop/install/uninstall",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		arg := ""
		if len(args) > 1 {
			arg = args[1]
		}
		code, out := admin_service.StartService(arg)
		fmt.Printf("exitCode:%d msg=%s", code, out)

	},
}
