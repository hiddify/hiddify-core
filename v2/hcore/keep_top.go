package hcore

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	sbox "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	"github.com/sagernet/sing-box/option"
	N "github.com/sagernet/sing/common/network"

	"github.com/hiddify/hiddify-core/v2/config"
)

// Keep chosen: with keep-tags set, the proxies of the profile the user chose run, and only them
// (and the ones they go through), as long as any of them is still in the profile.
//
// Keep top: with keep-top set and a profile with more proxies than that, the core starts
// with all of them, and once their first pings are in it keeps only the fastest (and the one the
// user selected) and stops the rest, by a hot reload the clients do not see. With keep-top-quick it
// does not wait for every first ping when the rest were slower last time (pickTopEarly): with a
// large subscription the pick comes after a few dozen pings. A hot reload keeps
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
	answering    bool              // keep-top answering: a kept proxy that stops answering is dropped
	countries    []string          // keep-top-countries the pick was for, sorted
	// keep-top, keep-tags and keep-top-countries of the run, and whether the reload under way is
	// keep top's own
	setting          int
	settingTags      []string
	settingCountries []string
	ownReload        bool
}

// keepTopSettings is the keep-top and keep-tags of the run: those of the settings the app sent,
// except in keep top's own reload, which keeps those the run started with. The app sends the
// settings of other profiles too, to check their configs; a reload with those ran every proxy.
func keepTopSettings() (int, []string, []string) {
	keepTop.Lock()
	defer keepTop.Unlock()
	if !keepTop.ownReload && static.HiddifyOptions != nil {
		keepTop.setting, keepTop.settingTags = static.HiddifyOptions.KeepTop, static.HiddifyOptions.KeepTags
		keepTop.settingCountries = normalizeCountries(static.HiddifyOptions.KeepTopCountries)
	}
	return keepTop.setting, keepTop.settingTags, keepTop.settingCountries
}

// normalizeCountries is codes upper-cased, sorted and without repeats or blanks; nil for none.
func normalizeCountries(codes []string) []string {
	var out []string
	for _, code := range codes {
		if code = strings.ToUpper(strings.TrimSpace(code)); len(code) == 2 && !slices.Contains(out, code) {
			out = append(out, code)
		}
	}
	slices.Sort(out)
	return out
}

// proxyCountry is where a proxy is: the country of its IP, as the monitoring looked it up, else
// the flag in its tag; "" when neither tells.
func proxyCountry(tag string, history *adapter.URLTestHistory) string {
	if history != nil && history.IpInfo != nil && len(history.IpInfo.CountryCode) == 2 {
		return strings.ToUpper(history.IpInfo.CountryCode)
	}
	runes := []rune(tag)
	regional := func(r rune) bool { return r >= 0x1F1E6 && r <= 0x1F1FF }
	for i := 0; i+1 < len(runes); i++ {
		if regional(runes[i]) && regional(runes[i+1]) {
			return string([]rune{runes[i] - 0x1F1E6 + 'A', runes[i+1] - 0x1F1E6 + 'A'})
		}
	}
	return ""
}

// outsideCountries reports whether a proxy is left out by keep-top-countries: its country is
// another, or it was tested and where it is is still not known.
func outsideCountries(countries []string, country string, tested bool) bool {
	if len(countries) == 0 {
		return false
	}
	if country == "" {
		return tested
	}
	return !slices.Contains(countries, country)
}

// buildRunningConfig is BuildConfig for the core to run, without the proxies keep top stopped;
// static.lock must be held.
// buildRunningConfig builds the config the core runs. A core that delegates the system proxy (see
// config.InboundOptions.DelegateSystemProxy) takes it out of the config here and keeps its address.
func buildRunningConfig(ctx context.Context, in *StartRequest) (*option.Options, error) {
	options, err := buildConfigToRun(ctx, in)
	if err == nil {
		recordDelegatedSystemProxy(options)
		markEngineLeftovers(options)
	}
	return options, err
}

func buildConfigToRun(ctx context.Context, in *StartRequest) (*option.Options, error) {
	count, chosen, countries := keepTopSettings()
	if in.EnableRawConfig { // a raw config keeps the tags the user wrote
		clearKeepTop()
		return BuildConfig(ctx, in)
	}
	input, err := config.ReadSingOptions(ctx, &config.ReadOptions{Content: in.ConfigContent, Path: in.ConfigPath})
	if err != nil {
		return nil, err
	}
	items := stableProfileTags(ctx, in.ConfigPath, input)
	if count <= 0 && count != config.KeepTopAnswering && len(countries) == 0 && len(chosen) == 0 {
		clearKeepTop()
		return config.BuildConfig(ctx, static.HiddifyOptions, &config.ReadOptions{Options: input})
	}
	total := len(config.UserProxyTags(input))
	answering := count == config.KeepTopAnswering
	// every proxy may be kept: the pick drops only the ones that do not answer, or of another country
	if answering || count <= 0 {
		count = total
	}
	filtered := answering || len(countries) > 0
	// the proxies the user chose run, and only them, while any of them is still in the profile
	if keep := config.ChosenProxies(input, chosen); len(keep) > 0 {
		clearKeepTop()
		stopped := config.KeepOnlyProxies(ctx, input, keep)
		Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep chosen: running %d of the %d proxies", total-stopped, total))
		return config.BuildConfig(ctx, static.HiddifyOptions, &config.ReadOptions{Options: input})
	}
	if count <= 0 || (total <= count && !filtered) {
		clearKeepTop()
		return config.BuildConfig(ctx, static.HiddifyOptions, &config.ReadOptions{Options: input})
	}
	fingerprints := config.Fingerprints(items) // the stable tags' own, so one pass tells both

	keepTop.Lock()
	keepTop.generation++
	keepTop.fingerprints = fingerprints
	keepTop.answering = answering
	// the same proxies stay while they are good; fewer of them, the fastest of those, with no new
	// pings; more of them, the whole list runs again to find them
	if keepTop.path == in.ConfigPath && keptStillGood(fingerprints) &&
		slices.Equal(keepTop.countries, countries) &&
		(keepTop.count == count || (count < keepTop.count && trimKeptLocked(count))) {
		keep := make(map[string]bool, len(keepTop.kept))
		for tag := range keepTop.kept {
			keep[tag] = true
		}
		stopped := config.KeepOnlyProxies(ctx, input, keep)
		keepTop.picking = false
		Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep top: running %d of the %d proxies", total-stopped, total))
	} else {
		if keepTop.path != in.ConfigPath || keepTop.count != count || !slices.Equal(keepTop.countries, countries) {
			keepTop.freshOnly = false
		}
		keepTop.countries = countries
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
	keepTop.picking, keepTop.freshOnly, keepTop.answering = false, false, false
}

// keptStillGood reports whether every kept proxy is still in the profile, unchanged, none of them
// timed out and one of them answers; keepTop must be locked. A kept proxy that died is replaced
// by running the whole list again, as at the first pick; the pings of the proxies that did not
// change stay, so only the others are tested.
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
	alive := false
	for tag := range keepTop.kept {
		delay := historyDelay(histories[tag])
		if pingTimedOut(delay) {
			return false
		}
		if pingAlive(delay) {
			alive = true
		}
	}
	return alive
}

// followKeepTop, run by the connectivity watcher, picks the fastest proxies once their first pings
// are in, and brings the whole list back once every kept one timed out.
func (h *HiddifyInstance) followKeepTop(state CoreStates) {
	keepTop.Lock()
	picking, kept, count, since, freshOnly, generation := keepTop.picking, keepTop.kept, keepTop.count, keepTop.since, keepTop.freshOnly, keepTop.generation
	answering, countries := keepTop.answering, keepTop.countries
	quick := static.HiddifyOptions != nil && static.HiddifyOptions.KeepTopQuick
	queued := keepTop.queued
	keepTop.Unlock()
	if queued || (!picking && kept == nil) {
		return
	}
	if !picking {
		if state == CoreStates_NO_AVAILABLE_PROXIES {
			Log(LogLevel_WARNING, LogType_CORE, "keep top: every kept proxy timed out, running them all again")
			queueKeepTop(generation, nil)
			return
		}
		// keep-top answering keeps only the ones that answer, also after the pick
		if answering {
			if ctx := h.Context(); ctx != nil {
				histories := monitoring.Get(ctx).OutboundsHistory("")
				delays := make(map[string]int32, len(kept))
				for tag := range kept {
					delays[tag] = historyDelay(histories[tag])
				}
				if pick := withoutTimedOut(delays); pick != nil {
					Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep top: %d of the %d kept proxies stopped answering, stopping them", len(kept)-len(pick), len(kept)))
					queueKeepTop(generation, pick)
				}
			}
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
	hints := make(map[string]int32)
	for _, tag := range group.All() {
		if !isProxyTag(tag) || isGroupTag(box, tag) {
			continue
		}
		history := histories[tag]
		hint := previousDelay(history)
		if freshOnly && history != nil && history.Time.Before(since) {
			history = nil
		}
		delay := historyDelay(history)
		if outsideCountries(countries, proxyCountry(tag, histories[tag]), delay != 0) {
			continue
		}
		hints[tag], delays[tag] = hint, delay
	}
	pick := pickTopProxies(delays, count, time.Since(since))
	how := "keeping"
	if pick == nil {
		if !quick {
			return
		}
		if pick = pickTopEarly(delays, hints, count); pick == nil {
			return
		}
		how = "the rest are slower by their last pings: keeping"
	}
	pick = withSelected(pick, count, selectedProxy(box))
	if len(countries) > 0 {
		how += " in " + strings.Join(countries, " ")
	}
	Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep top: %s %d of the %d proxies, stopping the others", how, len(pick), len(delays)))
	queueKeepTop(generation, pick)
}

// selectedProxy is the profile's proxy the user selected; "" for a group, or none.
func selectedProxy(box *sbox.Box) string {
	sel, ok := box.Outbound().Outbound(config.OutboundSelectTag)
	if !ok {
		return ""
	}
	selector, ok := sel.(adapter.OutboundGroup)
	if !ok {
		return ""
	}
	selected := selector.Selected(N.NetworkTCP)
	if selected == nil {
		return ""
	}
	if now := selected.Tag(); isProxyTag(now) && !isGroupTag(box, now) {
		return now
	}
	return ""
}

// withSelected is pick, fastest first, with the proxy the user selected in it: in place of the
// slowest when pick is full, so count proxies run, not one more.
func withSelected(pick []string, count int, selected string) []string {
	if selected == "" || slices.Contains(pick, selected) {
		return pick
	}
	if len(pick) >= count && count > 0 {
		pick = pick[:count-1]
	}
	return append(pick, selected)
}

// trimKeptLocked keeps the count fastest of the kept proxies by the pings they already have, with
// the one the user selected; keepTop must be locked, and keptStillGood true.
func trimKeptLocked(count int) bool {
	ctx := static.Context()
	if ctx == nil || static.StartedService == nil {
		return false
	}
	histories := monitoring.Get(ctx).OutboundsHistory("")
	tags := make([]string, 0, len(keepTop.kept))
	delays := make(map[string]int32, len(keepTop.kept))
	for tag := range keepTop.kept {
		tags = append(tags, tag)
		delays[tag] = historyDelay(histories[tag])
	}
	// the ones that answer first, the fastest of them first
	sort.Slice(tags, func(i, j int) bool {
		a, b := delays[tags[i]], delays[tags[j]]
		if pingAlive(a) != pingAlive(b) {
			return pingAlive(a)
		}
		if a != b {
			return a < b
		}
		return tags[i] < tags[j]
	})
	if len(tags) > count {
		tags = tags[:count]
	}
	selected := ""
	if instance := static.StartedService.Instance(); instance != nil && instance.Box() != nil {
		selected = selectedProxy(instance.Box())
		if _, kept := keepTop.kept[selected]; !kept {
			selected = ""
		}
	}
	kept := make(map[string]string, count)
	for _, tag := range withSelected(tags, count, selected) {
		kept[tag] = keepTop.kept[tag]
	}
	keepTop.kept, keepTop.count = kept, count
	return true
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

// previousDelay is what a proxy's ping was before this run tested it: its cached delay, or its
// last one when the kept proxies failed and only pings since count; previousFailed when that
// test failed, 0 when it has none of the last previousMaxAge (never tested, or too long ago).
func previousDelay(history *adapter.URLTestHistory) int32 {
	switch {
	case history == nil, time.Since(history.Time) > previousMaxAge:
		return 0
	case pingTimedOut(int32(history.Delay)):
		return previousFailed
	default:
		return int32(history.Delay)
	}
}

const (
	// previousFailed is previousDelay's mark of a proxy whose previous test failed.
	previousFailed = -1
	// previousMaxAge: a proxy not tested for longer is tested again before an early pick.
	previousMaxAge = 24 * time.Hour
)

// pickTopEarly picks the count fastest of this run before every proxy answered, once the others
// are not expected to beat them: count proxies answered, every proxy without a previous ping of
// the last previousMaxAge was tested, and the slowest of those count is as fast as the fastest previous ping still to be
// tested. The monitoring tests by previous ping, fastest first, with the untested ones between
// them, so with a large subscription this comes after a few dozen pings, not all of them. A
// proxy whose previous test failed does not hold the pick. delays are this run's pings
// (historyDelay), hints the previous ones (previousDelay); nil until then.
func pickTopEarly(delays, hints map[string]int32, count int) []string {
	if count <= 0 {
		return nil
	}
	var alive []string
	best := int32(maxPing)
	for tag, delay := range delays {
		switch {
		case pingAlive(delay):
			alive = append(alive, tag)
		case pingTimedOut(delay):
		case hints[tag] == 0:
			return nil // never tested: it might be the fastest
		case hints[tag] > 0:
			best = min(best, hints[tag])
		}
	}
	if len(alive) < count {
		return nil
	}
	sort.Slice(alive, func(i, j int) bool {
		if delays[alive[i]] != delays[alive[j]] {
			return delays[alive[i]] < delays[alive[j]]
		}
		return alive[i] < alive[j]
	})
	alive = alive[:count]
	if delays[alive[count-1]] > best {
		return nil
	}
	return alive
}

// withoutTimedOut is the kept proxies of delays (historyDelay) but the ones whose test of this
// run timed out, when there are such and one of the others answers; nil otherwise: all of them
// timing out brings the whole list back instead (followKeepTop).
func withoutTimedOut(delays map[string]int32) []string {
	var keep []string
	dropped, alive := false, false
	for tag, delay := range delays {
		switch {
		case pingTimedOut(delay):
			dropped = true
			continue
		case pingAlive(delay):
			alive = true
		}
		keep = append(keep, tag)
	}
	if !dropped || !alive {
		return nil
	}
	slices.Sort(keep)
	return keep
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
	keepTop.Lock()
	keepTop.ownReload = true
	keepTop.Unlock()
	_, err := reloadRunning(ctx, static.previousStartRequest)
	keepTop.Lock()
	keepTop.ownReload = false
	keepTop.Unlock()
	if err != nil {
		Log(LogLevel_WARNING, LogType_CORE, "keep top: ", err.Error())
		clearKeepTop()
	}
}
