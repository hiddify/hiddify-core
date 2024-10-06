package cmd

import (
	hcore "github.com/hiddify/hiddify-core/v2/hcore"
	"github.com/spf13/cobra"
)

var commandRun = &cobra.Command{
	Use:   "run",
	Short: "run",
	Args:  cobra.OnlyValidArgs,
	Run:   runCommand,
}

func init() {
	// commandRun.PersistentFlags().BoolP("help", "", false, "help for this command")
	// commandRun.Flags().StringVarP(&hiddifySettingPath, "hiddify", "d", "", "Hiddify Setting JSON Path")

	addHConfigFlags(commandRun)

	mainCommand.AddCommand(commandRun)
}

func runCommand(cmd *cobra.Command, args []string) {
	hcore.Setup("./tmp", "./", "./tmp", 0, false)
	hcore.RunStandalone(hiddifySettingPath, configPath, defaultConfigs)
}
