package cmd

import (
	"fmt"

	"github.com/hiddify/hiddify-core/v2/profile"
	"github.com/sagernet/sing-box/experimental/libbox"

	// "github.com/hiddify/hiddify-core/extension_repository/cleanip_scanner"
	"github.com/spf13/cobra"
)

var commandProfile = &cobra.Command{
	Use:   "profile",
	Short: "profile",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := libbox.BaseContext(nil)
		res, err := profile.AddByUrl(ctx, args[0], "", false)
		fmt.Printf("res=%v Error! %v", res, err)
	},
}

func init() {
	mainCommand.AddCommand(commandProfile)
}
