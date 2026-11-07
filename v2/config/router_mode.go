package config

import (
	"errors"
	"fmt"

	"github.com/hiddify/hiddify-core/v2/hutils"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// Router mode is the TUN with auto_redirect, on a Linux router or gateway. auto_redirect is what
// makes it a router: its nftables prerouting chain takes the traffic the router forwards for its
// network, and its mark mode routes only the packets it chose into the TUN, so connections from
// outside (to the router itself, or port forwarded to a device behind it) and their replies keep
// the main routing table. The plain TUN has neither: on OpenWrt the firewall drops what the network
// sends to it, and replies to a port forward are routed into it. So router mode never falls back to
// the plain TUN (see RouterModeError).

// ErrRouterModeUnsupported is the error of router mode on a system other than Linux.
var ErrRouterModeUnsupported = errors.New("router mode works only on Linux (a router, a gateway or a container on one)")

// ErrRouterModeNeedsRoot is the error of router mode without root: it changes the routing and the
// firewall of the whole system.
var ErrRouterModeNeedsRoot = errors.New("router mode needs root: it changes the routing and the firewall of the system")

// forRouterMode is hopts as router mode runs them: the TUN on, with auto_redirect, whatever the VPN
// settings say. Unchanged when router mode is off; an error where router mode can not run.
func forRouterMode(hopts *HiddifyOptions) (*HiddifyOptions, error) {
	if !hopts.RouterMode {
		return hopts, nil
	}
	if !C.IsLinux || C.IsAndroid {
		return nil, ErrRouterModeUnsupported
	}
	if !hutils.IsAdmin() {
		return nil, ErrRouterModeNeedsRoot
	}
	adapted := *hopts
	adapted.EnableTun = true
	adapted.AutoRedirect = true
	return &adapted, nil
}

// tunAutoRedirect says whether the TUN built for hopts asks for auto_redirect: always in router
// mode, which has no TUN without it; else where the setting and the system allow it.
func tunAutoRedirect(hopts *HiddifyOptions) bool {
	if hopts.RouterMode {
		return true
	}
	return hopts.AutoRedirect && AutoRedirectAvailable()
}

// RouterModeError is the error of a router mode start that failed because of auto_redirect, with
// what the system lacks. Router mode does not start without it (see the top of this file).
func RouterModeError(err error) error {
	return fmt.Errorf("router mode needs nftables and the nfqueue kernel module "+
		"(OpenWrt: kmod-nft-queue; Debian, Ubuntu: nftables and the nfnetlink_queue module): %w", err)
}

// DropAutoRedirect turns auto_redirect off in the TUN inbounds of options: for a start again
// without it, when options came as a raw config, which a rebuild does not change.
func DropAutoRedirect(options *option.Options) {
	if options == nil {
		return
	}
	for _, inbound := range options.Inbounds {
		if tunOptions, isTun := inbound.Options.(*option.TunInboundOptions); isTun {
			tunOptions.AutoRedirect = false
		}
	}
}
