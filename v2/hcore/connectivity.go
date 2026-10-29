package hcore

import (
	"context"
	"time"

	sbox "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"

	"github.com/hiddify/hiddify-core/v2/config"
)

// The core owns the connection state. Once the service is up the core is WAITING until the
// "lowest" (auto) group's pings decide: STARTED with a working ping, NO_AVAILABLE_PROXIES when
// every proxy in that group has timed out (see decideConnectivity). When the profile has only
// one proxy, "lowest" is not built — then the select group's leaf proxy is used instead.
// The user's selected outbound does not drive this when "lowest" exists: picking a single dead
// proxy must not flip the core to NO_AVAILABLE_PROXIES. With extra security the traffic leaves
// through its hop, so the core is STARTED only once the hop has a ping too (see withHop).
// Clients only display what the core reports.

// connectivityRecheck is how often the ping is looked at when the monitor reports no change.
const connectivityRecheck = 2 * time.Second

// connectivitySettle is the least time between two looks caused by monitor events. A test cycle
// reports a result for every proxy within moments, and looking at each one cost the memory limited
// iOS extension too much.
const connectivitySettle = 500 * time.Millisecond

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
			// let the rest of the burst arrive; those that do are folded into one look
			select {
			case <-ctx.Done():
				return
			case <-time.After(connectivitySettle):
			}
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

// proxiesConnectivity is the state by the proxies' pings, and with extra security by its hop's too.
func (h *HiddifyInstance) proxiesConnectivity() CoreStates {
	ctx, box := h.Context(), h.Box()
	if ctx == nil || box == nil || h.UrlTestHistory() == nil {
		return CoreStates_WAITING
	}
	histories := monitoring.Get(ctx).OutboundsHistory("")
	state := serversConnectivity(box, histories)
	if hop := chainHop(box); config.IsChainExtraSecurityTag(hop) {
		return withHop(state, historyDelay(histories[hop]))
	}
	return state
}

// withHop is the state with extra security: its hop dials through the proxies and carries all the
// traffic, so the proxies' STARTED waits until the hop has a ping too.
func withHop(servers CoreStates, hopDelay int32) CoreStates {
	if servers == CoreStates_STARTED && !pingAlive(hopDelay) {
		return CoreStates_WAITING
	}
	return servers
}

// serversConnectivity reads pings of the built-in "lowest" group from monitoring. It runs often,
// so it only walks that group's members and their histories — not the user's selected outbound.
// With a single-proxy profile there is no "lowest"; the select leaf is used instead.
func serversConnectivity(box *sbox.Box, histories map[string]*adapter.URLTestHistory) CoreStates {
	lowest, ok := box.Outbound().Outbound(config.OutboundURLTestTag)
	if !ok {
		return decideConnectivity(0, true, selectProxyLeaves(box, histories))
	}
	group, isGroup := lowest.(adapter.OutboundGroup)
	if !isGroup {
		return decideConnectivity(historyDelay(histories[config.OutboundURLTestTag]), false, nil)
	}
	var leaves []int32
	for _, tag := range group.All() {
		if !isProxyTag(tag) || isGroupTag(box, tag) {
			continue
		}
		leaves = append(leaves, historyDelay(histories[tag]))
	}
	return decideConnectivity(historyDelay(histories[config.OutboundURLTestTag]), true, leaves)
}

// selectProxyLeaves are delays of non-group proxy members of "select" (the one proxy when
// "lowest" was not built).
func selectProxyLeaves(box *sbox.Box, histories map[string]*adapter.URLTestHistory) []int32 {
	sel, ok := box.Outbound().Outbound(config.OutboundSelectTag)
	if !ok {
		return nil
	}
	group, isGroup := sel.(adapter.OutboundGroup)
	if !isGroup {
		if isProxyTag(sel.Tag()) {
			return []int32{historyDelay(histories[sel.Tag()])}
		}
		return nil
	}
	var leaves []int32
	for _, tag := range group.All() {
		if !isProxyTag(tag) || isGroupTag(box, tag) {
			continue
		}
		leaves = append(leaves, historyDelay(histories[tag]))
	}
	return leaves
}

// historyDelay is the delay of a url test, or 0 when there is none or it came from the cache.
func historyDelay(history *adapter.URLTestHistory) int32 {
	if history == nil || history.IsFromCache {
		return 0
	}
	return int32(history.Delay)
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

func isGroupTag(box *sbox.Box, tag string) bool {
	if out, ok := box.Outbound().Outbound(tag); ok {
		_, isGroup := out.(adapter.OutboundGroup)
		return isGroup
	}
	return false
}

// decideConnectivity is the state of a running core by the last url tests of the "lowest" group.
// A delay of 0 is a test still to come (or a result from the cache), a delay of maxPing or more is
// a timeout.
//
//   - the lowest group or any of its leaf proxies has a ping: STARTED.
//   - every proxy in the lowest group timed out: NO_AVAILABLE_PROXIES.
//   - otherwise WAITING (tests still running, or no members yet).
//
// groupDelay is the aggregated delay of the lowest group; leaves are delays of its member proxies
// (or of select when lowest is absent for a one-proxy profile).
func decideConnectivity(groupDelay int32, isGroup bool, leaves []int32) CoreStates {
	if pingAlive(groupDelay) {
		return CoreStates_STARTED
	}
	if !isGroup {
		if pingTimedOut(groupDelay) {
			return CoreStates_NO_AVAILABLE_PROXIES
		}
		return CoreStates_WAITING
	}
	if len(leaves) == 0 {
		return CoreStates_WAITING
	}
	pending := false
	for _, delay := range leaves {
		if pingAlive(delay) {
			return CoreStates_STARTED
		}
		if !pingTimedOut(delay) {
			pending = true
		}
	}
	if pending {
		return CoreStates_WAITING
	}
	return CoreStates_NO_AVAILABLE_PROXIES
}
