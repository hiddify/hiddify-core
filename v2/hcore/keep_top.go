package hcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/sagernet/sing-box/common/monitoring"
	"github.com/sagernet/sing-box/option"

	"github.com/hiddify/hiddify-core/v2/config"
)

// Keep chosen: with keep-tags set, the proxies of the profile the user chose run, and only them
// (and the ones they go through), as long as any of them is still in the profile.
//
// Keep top: with keep-top set and a profile with more proxies than that, the core uses only the
// fastest of them, and the one the user selected. The core does all of it by itself (sing-box's
// keep top, see config.ApplyKeepTop and the monitoring's h_keep_top.go): the config keeps every
// proxy, as lazy outbounds that exist only while something goes through them, and the monitoring
// goes through them a few at a time and leaves all but the fastest at rest. Nothing is reloaded
// for it. Here is only what turns it on, and what the app is told while it looks.
//
// Keep top manages the proxies that can be lazy (config.LazyProxyTags). The endpoints (WireGuard,
// WARP), the outbounds that search before they work (Psiphon, the DNS tunnels) and the extra
// security and unblocker hops run for the whole run, as without keep top.

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
	hopt := static.HiddifyOptions
	if hopt == nil {
		return nil, errors.New("HiddifyOptions not initialized")
	}
	if in.EnableRawConfig { // a raw config keeps the tags the user wrote
		return BuildConfig(ctx, in)
	}
	input, err := config.ReadSingOptions(ctx, &config.ReadOptions{Content: in.ConfigContent, Path: in.ConfigPath})
	if err != nil {
		return nil, err
	}
	stableProfileTags(ctx, in.ConfigPath, input)
	// the proxies the user chose run, and only them, while any of them is still in the profile
	if keep := config.ChosenProxies(input, hopt.KeepTags); len(keep) > 0 {
		total := len(config.UserProxyTags(input))
		stopped := config.KeepOnlyProxies(ctx, input, keep)
		Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep chosen: running %d of the %d proxies", total-stopped, total))
		return config.BuildConfig(ctx, hopt, &config.ReadOptions{Options: input})
	}
	candidates := config.LazyProxyTags(input)
	options, err := config.BuildConfig(ctx, hopt, &config.ReadOptions{Options: input})
	if err != nil {
		return nil, err
	}
	if managed := config.ApplyKeepTop(options, hopt, candidates); managed > 0 {
		Log(LogLevel_INFO, LogType_CORE, fmt.Sprintf("keep top: the core finds the fastest of %d proxies and leaves the others at rest", managed))
	}
	return options, nil
}

// keepTopConnectivity is the state of the core by its proxies, with keep top: while its first
// pick still has proxies to test, the ones in use all timing out does not mean none is available.
func keepTopConnectivity(h *HiddifyInstance, state CoreStates) CoreStates {
	if state != CoreStates_NO_AVAILABLE_PROXIES {
		return state
	}
	if ctx := h.Context(); ctx != nil {
		if monitor := monitoring.Get(ctx); monitor != nil && monitor.KeepTopSearching() {
			return CoreStates_WAITING
		}
	}
	return state
}
