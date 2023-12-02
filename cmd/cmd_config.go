package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiddify/libcore/shared"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"

	"github.com/spf13/cobra"
)

var commandBuild = &cobra.Command{
	Use:   "build",
	Short: "Build configuration",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := build(args[0], args[1])
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	mainCommand.AddCommand(commandBuild)
}

func build(path string, optionsPath string) error {
	if workingDir != "" {
		path = filepath.Join(workingDir, path)
		optionsPath = filepath.Join(workingDir, optionsPath)
	}
	options, err := readConfigAt(path)
	if err != nil {
		return err
	}
	configOptions, err := readConfigOptionsAt(optionsPath)
	if err != nil {
		return err
	}
	config, err := shared.BuildConfigJson(*configOptions, *options)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", config)
	return err
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

func readConfigOptionsAt(path string) (*shared.ConfigOptions, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var options shared.ConfigOptions
	err = json.Unmarshal(content, &options)
	if err != nil {
		return nil, err
	}
	return &options, nil
}
