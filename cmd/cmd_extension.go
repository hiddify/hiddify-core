package cmd

import (
	_ "github.com/hiddify/hiddify-core/extension/repository"
	"github.com/hiddify/hiddify-core/extension/server"
	"github.com/spf13/cobra"
)

var extension_id string

var commandExtension = &cobra.Command{
	Use:   "extension",
	Short: "extension configuration",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		server.StartTestExtensionServer()
	},
}

func init() {
	// commandWarp.Flags().StringVarP(&warpKey, "key", "k", "", "warp key")
	mainCommand.AddCommand(commandExtension)
}
