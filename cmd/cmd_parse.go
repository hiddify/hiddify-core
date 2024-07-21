package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/config"
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
	if workingDir != "" {
		path = filepath.Join(workingDir, path)
	}
	config, err := config.ParseConfig(path, true)
	if err != nil {
		return err
	}
	if commandParseOutputPath != "" {
		outputPath, _ := filepath.Abs(filepath.Join(workingDir, commandParseOutputPath))
		err = os.WriteFile(outputPath, config, 0644)
		if err != nil {
			return err
		}
		fmt.Println("result successfully written to ", outputPath)
	} else {
		os.Stdout.Write(config)
	}
	return nil
}
