package main

import (
	"os"
	"time"

	"context"

	"github.com/sagernet/sing-box/log"

	"github.com/spf13/cobra"
)

var (
	workingDir   string
	disableColor bool
)

var mainCommand = &cobra.Command{
	Use:              "hiddify-next",
	PersistentPreRun: preRun,
}

func init() {
	mainCommand.AddCommand(commandService)
	mainCommand.AddCommand(commandGenerateCertification)

	mainCommand.PersistentFlags().StringVarP(&workingDir, "directory", "D", "", "set working directory")
	mainCommand.PersistentFlags().BoolVarP(&disableColor, "disable-color", "", false, "disable color output")

}

func main() {
	if err := mainCommand.Execute(); err != nil {
		log.Fatal(err)
	}
}

func preRun(cmd *cobra.Command, args []string) {
	if disableColor {
		log.SetStdLogger(log.NewDefaultFactory(context.Background(), log.Formatter{BaseTime: time.Now(), DisableColors: true}, os.Stderr, "", nil, false).Logger())
	}
	if workingDir != "" {
		_, err := os.Stat(workingDir)
		if err != nil {
			os.MkdirAll(workingDir, 0o0644)
		}
		if err := os.Chdir(workingDir); err != nil {
			log.Fatal(err)
		}
	}
}
