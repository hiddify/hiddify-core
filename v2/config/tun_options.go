package config

import (
	"strings"
	"sync/atomic"

	"github.com/hiddify/hiddify-core/v2/hutils"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// TunStack is the TUN `stack` option for the app's setting. "sing15" (the app's name for it),
// "auto" and nothing are sing-tun's own TCP/IP stack, the default since sing-box 1.15: no option
// is sent. The legacy names pass through for a user who chose one, with sing-box's deprecation
// note, until sing-box 1.17 removes them.
func TunStack(setting string) string {
	switch setting {
	case "", "sing15", "auto", "go":
		return ""
	}
	return setting
}

// autoRedirectUnavailable is set once a TUN failed to start because of auto_redirect: the
// system lacks nftables, iptables or the nfqueue module. The next build leaves auto_redirect
// out instead of failing the same way again.
var autoRedirectUnavailable atomic.Bool

// AutoRedirectAvailable says whether a TUN built here gets auto_redirect: on Linux, where it
// improves routing and performance and keeps Docker bridges working, when the process has the
// privileges it needs, and not after it failed on this system. Android is left out: there it
// needs a root service the app does not have.
func AutoRedirectAvailable() bool {
	return C.IsLinux && !C.IsAndroid && hutils.IsAdmin() && !autoRedirectUnavailable.Load()
}

// AutoRedirectFailed says whether a start of options failed because of their TUN's auto_redirect,
// and remembers that, so a config built after it goes without. A TUN with auto_redirect fails
// as a whole when nftables, iptables or the nfqueue module is missing; without it the TUN works
// as before.
func AutoRedirectFailed(options *option.Options, err error) bool {
	if err == nil || options == nil || !strings.Contains(err.Error(), "auto-redirect") {
		return false
	}
	for _, inbound := range options.Inbounds {
		if tunOptions, isTun := inbound.Options.(*option.TunInboundOptions); isTun && tunOptions.AutoRedirect {
			autoRedirectUnavailable.Store(true)
			return true
		}
	}
	return false
}
