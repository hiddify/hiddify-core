package main

import (
	"github.com/hiddify/libcore/service"
	"github.com/spf13/cobra"
)

var commandServiceStop = &cobra.Command{
	Use:   "stop",
	Short: "stop sign box",
	Run:   service.StopService,
}
