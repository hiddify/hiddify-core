package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/spf13/cobra"
)

var commandParseOutputPath string

var commandParse = &cobra.Command{
	Use:   "parse",
	Short: "Parse configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := parse(args[0])
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	commandParse.Flags().StringVarP(&commandParseOutputPath, "output", "o", "", "write result to file path instead of stdout")

	mainCommand.AddCommand(commandParse)
}

func parse(path string) error {
	ctx := libbox.BaseContext(nil)

	configStr, err := config.ParseConfigBytes(ctx, &config.ReadOptions{Path: path}, true, nil, false)
	if err != nil {
		return err
	}
	if err := libbox.CheckConfig(string(configStr)); err != nil {
		return fmt.Errorf("config check failed: %w", err)
	}

	if commandParseOutputPath != "" {
		outputPath, _ := filepath.Abs(filepath.Join(workingDir, commandParseOutputPath))
		err = os.WriteFile(outputPath, configStr, 0o644)
		if err != nil {
			return err
		}
		fmt.Println("result successfully written to ", outputPath)
	} else {
		os.Stdout.Write(configStr)
	}
	return nil
}
