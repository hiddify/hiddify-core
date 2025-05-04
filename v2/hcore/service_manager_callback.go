package hcore

import (
	"github.com/hiddify/hiddify-core/v2/service_manager"
	"github.com/sagernet/sing-box/adapter"
)

type hiddifyMainServiceManager struct{}

var _ adapter.Service = (*hiddifyMainServiceManager)(nil)

func (h *hiddifyMainServiceManager) Start() error {
	return service_manager.OnMainServiceStart()
}

func (h *hiddifyMainServiceManager) Close() error {
	return service_manager.OnMainServiceClose()
}
