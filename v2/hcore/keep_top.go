package hcore

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	"github.com/sagernet/sing-box/option"
	N "github.com/sagernet/sing/common/network"

	"github.com/hiddify/hiddify-core/v2/config"
)

// Keep top: with keep-top set and a profile with more proxies than that, the core starts
// with all of them, and once their first pings are in it keeps only the fastest (and the one the
// user selected) and stops the rest, by a hot reload the clients do not see. A hot reload keeps
// that choice while the kept proxies are still in the profile, unchanged, and one of them answers;
// otherwise the whole list runs again and the fastest are picked anew. When every kept proxy has
// timed out, the whole list comes back as well. The extra security and unblocker hops are not the
// profile's proxies: they always run.

const (
	// keepTopSettle: once enough proxies answered, how long the slow ones may still take
	keepTopSettle = 20 * time.Second
	// keepTopMaxWait: how long to wait for the first pings of every proxy, with one answering
	keepTopMaxWait = 60 * time.Second
)

var keepTop struct {
	sync.Mutex
	path  string // the profile picked from
	count int    // the keep-top picked for
	// fingerprints of the profile's proxies, by tag, when the config was built
	fingerprints map[string]string
	kept         map[string]string // tag → fingerprint; nil while the whole list runs
	picking      bool              // the whole list runs, and the fastest are to be picked
	since        time.Time         // when picking began
	freshOnly    bool              // only pings since then count: the earlier ones failed us
	generation   uint64            // a config built since a pick was decided makes it stale
	queued       bool              // a pick or a return to the whole list is waiting in the queue
}

// buildRunningConfig is BuildConfig for the core to run, without the proxies keep top stopped;
// static.lock must be held.
func buildRunningConfig(ctx context.Context, in *StartRequest) (*option.Options, error) {
	count := 0
	if static.HiddifyOptions != nil {
		count = static.HiddifyOptions.KeepTop
	}
	if in.EnableRawConfig || count <= 0 {
		clearKeepTop()
		return BuildConfig(ctx, in)
	}
	input, err := config.ReadSingOptions(ctx, &config.ReadOptions{Content: in.ConfigContent, Path: in.ConfigPath})
	if err != nil {
		return nil, err
	}
	total := len(config.UserProxyTags(input))
	if total <= count {
		clearKeepTop()
		return config.BuildConfig(ctx, static.HiddifyOptions, &config.ReadOptions{Options: input})
	}
	fingerprints := config.ProxyFingerprints(ctx, input)

	keepTop.Lock()
	keepTop.generation++
	keepTop.fingerprints = fingerprints
	if keepTop.path == in.ConfigPath && keepTop.count == count && keptStillGood(fingerprints) {
		keep := make(map[string]bool, len(keepTop.kept))
		for tag := range keepTop.kept {
			keep[tag] = true
		}
		stopped := config.KeepOnlyProxies(ctx, input, keep)
		keepTop.picking = false
		Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep top: running %d of the %d proxies", total-stopped, total))
	} else {
		if keepTop.path != in.ConfigPath || keepTop.count != count {
			keepTop.freshOnly = false
		}
		keepTop.path, keepTop.count, keepTop.kept = in.ConfigPath, count, nil
		keepTop.picking, keepTop.since = true, time.Now()
		Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep top: running all %d proxies until the fastest %d are known", total, count))
	}
	keepTop.Unlock()
	return config.BuildConfig(ctx, static.HiddifyOptions, &config.ReadOptions{Options: input})
}

func clearKeepTop() {
	keepTop.Lock()
	defer keepTop.Unlock()
	keepTop.generation++
	keepTop.path, keepTop.count, keepTop.fingerprints, keepTop.kept = "", 0, nil, nil
	keepTop.picking, keepTop.freshOnly = false, false
}

// keptStillGood reports whether every kept proxy is still in the profile, unchanged, and one of
// them answers; keepTop must be locked.
func keptStillGood(fingerprints map[string]string) bool {
	if len(keepTop.kept) == 0 {
		return false
	}
	for tag, fingerprint := range keepTop.kept {
		if fingerprints[tag] != fingerprint {
			return false
		}
	}
	ctx := static.Context()
	if ctx == nil {
		return false
	}
	histories := monitoring.Get(ctx).OutboundsHistory("")
	for tag := range keepTop.kept {
		if pingAlive(historyDelay(histories[tag])) {
			return true
		}
	}
	return false
}

// followKeepTop, run by the connectivity watcher, picks the fastest proxies once their first pings
// are in, and brings the whole list back once every kept one timed out.
func (h *HiddifyInstance) followKeepTop(state CoreStates) {
	keepTop.Lock()
	picking, kept, count, since, freshOnly, generation := keepTop.picking, keepTop.kept, keepTop.count, keepTop.since, keepTop.freshOnly, keepTop.generation
	queued := keepTop.queued
	keepTop.Unlock()
	if queued || (!picking && kept == nil) {
		return
	}
	if !picking {
		if state == CoreStates_NO_AVAILABLE_PROXIES {
			Log(LogLevel_WARNING, LogType_CORE, "keep top: every kept proxy timed out, running them all again")
			queueKeepTop(generation, nil)
		}
		return
	}
	ctx, box := h.Context(), h.Box()
	if ctx == nil || box == nil {
		return
	}
	lowest, ok := box.Outbound().Outbound(config.OutboundURLTestTag)
	if !ok {
		return
	}
	group, isGroup := lowest.(adapter.OutboundGroup)
	if !isGroup {
		return
	}
	histories := monitoring.Get(ctx).OutboundsHistory("")
	delays := make(map[string]int32)
	for _, tag := range group.All() {
		if !isProxyTag(tag) || isGroupTag(box, tag) {
			continue
		}
		history := histories[tag]
		if freshOnly && history != nil && history.Time.Before(since) {
			history = nil
		}
		delays[tag] = historyDelay(history)
	}
	pick := pickTopProxies(delays, count, time.Since(since))
	if pick == nil {
		return
	}
	// the proxy the user chose stays as well
	if sel, ok := box.Outbound().Outbound(config.OutboundSelectTag); ok {
		if selector, ok := sel.(adapter.OutboundGroup); ok {
			if selected := selector.Selected(N.NetworkTCP); selected != nil {
				if now := selected.Tag(); isProxyTag(now) && !isGroupTag(box, now) && !slices.Contains(pick, now) {
					pick = append(pick, now)
				}
			}
		}
	}
	Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep top: keeping %d of the %d proxies, stopping the others", len(pick), len(delays)))
	queueKeepTop(generation, pick)
}

// pickTopProxies is the count fastest proxies by delays (see historyDelay), once enough first pings
// are in: all of them, or count after keepTopSettle, or any after keepTopMaxWait. Nil until then.
func pickTopProxies(delays map[string]int32, count int, waited time.Duration) []string {
	var alive []string
	pending := 0
	for tag, delay := range delays {
		switch {
		case pingAlive(delay):
			alive = append(alive, tag)
		case !pingTimedOut(delay):
			pending++
		}
	}
	ready := len(alive) > 0 && (pending == 0 ||
		(len(alive) >= count && waited >= keepTopSettle) ||
		waited >= keepTopMaxWait)
	if !ready {
		return nil
	}
	sort.Slice(alive, func(i, j int) bool {
		if delays[alive[i]] != delays[alive[j]] {
			return delays[alive[i]] < delays[alive[j]]
		}
		return alive[i] < alive[j]
	})
	if len(alive) > count {
		alive = alive[:count]
	}
	return alive
}

// queueKeepTop applies a pick (nil: the whole list again) in the operation queue, unless a config
// was built since generation.
func queueKeepTop(generation uint64, pick []string) {
	keepTop.Lock()
	keepTop.queued = true
	keepTop.Unlock()

	queue.Lock()
	defer queue.Unlock()
	// whether the core ends up started is what the operations before it decide
	starting := startingLocked()
	ctx := withStopSignalLocked(static.BaseContext)
	enqueueLocked(operationStart, "keep top", func() (*CoreInfoResponse, error) {
		applyKeepTopLocked(ctx, generation, pick)
		return nil, nil
	})
	if !starting {
		queue.lastKind = operationStop
	}
}

// applyKeepTopLocked is queueKeepTop's operation; static.lock is held.
func applyKeepTopLocked(ctx context.Context, generation uint64, pick []string) {
	keepTop.Lock()
	keepTop.queued = false
	stale := keepTop.generation != generation
	if !stale {
		if pick == nil {
			keepTop.kept, keepTop.freshOnly = nil, true
		} else {
			keepTop.kept = make(map[string]string, len(pick))
			for _, tag := range pick {
				keepTop.kept[tag] = keepTop.fingerprints[tag]
			}
		}
	}
	keepTop.Unlock()
	if stale || !slices.Contains(running, currentCoreState()) || static.StartedService == nil || static.previousStartRequest == nil {
		return
	}
	if err := contextErr(ctx); err != nil {
		return
	}
	if _, err := reloadRunning(ctx, static.previousStartRequest); err != nil {
		Log(LogLevel_WARNING, LogType_CORE, "keep top: ", err.Error())
		clearKeepTop()
	}
}
