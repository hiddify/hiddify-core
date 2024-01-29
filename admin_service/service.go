package admin_service

import (
	"log"

	"github.com/hiddify/libcore/global"
	"github.com/kardianos/service"
)

var logger service.Logger

type hiddifyNext struct{}

var port int = 18020

func (m *hiddifyNext) Start(s service.Service) error {
	go m.run()
	return nil
}
func (m *hiddifyNext) Stop(s service.Service) error {
	err := global.StopService()
	if err != nil {
		return err
	}
	// Stop should not block. Return with a few seconds.
	// <-time.After(time.Second * 1)
	return nil
}
func (m *hiddifyNext) run() {
	StartWebServer(port, false)
}

func StartService(goArg string) {
	svcConfig := &service.Config{
		Name:        "Hiddify Tunnel Service",
		DisplayName: "Hiddify Tunnel Service",
		Description: "This is a bridge for tunnel",
	}

	prg := &hiddifyNext{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	if len(goArg) > 0 {
		if goArg == "uninstall" {
			err = s.Stop()
			if err != nil {
				log.Fatal(err)
			}
		}
		err = service.Control(s, goArg)
		if err != nil {
			log.Fatal(err)
		}
		if goArg == "install" {
			err = s.Start()
			if err != nil {
				log.Fatal(err)
			}
		}

		return
	}

	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
