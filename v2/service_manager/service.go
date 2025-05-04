package service_manager

import (
	"github.com/sagernet/sing-box/option"
)

type HService interface {
	Init() error
	Dispose() error

	OnMainServicePreStart(singconfig *option.Options) error
	OnMainServiceStart() error
	OnMainServiceClose() error
}
