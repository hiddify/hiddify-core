package hcore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/hiddify/hiddify-core/v2/config"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	tun "github.com/sagernet/sing-tun"
)

// A core that is killed (SIGKILL, a crash, a power cut) cannot undo what it changed in the host:
//   - the system proxy it set keeps pointing at a closed port;
//   - on Linux, the policy rules and the nftables table of its TUN (auto_route, auto_redirect) keep
//     sending the host's traffic to a TUN and a listener that are gone.
//
// Either way the host has no network until the core starts again. So while such a change is on, a
// marker in the core's working dir records it, and a core that starts in the same dir and finds a
// marker undoes it. The app starts a new core as soon as its core dies, and again when it is
// launched; systemd and Windows restart the service.
//
// The marker is written before an engine applies a config and cleared only once the engine is
// closed: a core killed in between leaves it, and what it names may or may not be set, which the
// cleanup handles. While a hot reload applies a config, the running one's changes stay marked too.
var leftovers struct {
	access sync.Mutex
	// what the running engine changed, and what the config being applied will change
	running, applying engineLeftovers
	// set by EnableUserSystemProxy, for a root core's engine
	userProxy string
}

// engineLeftovers is what an engine's config changes in the host.
type engineLeftovers struct {
	// set by the config's set_system_proxy inbound
	proxy string
	// the config's TUNs that change the routing (Linux)
	tuns []tunLeftover
}

func engineLeftoversOf(options *option.Options) engineLeftovers {
	var it engineLeftovers
	if target := config.SystemProxyOf(options); target != nil {
		it.proxy = target.Address
	}
	it.tuns = routingTuns(options)
	return it
}

// tunLeftover names what a TUN set up in the host (see tun.CleanupLeftovers).
type tunLeftover struct {
	RuleIndex         int    `json:"rule_index,omitempty"`
	FallbackRuleIndex int    `json:"fallback_rule_index,omitempty"`
	AutoRedirect      bool   `json:"auto_redirect,omitempty"`
	TProxyMark        uint32 `json:"tproxy_mark,omitempty"`
}

const autoRedirectTableName = "sing-box"

func leftoverPath(workingDir string, name string) string {
	if workingDir == "" {
		return ""
	}
	return filepath.Join(workingDir, "data", name)
}

// engineApplying marks what options change in the host, before the engine applies them (a start,
// or a hot reload of the running engine).
func engineApplying(options *option.Options) {
	leftovers.access.Lock()
	defer leftovers.access.Unlock()
	leftovers.applying = engineLeftoversOf(options)
	writeLeftoversLocked()
}

// engineApplied ends what engineApplying began: applied, the config's changes are the running
// engine's; not applied (a refused hot reload), the running engine keeps its own.
func engineApplied(applied bool) {
	leftovers.access.Lock()
	defer leftovers.access.Unlock()
	if applied {
		leftovers.running = leftovers.applying
	}
	leftovers.applying = engineLeftovers{}
	writeLeftoversLocked()
}

// engineClosed clears the engine's marker once the engine is closed: its inbounds undid what
// they changed. A close that failed keeps it, for the cleanup of the next core.
func engineClosed() {
	leftovers.access.Lock()
	defer leftovers.access.Unlock()
	leftovers.running, leftovers.applying = engineLeftovers{}, engineLeftovers{}
	writeLeftoversLocked()
}

func markUserSystemProxy(address string) {
	leftovers.access.Lock()
	defer leftovers.access.Unlock()
	leftovers.userProxy = address
	writeLeftoversLocked()
}

func routingTuns(options *option.Options) []tunLeftover {
	if runtime.GOOS != "linux" || options == nil {
		return nil
	}
	var tuns []tunLeftover
	for _, inbound := range options.Inbounds {
		tunOptions, ok := inbound.Options.(*option.TunInboundOptions)
		if inbound.Type != C.TypeTun || !ok || tunOptions == nil || !tunOptions.AutoRoute {
			continue
		}
		tuns = append(tuns, tunLeftover{
			RuleIndex:         tunOptions.IPRoute2RuleIndex,
			FallbackRuleIndex: tunOptions.AutoRedirectFallbackRuleIndex,
			AutoRedirect:      tunOptions.AutoRedirect,
			TProxyMark:        uint32(tunOptions.AutoRedirectTProxyMark),
		})
	}
	return tuns
}

func writeLeftoversLocked() {
	var addresses []string
	for _, address := range []string{leftovers.running.proxy, leftovers.applying.proxy, leftovers.userProxy} {
		if address != "" && !slices.Contains(addresses, address) {
			addresses = append(addresses, address)
		}
	}
	var tuns []tunLeftover
	for _, it := range append(slices.Clone(leftovers.running.tuns), leftovers.applying.tuns...) {
		if !slices.Contains(tuns, it) {
			tuns = append(tuns, it)
		}
	}
	var tunContent []byte
	if len(tuns) > 0 {
		tunContent, _ = json.Marshal(tuns)
	}
	writeMarker(leftoverPath(sWorkingPath, "system-proxy.on"), []byte(strings.Join(addresses, "\n")))
	writeMarker(leftoverPath(sWorkingPath, "tun.on"), tunContent)
}

func writeMarker(path string, content []byte) {
	if path == "" {
		return
	}
	if len(content) == 0 {
		_ = os.Remove(path)
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, content, 0o644)
}

// cleanupLeftovers undoes what a core killed earlier in this core's working dir left in the host.
func cleanupLeftovers() {
	CleanupLeftovers(sWorkingPath, func(level LogLevel, message string) {
		Log(level, LogType_CORE, message)
	})
}

// CleanupLeftovers undoes what a core killed earlier in workingDir left in the host. A service
// calls it as it starts, before any core runs: its engine starts only when the app connects.
func CleanupLeftovers(workingDir string, logf func(level LogLevel, message string)) {
	if path := leftoverPath(workingDir, "tun.on"); path != "" {
		if content, err := os.ReadFile(path); err == nil {
			var tuns []tunLeftover
			if err := json.Unmarshal(content, &tuns); err != nil {
				logf(LogLevel_WARNING, "reading the TUN a stopped core left: "+err.Error())
			}
			for _, it := range tuns {
				tun.CleanupLeftovers(tun.Options{
					IPRoute2RuleIndex:                     it.RuleIndex,
					IPRoute2AutoRedirectFallbackRuleIndex: it.FallbackRuleIndex,
					AutoRedirectMarkMode:                  it.AutoRedirect,
					AutoRedirectTProxyMark:                it.TProxyMark,
				}, autoRedirectTableName)
				logf(LogLevel_INFO, "removed the routing a stopped core's TUN left")
			}
			_ = os.Remove(path)
		}
	}
	if path := leftoverPath(workingDir, "system-proxy.on"); path != "" {
		if content, err := os.ReadFile(path); err == nil {
			for _, address := range strings.Fields(string(content)) {
				if err := restoreSystemProxy(address); err != nil {
					logf(LogLevel_WARNING, "restoring the system proxy a stopped core left at "+address+": "+err.Error())
				} else {
					logf(LogLevel_INFO, "restored the system proxy a stopped core left at "+address)
				}
			}
			_ = os.Remove(path)
		}
	}
}
