package config

import (
	"net/netip"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
)

// SystemProxyTarget is where an inbound asked the system proxy to point.
type SystemProxyTarget struct {
	Address      string
	SupportSOCKS bool
}

// TakeSystemProxy turns off set_system_proxy on every inbound, the app's and a profile's own (a
// full config can bring a mixed or HTTP inbound beside the TUN), and returns where the first one
// pointed. A wildcard listen address is reached on the loopback.
func TakeSystemProxy(options *option.Options) *SystemProxyTarget {
	return systemProxy(options, true)
}

// SystemProxyOf is where the config's first set_system_proxy inbound points, nil when none has it.
func SystemProxyOf(options *option.Options) *SystemProxyTarget {
	return systemProxy(options, false)
}

func systemProxy(options *option.Options, take bool) *SystemProxyTarget {
	if options == nil {
		return nil
	}
	var target *SystemProxyTarget
	for _, inbound := range options.Inbounds {
		mixed, ok := inbound.Options.(*option.HTTPMixedInboundOptions)
		if !ok || mixed == nil || !mixed.SetSystemProxy {
			continue
		}
		if take {
			mixed.SetSystemProxy = false
		}
		if target != nil {
			continue
		}
		host := netip.AddrFrom4([4]byte{127, 0, 0, 1})
		if mixed.Listen != nil {
			if addr := netip.Addr(*mixed.Listen); addr.IsValid() && !addr.IsUnspecified() {
				host = addr
			}
		}
		target = &SystemProxyTarget{
			Address:      M.SocksaddrFrom(host, mixed.ListenPort).String(),
			SupportSOCKS: inbound.Type == C.TypeMixed,
		}
		if !take {
			break
		}
	}
	return target
}
