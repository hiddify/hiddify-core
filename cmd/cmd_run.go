package cmd

import (
	v2 "github.com/hiddify/libcore/v2"

	"github.com/spf13/cobra"
)

var (
	hiddifySettingPath string
	configPath         string
)

var commandRun = &cobra.Command{
	Use:   "run",
	Short: "run",
	Args:  cobra.OnlyValidArgs,
	Run:   runCommand,
}

func init() {
	commandRun.PersistentFlags().BoolP("help", "", false, "help for this command")
	commandRun.Flags().StringVarP(&hiddifySettingPath, "hiddify", "h", "", "Hiddify Setting JSON Path")
	commandRun.Flags().StringVarP(&configPath, "config", "c", "", "proxy config path or url")

	commandRun.MarkFlagRequired("config")

	mainCommand.AddCommand(commandRun)
}

func runCommand(cmd *cobra.Command, args []string) {
	v2.RunStandalone(hiddifySettingPath, configPath)
}
