package cmd

import (
	"context"

	box "github.com/sagernet/sing-box"
)

// var commandCheck = &cobra.Command{
// 	Use:   "check",
// 	Short: "Check configuration",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		err := check()
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 	},
// 	Args: cobra.NoArgs,
// }

// func init() {
// 	mainCommand.AddCommand(commandCheck)
// }

func check() error {
	options, err := readConfigAndMerge()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(globalCtx)
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err == nil {
		instance.Close()
	}
	cancel()
	return err
}
