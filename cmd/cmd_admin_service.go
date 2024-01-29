package main

import (
	"github.com/hiddify/libcore/admin_service"

	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Use:   "admin-service",
	Short: "Sign box service start/stop/install/uninstall",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			admin_service.StartService("")
		}
		admin_service.StartService(args[1])
	},
}
