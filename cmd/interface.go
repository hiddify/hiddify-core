package cmd

import (
	"os"
	"os/user"
	"strconv"
	"time"

	"context"

	"github.com/sagernet/sing-box/experimental/deprecated"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"

	"github.com/spf13/cobra"
)

var (
	workingDir   string
	disableColor bool
)

var mainCommand = &cobra.Command{
	Use:              "HiddifyCli",
	PersistentPreRun: preRun,
}

func init() {
	mainCommand.AddCommand(commandService)
	mainCommand.AddCommand(commandGenerateCertification)

	mainCommand.PersistentFlags().StringVarP(&workingDir, "directory", "D", "", "set working directory")
	mainCommand.PersistentFlags().BoolVarP(&disableColor, "disable-color", "", false, "disable color output")

}

func ParseCli(args []string) error {
	mainCommand.SetArgs(args)
	err := mainCommand.Execute()
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func preRun(cmd *cobra.Command, args []string) {
	globalCtx = context.Background()
	sudoUser := os.Getenv("SUDO_USER")
	sudoUID, _ := strconv.Atoi(os.Getenv("SUDO_UID"))
	sudoGID, _ := strconv.Atoi(os.Getenv("SUDO_GID"))
	if sudoUID == 0 && sudoGID == 0 && sudoUser != "" {
		sudoUserObject, _ := user.Lookup(sudoUser)
		if sudoUserObject != nil {
			sudoUID, _ = strconv.Atoi(sudoUserObject.Uid)
			sudoGID, _ = strconv.Atoi(sudoUserObject.Gid)
		}
	}
	if sudoUID > 0 && sudoGID > 0 {
		globalCtx = filemanager.WithDefault(globalCtx, "", "", sudoUID, sudoGID)
	}
	if disableColor {
		log.SetStdLogger(log.NewDefaultFactory(context.Background(), log.Formatter{BaseTime: time.Now(), DisableColors: true}, os.Stderr, "", nil, false).Logger())
	}
	if workingDir != "" {
		_, err := os.Stat(workingDir)
		if err != nil {
			filemanager.MkdirAll(globalCtx, workingDir, 0o777)
		}
		err = os.Chdir(workingDir)
		if err != nil {
			log.Fatal(err)
		}
	}
	// if len(configPaths) == 0 && len(configDirectories) == 0 {
	// 	configPaths = append(configPaths, "config.json")
	// }
	globalCtx = include.Context(service.ContextWith(globalCtx, deprecated.NewStderrManager(log.StdLogger())))
}
