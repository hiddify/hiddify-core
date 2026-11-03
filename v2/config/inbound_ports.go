package config

import (
	"net/netip"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common/auth"

	"github.com/hiddify/hiddify-core/v2/hutils"
)

// ProxyUser is the user name of the mixed port; its password is the proxy secret.
const ProxyUser = "hiddify"

// transparentPortsOpen says whether tproxy, redirect and direct open: they serve no app in VPN
// mode, and take no password, so there only while the VPN is shared (a Linux gateway).
func transparentPortsOpen(hopt *HiddifyOptions) bool {
	return !hopt.EnableTun || hopt.AllowConnectionFromLAN
}

// portEnabled says whether a port opens: set, and its switch on, or unset and on by default.
func portEnabled(enabled *bool, port uint16, byDefault bool) bool {
	if enabled == nil {
		return port > 0 && byDefault
	}
	return port > 0 && *enabled
}

// localPortOpen says whether this device's own proxy port (the mixed port, on loopback) opens: in
// proxy and system proxy mode; in VPN mode, where every app already goes through the TUN, only when
// the user asked for it.
func localPortOpen(hopt *HiddifyOptions) bool {
	return portEnabled(hopt.EnableMixedPort, hopt.MixedPort, true) && (!hopt.EnableTun || hopt.LocalProxyInVpn)
}

// localUsers are the users the local port asks for: none unless the user asked for the secret, as
// browsers can not send a SOCKS5 password; and none as the system proxy, which can not pass one.
func localUsers(hopt *HiddifyOptions) []auth.User {
	if !hopt.LocalProxyAuth || hopt.SetSystemProxy {
		return secretUsers("")
	}
	return secretUsers(hopt.ProxySecret)
}

// secretUsers is the one user of a port that asks for [secret], or none for no secret.
func secretUsers(secret string) []auth.User {
	if secret == "" {
		return nil
	}
	return []auth.User{{Username: ProxyUser, Password: secret}}
}

// proxyListener is one address and port a mixed inbound listens on, with the users it asks for.
type proxyListener struct {
	tag   string
	addr  netip.Addr
	port  uint16
	users []auth.User
	// systemProxy: this listener is the one the system proxy points at
	systemProxy bool
}

// proxyListeners are the mixed inbounds: this device's local port on loopback (localPortOpen), and
// while the VPN is shared, the LAN sharing port on every address, always asking for the secret
// (open only if the user removed it). The two are apart, so the local port can stay without a
// password; on the same port, the shared one takes both, asking for the secret.
//
// Only the IPv4 loopback listener sets the system proxy: each listener that does so overwrites the
// setting as it starts, and the ::1 one, starting last, left Windows pointed at http://[::1]:port,
// which Telegram and other apps that read the system proxy can not use (5.5.1).
func proxyListeners(hopt *HiddifyOptions, ipv6 bool) []proxyListener {
	shared := hopt.AllowConnectionFromLAN && hopt.LanSharingPort > 0
	var listeners []proxyListener
	if localPortOpen(hopt) && !(shared && hopt.LanSharingPort == hopt.MixedPort) {
		loopback := []netip.Addr{netip.MustParseAddr("127.0.0.1")}
		if ipv6 {
			loopback = append(loopback, netip.MustParseAddr("::1"))
		}
		for _, addr := range loopback {
			listeners = append(listeners, proxyListener{
				tag:         InboundMixedTag + addr.String(),
				addr:        addr,
				port:        hopt.MixedPort,
				users:       localUsers(hopt),
				systemProxy: hopt.SetSystemProxy && addr.Is4(),
			})
		}
	}
	if shared {
		every := netip.MustParseAddr("0.0.0.0")
		if ipv6 {
			every = netip.MustParseAddr("::")
		}
		listeners = append(listeners, proxyListener{
			tag:         InboundLanSharingTag + every.String(),
			addr:        every,
			port:        hopt.LanSharingPort,
			users:       secretUsers(hopt.ProxySecret),
			systemProxy: hopt.SetSystemProxy && hopt.LanSharingPort == hopt.MixedPort && localPortOpen(hopt),
		})
	}
	return listeners
}

// tproxyAvailable: tproxy needs Linux's transparent sockets, which only root may open.
func tproxyAvailable() bool {
	return C.IsLinux && !C.IsAndroid && hutils.IsAdmin()
}

// redirectAvailable: redirect reads the original address from Linux netfilter or the macOS pf.
func redirectAvailable() bool {
	return (C.IsLinux || C.IsDarwin) && !C.IsAndroid && !C.IsIos
}
