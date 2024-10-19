package service_manager

import (
	"github.com/sagernet/sing-box/option"
)

var (
	services    = []HService{}
	preservices = []HService{}
)

func RegisterPreService(service HService) {
	preservices = append(services, service)
}

func Register(service HService) {
	services = append(services, service)
}

func StartServices() error {
	DisposeServices()
	for _, service := range preservices {
		if err := service.Init(); err != nil {
			return err
		}
	}
	for _, service := range services {
		if err := service.Init(); err != nil {
			return err
		}
	}
	return nil
}

func DisposeServices() error {
	for _, service := range services {
		if err := service.Dispose(); err != nil {
			return err
		}
	}
	for _, service := range preservices {
		if err := service.Dispose(); err != nil {
			return err
		}
	}
	return nil
}

func OnMainServicePreStart(singconfig *option.Options) error {
	for _, service := range preservices {
		if err := service.OnMainServicePreStart(singconfig); err != nil {
			return err
		}
	}
	for _, service := range services {
		if err := service.OnMainServicePreStart(singconfig); err != nil {
			return err
		}
	}
	return nil
}

func OnMainServiceStart() error {
	for _, service := range preservices {
		if err := service.OnMainServiceStart(); err != nil {
			return err
		}
	}
	for _, service := range services {
		if err := service.OnMainServiceStart(); err != nil {
			return err
		}
	}
	return nil
}

func OnMainServiceClose() error {
	for _, service := range preservices {
		if err := service.OnMainServiceClose(); err != nil {
			return err
		}
	}
	for _, service := range services {
		if err := service.OnMainServiceClose(); err != nil {
			return err
		}
	}
	return nil
}
