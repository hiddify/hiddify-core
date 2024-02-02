package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hiddify/libcore/config"
	"github.com/hiddify/libcore/global"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"

	"github.com/spf13/cobra"
)

var commandRunInputPath string

var commandRun = &cobra.Command{
	Use:   "run",
	Short: "run",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		err := runSingbox(commandRunInputPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	commandRun.Flags().StringVarP(&commandRunInputPath, "config", "c", "", "read config")
	mainCommand.AddCommand(commandRun)

}

func runSingbox(configPath string) error {
	options, err := readConfigAt(configPath)
	if err != nil {
		return err
	}
	options.Log = &option.LogOptions{}
	options.Log.Disabled = false
	options.Log.Level = "trace"
	options.Log.Output = ""
	options.Log.DisableColor = false

	err = global.SetupC("./", "./", "./tmp", false)
	if err != nil {
		return err
	}
	configStr, err := config.ToJson(*options)
	if err != nil {
		return err
	}
	go global.StartServiceC(false, configStr)
	fmt.Printf("Waiting for 30 seconds\n")
	// <-time.After(time.Second * 30)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	return err
}
