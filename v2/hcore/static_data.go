package hcore

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/common/monitoring"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

type HiddifyInstance struct {
	// startedService is the running service. Start and stop set it under lock; everything else
	// (streams, commands, the connectivity watcher) reads it at any time, so it is atomic, and a
	// reader takes it once (Instance) instead of reading it again for the box and the context.
	startedService atomic.Pointer[daemon.StartedService]
	HiddifyOptions *config.HiddifyOptions
	// activeConfigPath string
	CoreLogFactory            log.Factory
	coreInfoObserver          *monitoring.Broadcaster[*CoreInfoResponse]
	CoreState                 CoreStates
	logObserver               *monitoring.Broadcaster[*LogMessage]
	systemInfoObserver        *monitoring.Broadcaster[*SystemInfo]
	outboundsInfoObserver     *monitoring.Broadcaster[*OutboundGroupList]
	mainOutboundsInfoObserver *monitoring.Broadcaster[*OutboundGroupList]
	lock                      sync.Mutex
	globalPlatformInterface   libbox.PlatformInterface
	previousStartRequest      *StartRequest
	debug                     bool
	setupDebug                bool // debug as the platform set the core up; debug adds a verbose log level, which can end
	ListenPort                uint16
	BaseContext               context.Context
	endPauseTimer             *time.Timer // only for ios

	// read by every log call from any goroutine
	logLevel atomic.Int32
}

var static = &HiddifyInstance{
	CoreState:                 CoreStates_STOPPED,
	coreInfoObserver:          monitoring.NewBroadcaster[*CoreInfoResponse](context.Background()),
	logObserver:               monitoring.NewBroadcaster[*LogMessage](context.Background()),
	systemInfoObserver:        monitoring.NewBroadcaster[*SystemInfo](context.Background()),
	outboundsInfoObserver:     monitoring.NewBroadcaster[*OutboundGroupList](context.Background()),
	mainOutboundsInfoObserver: monitoring.NewBroadcaster[*OutboundGroupList](context.Background()),
}
