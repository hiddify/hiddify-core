//go:build linux && !android

package hcore

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// routerHost holds the kernel settings router mode changed, to put back when it stops.
var routerHost struct {
	sync.Mutex
	saved map[string]string
}

// prepareRouterHost readies a Linux system for router mode: it loads the nfqueue modules that
// auto_redirect needs where they are modules, and turns on IP forwarding, without which a router
// forwards nothing (OpenWrt has it on; a Linux box made a gateway often has it off). For IPv6,
// interfaces that took router advertisements keep taking them (accept_ra 2), as forwarding would
// end that and drop their IPv6 default route. What it changed goes back at restoreRouterHost.
func prepareRouterHost() {
	for _, module := range []string{"nfnetlink_queue", "nft_queue"} {
		// best effort: built in, already loaded, or no modprobe (a container) all leave it as it is
		_ = exec.Command("modprobe", "-q", module).Run()
	}
	routerHost.Lock()
	defer routerHost.Unlock()
	if routerHost.saved == nil {
		routerHost.saved = map[string]string{}
	}
	setSysctl("/proc/sys/net/ipv4/ip_forward", "1")
	if _, err := os.Stat("/proc/sys/net/ipv6"); err != nil {
		return
	}
	if readSysctl("/proc/sys/net/ipv6/conf/all/forwarding") != "1" {
		interfaces, _ := filepath.Glob("/proc/sys/net/ipv6/conf/*/accept_ra")
		for _, path := range interfaces {
			if readSysctl(path) == "1" {
				setSysctl(path, "2")
			}
		}
	}
	setSysctl("/proc/sys/net/ipv6/conf/all/forwarding", "1")
}

// restoreRouterHost puts back the kernel settings prepareRouterHost changed: forwarding first, so
// no interface stops taking router advertisements while it is on.
func restoreRouterHost() {
	routerHost.Lock()
	defer routerHost.Unlock()
	for _, path := range []string{"/proc/sys/net/ipv6/conf/all/forwarding", "/proc/sys/net/ipv4/ip_forward"} {
		if value, changed := routerHost.saved[path]; changed {
			writeSysctl(path, value)
			delete(routerHost.saved, path)
		}
	}
	for path, value := range routerHost.saved {
		writeSysctl(path, value)
	}
	routerHost.saved = nil
}

// setSysctl sets path to value and remembers what it was, the first time it changes it.
func setSysctl(path string, value string) {
	old := readSysctl(path)
	if old == "" || old == value {
		return
	}
	if err := writeSysctl(path, value); err != nil {
		Log(LogLevel_WARNING, LogType_CORE, "router mode: could not set ", path, " to ", value, ": ", err.Error())
		return
	}
	if _, kept := routerHost.saved[path]; !kept {
		routerHost.saved[path] = old
	}
	Log(LogLevel_INFO, LogType_CORE, "router mode: set ", path, " to ", value, " (was ", old, ")")
}

func readSysctl(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func writeSysctl(path string, value string) error {
	return os.WriteFile(path, []byte(value), 0o644)
}
