package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiddify/libcore/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"

	"github.com/spf13/cobra"
)

var commandBuildOutputPath string

var commandBuild = &cobra.Command{
	Use:   "build",
	Short: "Build configuration",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var optionsPath string
		if len(args) > 1 {
			optionsPath = args[1]
		}
		err := build(args[0], optionsPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var commandCheck = &cobra.Command{
	Use:   "check",
	Short: "Check configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := check(args[0])
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	commandBuild.Flags().StringVarP(&commandBuildOutputPath, "output", "o", "", "write result to file path instead of stdout")
	mainCommand.AddCommand(commandBuild)
	mainCommand.AddCommand(commandCheck)
}

func build(path string, optionsPath string) error {
	if workingDir != "" {
		path = filepath.Join(workingDir, path)
		if optionsPath != "" {
			optionsPath = filepath.Join(workingDir, optionsPath)
		}
		os.Chdir(workingDir)
	}
	options, err := readConfigAt(path)
	if err != nil {
		return err
	}
	configOptions := config.DefaultConfigOptions()
	if optionsPath != "" {
		configOptions, err = readConfigOptionsAt(optionsPath)
		if err != nil {
			return err
		}
	}
	config, err := config.BuildConfigJson(*configOptions, *options)
	if err != nil {
		return err
	}
	if commandBuildOutputPath != "" {
		outputPath, _ := filepath.Abs(filepath.Join(workingDir, commandBuildOutputPath))
		err = os.WriteFile(outputPath, []byte(config), 0644)
		if err != nil {
			return err
		}
		fmt.Println("result successfully written to ", outputPath)
		// libbox.Setup(outputPath, workingDir, workingDir, true)
		// instance, err := NewService(*patchedOptions)
	} else {
		os.Stdout.WriteString(config)
	}
	return nil
}

func check(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return libbox.CheckConfig(string(content))
}

func readConfigAt(path string) (*option.Options, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var options option.Options
	err = options.UnmarshalJSON(content)
	if err != nil {
		return nil, err
	}
	return &options, nil
}

func readConfigOptionsAt(path string) (*config.ConfigOptions, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var options config.ConfigOptions
	err = json.Unmarshal(content, &options)

	if err != nil {
		return nil, err
	}
	if options.Warp.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(options.Warp.WireguardConfigStr), &options.Warp.WireguardConfig)
		if err != nil {
			return nil, err
		}
	}

	return &options, nil
}
