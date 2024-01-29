package main

import (
	"github.com/hiddify/libcore/service"
	"github.com/spf13/cobra"
)

var commandServiceInstall = &cobra.Command{
	Use:   "install",
	Short: "install the service",
	Run:   service.InstallService,
}
