package service

import (
	"github.com/hiddify/libcore/global"
	"github.com/hiddify/libcore/web"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

type hiddifyNext struct{}

var port int

func (m *hiddifyNext) Start(s service.Service) error {
	go m.run()
	return nil
}
func (m *hiddifyNext) Stop(s service.Service) error {
	err := global.StopService()
	if err != nil {
		return err
	}
	return nil
}
func (m *hiddifyNext) run() {
	web.StartWebServer(port)
}
func StartService(cmd *cobra.Command, args []string) {
	port, _ = cmd.Flags().GetInt("port")
	svcConfig := &service.Config{
		Name:        "hiddify_next_core",
		DisplayName: "hiddify next core",
		Description: "@hiddify_com set this",
	}
	prg := &hiddifyNext{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		panic("Error: " + err.Error())
	}
	err = svc.Run()
	if err != nil {
		panic("Error: " + err.Error())
	}
}
func StopService(cmd *cobra.Command, args []string) {
	svcConfig := &service.Config{
		Name:        "hiddify_next_core",
		DisplayName: "hiddify next core",
		Description: "@hiddify_com set this",
	}
	prg := &hiddifyNext{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		panic("Error: " + err.Error())
	}
	err = svc.Stop()
	if err != nil {
		panic("Error: " + err.Error())
	}
}
func InstallService(cmd *cobra.Command, args []string) {
	svcConfig := &service.Config{
		Name:        "hiddify_next_core",
		DisplayName: "hiddify next core",
		Description: "@hiddify_com set this",
	}
	prg := &hiddifyNext{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		panic("Error: " + err.Error())
	}
	err = svc.Install()
	if err != nil {
		panic("Error: " + err.Error())
	}
}
