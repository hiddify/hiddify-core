package cmd

import (
	json "github.com/goccy/go-json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiddify/hiddify-core/config"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	v2 "github.com/hiddify/hiddify-core/v2"
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
		conf, err := v2.GenerateConfig(&pb.GenerateConfigRequest{Path: args[0]})
		if err != nil {
			log.Fatal(err)
		}
		log.Debug(string(conf.ConfigContent))
	},
}

var commandCheck = &cobra.Command{
	Use:   "check",
	Short: "Check configuration",
	Run: func(cmd *cobra.Command, args []string) {
		err := check(configPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	commandBuild.Flags().StringVarP(&commandBuildOutputPath, "output", "o", "", "write result to file path instead of stdout")
	addHConfigFlags(commandBuild)
	mainCommand.AddCommand(commandBuild)
	mainCommand.AddCommand(generateConfig)
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
	HiddifyOptions := &defaultConfigs
	if optionsPath != "" {
		HiddifyOptions, err = readHiddifyOptionsAt(optionsPath)
		if err != nil {
			return err
		}
	}
	configStr, err := config.BuildConfigJson(*HiddifyOptions, *options)
	if err != nil {
		return err
	}
	if commandBuildOutputPath != "" {
		outputPath, _ := filepath.Abs(filepath.Join(workingDir, commandBuildOutputPath))
		if err := os.WriteFile(outputPath, []byte(configStr), 0o644); err != nil {
			return err
		}
		fmt.Println("result successfully written to ", outputPath)
	} else {
		os.Stdout.WriteString(configStr)
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
	if err := json.Unmarshal(content, &options); err != nil {
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
	if err := json.Unmarshal(content, &options); err != nil {
		return nil, err
	}
	// if options.Warp.WireguardConfigStr != "" {
	// 	if err := json.Unmarshal([]byte(options.Warp.WireguardConfigStr), &options.Warp.WireguardConfig); err != nil {
	// 		return nil, err
	// 	}
	// }
	// if options.Warp2.WireguardConfigStr != "" {
	// 	if err := json.Unmarshal([]byte(options.Warp2.WireguardConfigStr), &options.Warp2.WireguardConfig); err != nil {
	// 		return nil, err
	// 	}
	// }
	return &options, nil
}

func addHConfigFlags(commandRun *cobra.Command) {
	commandRun.Flags().StringVarP(&configPath, "config", "c", "", "proxy config path or url")
	commandRun.MarkFlagRequired("config")
	commandRun.Flags().BoolVar(&defaultConfigs.EnableFullConfig, "full-config", false, "allows including tags other than output")
	commandRun.Flags().StringVar(&defaultConfigs.LogLevel, "log", "warn", "log level")
	commandRun.Flags().BoolVar(&defaultConfigs.InboundOptions.EnableTun, "tun", false, "Enable Tun")
	commandRun.Flags().BoolVar(&defaultConfigs.InboundOptions.EnableTunService, "tun-service", false, "Enable Tun Service")
	commandRun.Flags().BoolVar(&defaultConfigs.InboundOptions.SetSystemProxy, "system-proxy", false, "Enable System Proxy")
	commandRun.Flags().Uint16Var(&defaultConfigs.InboundOptions.MixedPort, "in-proxy-port", 2334, "Input Mixed Port")
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
}
