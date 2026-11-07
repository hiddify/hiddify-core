package hcore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
var leftovers struct {
	access sync.Mutex
	// set by the running config's set_system_proxy inbound
	engineProxy string
	// set by EnableUserSystemProxy, for a root core's engine
	userProxy string
	// the running config's TUNs that change the routing (Linux)
	tuns []tunLeftover
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

// markEngineLeftovers records what the config about to run changes in the host; nil when the
// engine stops (its inbounds undid it).
func markEngineLeftovers(options *option.Options) {
	address := ""
	if target := config.SystemProxyOf(options); target != nil {
		address = target.Address
	}
	tuns := routingTuns(options)
	leftovers.access.Lock()
	defer leftovers.access.Unlock()
	leftovers.engineProxy = address
	leftovers.tuns = tuns
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
	for _, address := range []string{leftovers.engineProxy, leftovers.userProxy} {
		if address != "" {
			addresses = append(addresses, address)
		}
	}
	var tunContent []byte
	if len(leftovers.tuns) > 0 {
		tunContent, _ = json.Marshal(leftovers.tuns)
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
