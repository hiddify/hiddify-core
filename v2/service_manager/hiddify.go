package service_manager

import (
	"github.com/sagernet/sing-box/adapter"
)

var (
	services    = []adapter.Service{}
	preservices = []adapter.Service{}
)

func RegisterPreservice(service adapter.Service) {
	preservices = append(services, service)
}

func Register(service adapter.Service) {
	services = append(services, service)
}

func StartServices() error {
	CloseServices()
	for _, service := range preservices {
		if err := service.Start(); err != nil {
			return err
		}
	}
	for _, service := range services {
		if err := service.Start(); err != nil {
			return err
		}
	}
	return nil
}

func CloseServices() error {
	for _, service := range services {
		if err := service.Close(); err != nil {
			return err
		}
	}
	for _, service := range preservices {
		if err := service.Close(); err != nil {
			return err
		}
	}
	return nil
}
