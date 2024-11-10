package hcore

import (
	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/observable"
)

type HiddifyInstance struct {
	Box            *libbox.BoxService
	HiddifyOptions *config.HiddifyOptions
	// activeConfigPath string
	CoreLogFactory            log.Factory
	coreInfoObserver          *observable.Observer[*CoreInfoResponse]
	CoreState                 CoreStates
	logObserver               *observable.Observer[*LogMessage]
	systemInfoObserver        *observable.Observer[*SystemInfo]
	outboundsInfoObserver     *observable.Observer[*OutboundGroupList]
	mainOutboundsInfoObserver *observable.Observer[*OutboundGroupList]
}

var static = &HiddifyInstance{
	coreInfoObserver:          NewObserver[*CoreInfoResponse](1),
	CoreState:                 CoreStates_STOPPED,
	logObserver:               NewObserver[*LogMessage](1),
	systemInfoObserver:        NewObserver[*SystemInfo](1),
	outboundsInfoObserver:     NewObserver[*OutboundGroupList](1),
	mainOutboundsInfoObserver: NewObserver[*OutboundGroupList](1),
}
