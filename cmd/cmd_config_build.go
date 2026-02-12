package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/v2/config"
	hcore "github.com/hiddify/hiddify-core/v2/hcore"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"

	"github.com/spf13/cobra"
)

var (
	hiddifySettingPath     string
	configPath             string
	defaultConfigs         config.HiddifyOptions = *config.DefaultHiddifyOptions()
	commandBuildOutputPath string
)

var commandBuild = &cobra.Command{
	Use:   "build",
	Short: "Build configuration",
	Run: func(cmd *cobra.Command, args []string) {
		err := build(configPath, hiddifySettingPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var generateConfig = &cobra.Command{
	Use:   "gen",
	Short: "gen configuration",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := libbox.BaseContext(nil)
		conf, err := hcore.GenerateConfig(ctx, &hcore.GenerateConfigRequest{
			Path: args[0],
		})
		if err != nil {
			log.Fatal(err)
		}
		log.Debug(string(conf.ConfigContent))
	},
}

func init() {
	commandBuild.Flags().StringVarP(&commandBuildOutputPath, "output", "o", "", "write result to file path instead of stdout")
	addHConfigFlags(commandBuild)

	mainCommand.AddCommand(commandBuild)
	mainCommand.AddCommand(generateConfig)
}

func build(path string, optionsPath string) error {

	ctx := libbox.BaseContext(nil)
	var err error

	hiddifyOptions := &defaultConfigs // config.DefaultHiddifyOptions()
	if optionsPath != "" {
		hiddifyOptions, err = readHiddifyOptionsAt(optionsPath)
		if err != nil {
			return err
		}
	}

	config, err := config.BuildConfigJson(ctx, hiddifyOptions, &config.ReadOptions{Path: path})
	if err != nil {
		return err
	}
	if commandBuildOutputPath != "" {
		outputPath, _ := filepath.Abs(filepath.Join(workingDir, commandBuildOutputPath))
		err = os.WriteFile(outputPath, []byte(config), 0o644)
		if err != nil {
			return err
		}
		fmt.Println("result successfully written to ", outputPath)
		// libbox.Setup(outputPath, workingDir, workingDir, true)
		// instance, err := NewService(*patchedOptions)
	} else {
		os.Stdout.WriteString(string(config))
	}
	return nil
}

func checkConfig(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return libbox.CheckConfig(string(content))
}

func readConfigAt(ctx context.Context, path string) (*option.Options, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var options option.Options
	err = options.UnmarshalJSONContext(ctx, content)
	if err != nil {
		return nil, err
	}
	return &options, nil
}

func readHiddifyOptionsAt(path string) (*config.HiddifyOptions, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var options config.HiddifyOptions
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
	if options.Warp2.WireguardConfigStr != "" {
		err := json.Unmarshal([]byte(options.Warp2.WireguardConfigStr), &options.Warp2.WireguardConfig)
		if err != nil {
			return nil, err
		}
	}

	return &options, nil
}

func addHConfigFlags(commandRun *cobra.Command) {

	commandRun.MarkFlagRequired("config")
	commandRun.Flags().StringVarP(&hiddifySettingPath, "hiddify", "d", "", "Hiddify Setting JSON Path")
	commandRun.Flags().BoolVar(&defaultConfigs.EnableFullConfig, "full-config", false, "allows including tags other than output")
	commandRun.Flags().StringVar(&defaultConfigs.LogLevel, "log", "warn", "log level")
	commandRun.Flags().BoolVar(&defaultConfigs.InboundOptions.EnableTun, "tun", false, "Enable Tun")
	commandRun.Flags().BoolVar(&defaultConfigs.InboundOptions.EnableTunService, "tun-service", false, "Enable Tun Service")
	commandRun.Flags().BoolVar(&defaultConfigs.InboundOptions.SetSystemProxy, "system-proxy", false, "Enable System Proxy")
	commandRun.Flags().Uint16Var(&defaultConfigs.InboundOptions.MixedPort, "in-proxy-port", 12334, "Input Mixed Port")
	commandRun.Flags().BoolVar(&defaultConfigs.TLSTricks.EnableFragment, "fragment", false, "Enable Fragment")
	commandRun.Flags().StringVar(&defaultConfigs.TLSTricks.FragmentSize, "fragment-size", "2-4", "FragmentSize")
	commandRun.Flags().StringVar(&defaultConfigs.TLSTricks.FragmentSleep, "fragment-sleep", "2-4", "FragmentSleep")

	commandRun.Flags().BoolVar(&defaultConfigs.TLSTricks.EnablePadding, "padding", false, "Enable Padding")
	commandRun.Flags().StringVar(&defaultConfigs.TLSTricks.PaddingSize, "padding-size", "1300-1400", "PaddingSize")

	commandRun.Flags().BoolVar(&defaultConfigs.TLSTricks.MixedSNICase, "mixed-sni-case", false, "MixedSNICase")

	commandRun.Flags().StringVar(&defaultConfigs.RemoteDnsAddress, "dns-remote", "1.1.1.1", "RemoteDNS (1.1.1.1, https://1.1.1.1/dns-query)")
	commandRun.Flags().StringVar(&defaultConfigs.DirectDnsAddress, "dns-direct", "1.1.1.1", "DirectDNS (1.1.1.1, https://1.1.1.1/dns-query)")
	commandRun.Flags().StringVar(&defaultConfigs.ClashApiSecret, "web-secret", "", "Web Server Secret")
	commandRun.Flags().Uint16Var(&defaultConfigs.ClashApiPort, "web-port", 6756, "Web Server Port")
	commandRun.Flags().StringVar(&defaultConfigs.LogLevel, "log-level", "warn", "log level")
}
