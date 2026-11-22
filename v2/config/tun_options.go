package config

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync/atomic"

	"github.com/hiddify/hiddify-core/v2/hutils"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"go4.org/netipx"
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

// tunAddresses are the TUN's own networks. Its gateway is the DNS server the system is given, so
// they stay in the TUN whatever route-exclude-address says.
var tunAddresses = []netip.Prefix{
	netip.MustParsePrefix("172.19.0.1/28"),
	netip.MustParsePrefix("fdfe:dcba:9876::1/126"),
}

// routeExcludePrefixes reads the route-exclude-address setting: an IP stands for itself (a /32 or
// /128 prefix), a CIDR prefix for its network, an IPv4-mapped IPv6 one for its IPv4 address.
// Blank entries, left by a list edited line by line, are skipped. A /0 is refused: it takes a
// whole address family out of the VPN, which is a typo far more often than a wish. The TUN's own
// networks are cut out of the result (a corporate 172.16.0.0/12 covers 172.19.0.0/28), and the
// rest is merged into the fewest prefixes.
func routeExcludePrefixes(addresses []string) ([]netip.Prefix, error) {
	var excluded netipx.IPSetBuilder
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}
		prefix, err := parseRouteExclude(address)
		if err != nil {
			return nil, fmt.Errorf("route-exclude-address: %q %w", address, err)
		}
		excluded.AddPrefix(prefix)
	}
	for _, own := range tunAddresses {
		excluded.RemovePrefix(own.Masked())
	}
	set, err := excluded.IPSet()
	if err != nil {
		return nil, fmt.Errorf("route-exclude-address: %w", err)
	}
	return set.Prefixes(), nil
}

func parseRouteExclude(address string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(address)
	if err != nil {
		ip, ipErr := netip.ParseAddr(address)
		if ipErr != nil || ip.Zone() != "" {
			return netip.Prefix{}, errors.New("is not an IP or a CIDR prefix")
		}
		prefix = netip.PrefixFrom(ip, ip.BitLen())
	}
	if prefix.Addr().Is4In6() {
		if prefix.Bits() < 96 {
			return netip.Prefix{}, errors.New("mixes IPv4-mapped and plain IPv6 addresses")
		}
		prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
	}
	if prefix.Bits() == 0 {
		return netip.Prefix{}, errors.New("would take every address of its family out of the VPN")
	}
	return prefix.Masked(), nil
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
