package main

import (
	"github.com/hiddify/libcore/service"
	"github.com/spf13/cobra"
)

var commandServiceStart = &cobra.Command{
	Use:   "start",
	Short: "Start a sign box instance",
	Run:   service.StartService,
}
