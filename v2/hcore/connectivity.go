package hcore

import (
	"context"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"

	"github.com/hiddify/hiddify-core/v2/config"
)

// The core owns the connection state. Once the service is up the core is WAITING until a ping
// decides: STARTED with a ping, NO_AVAILABLE_PROXIES without one (see decideConnectivity). A ping
// that arrives moves it to STARTED, and one that goes away to NO_AVAILABLE_PROXIES. Clients only
// display what the core reports.

// connectivityRecheck is how often the ping is looked at when the monitor reports no change.
const connectivityRecheck = 2 * time.Second

// running are the states of a core that is up.
var running = []CoreStates{CoreStates_WAITING, CoreStates_STARTED, CoreStates_NO_AVAILABLE_PROXIES}

var watcher struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// startConnectivityWatcher follows the ping until stopConnectivityWatcher; static.lock must be held.
func startConnectivityWatcher() {
	stopConnectivityWatcher()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	watcher.cancel, watcher.done = cancel, done
	go func() {
		defer close(done)
		static.watchConnectivity(ctx)
	}()
}

// stopConnectivityWatcher ends the watcher and waits for it, so it cannot change the state of a
// core that is stopping; static.lock must be held.
func stopConnectivityWatcher() {
	if watcher.cancel == nil {
		return
	}
	watcher.cancel()
	<-watcher.done
	watcher.cancel, watcher.done = nil, nil
}

func (h *HiddifyInstance) watchConnectivity(ctx context.Context) {
	defer config.DeferPanicToError("connectivity", func(err error) {
		Log(LogLevel_ERROR, LogType_CORE, "connectivity watcher: ", err.Error())
	})
	var changes <-chan monitoring.GroupEvent
	if serviceCtx := h.Context(); serviceCtx != nil {
		monitor := monitoring.Get(serviceCtx)
		if sub, err := monitor.SubscribeGroup(""); err == nil {
			changes = sub
			defer monitor.UnsubscribeGroup("", sub)
		}
	}
	ticker := time.NewTicker(connectivityRecheck)
	defer ticker.Stop()
	for {
		h.updateConnectivity(ctx)
		select {
		case <-ctx.Done():
			return
		case <-changes:
		case <-ticker.C:
		}
	}
}

// updateConnectivity moves the running core between WAITING, STARTED and NO_AVAILABLE_PROXIES by
// what the monitoring knows of the proxies.
func (h *HiddifyInstance) updateConnectivity(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	state := probeConnectivity(h)
	setCoreStatusIf(running, state, MessageType_EMPTY, "")
}

// connectedState is the state of a running core by its proxies.
func (h *HiddifyInstance) connectedState() CoreStates {
	return probeConnectivity(h)
}

// probeConnectivity is replaced in tests, which have no outbound to ping.
var probeConnectivity = (*HiddifyInstance).proxiesConnectivity

// maxPing is the first delay that is no ping: a real delay is 0 < delay < maxPing, a test that
// failed is timeoutPing.
const (
	maxPing     = 65000
	timeoutPing = int32(monitoring.TimeoutDelay)
)

func pingAlive(delay int32) bool    { return delay > 0 && delay < maxPing }
func pingTimedOut(delay int32) bool { return delay >= maxPing }

// proxiesConnectivity reads the ping of what the user is connected through, from the monitoring.
func (h *HiddifyInstance) proxiesConnectivity() CoreStates {
	ctx, box := h.Context(), h.Box()
	if ctx == nil || box == nil || h.UrlTestHistory() == nil {
		return CoreStates_WAITING
	}
	histories := monitoring.Get(ctx).OutboundsHistory("")
	selected := h.selectedOutbound(histories)
	if selected == nil {
		return CoreStates_WAITING
	}
	var leaves []int32
	for tag, history := range histories {
		if !isProxyTag(tag) {
			continue
		}
		if out, ok := box.Outbound().Outbound(tag); ok {
			if _, isGroup := out.(adapter.OutboundGroup); isGroup {
				continue
			}
		}
		delay := int32(0)
		if history != nil && !history.IsFromCache {
			delay = int32(history.Delay)
		}
		leaves = append(leaves, delay)
	}
	return decideConnectivity(selected.UrlTestDelay, selected.IsGroup, leaves)
}

// isProxyTag reports whether the outbound is one the user's configs provide, not a built-in that
// goes straight to the network: those always answer and say nothing about the connection.
func isProxyTag(tag string) bool {
	switch tag {
	case config.OutboundDirectTag, config.OutboundBypassTag, config.OutboundDirectFragmentTag:
		return false
	}
	return true
}

// selectedOutbound is the entry of the main group the user is connected through: the selected
// outbound, or the extra security hop in front of it.
func (h *HiddifyInstance) selectedOutbound(histories map[string]*adapter.URLTestHistory) *OutboundInfo {
	groups := h.GetAllProxiesInfo(histories, true)
	if groups == nil {
		return nil
	}
	for _, group := range groups.Items {
		if group.Tag != config.OutboundSelectTag {
			continue
		}
		for _, item := range group.Items {
			if item.Tag == group.Selected {
				return item
			}
		}
	}
	return nil
}

// decideConnectivity is the state of a running core by the last url tests. A delay of 0 is a test
// still to come (or a result from the cache), a delay of maxPing or more is a timeout.
//
//   - the selected outbound has a ping: STARTED, whether it is a proxy or a balancer.
//   - a proxy the user selected has no ping: NO_AVAILABLE_PROXIES once its test timed out.
//   - a balancer has no ping: NO_AVAILABLE_PROXIES once every proxy was tested and timed out;
//     WAITING while any is still to be tested, or a proxy answered that the balancer has yet to use.
//
// leaves are the delays of the proxies, not of the groups.
func decideConnectivity(selectedDelay int32, selectedIsGroup bool, leaves []int32) CoreStates {
	if pingAlive(selectedDelay) {
		return CoreStates_STARTED
	}
	if !selectedIsGroup {
		if pingTimedOut(selectedDelay) {
			return CoreStates_NO_AVAILABLE_PROXIES
		}
		return CoreStates_WAITING
	}
	if len(leaves) == 0 {
		return CoreStates_WAITING
	}
	for _, delay := range leaves {
		if !pingTimedOut(delay) {
			return CoreStates_WAITING
		}
	}
	return CoreStates_NO_AVAILABLE_PROXIES
}
