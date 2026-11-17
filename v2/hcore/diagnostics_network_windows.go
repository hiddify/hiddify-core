package hcore

import (
	"sort"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

// windowsInterface is what sing-tun's default interface monitor (monitor_windows.go) looks at to
// pick the interface the core binds its connections to: the lowest metric default route on an
// interface that is up, connected, not virtual and has no DisableDefaultRoutes, which a RAS
// (dial-up, PPPoE, built-in VPN) connection sets on every other interface. Binding to one with
// no usable address or default route fails every dial with WSAEADDRNOTAVAIL.
type windowsInterface struct {
	Index uint32           `json:"index"`
	Name  string           `json:"name"`
	Type  uint32           `json:"type"`
	Up    bool             `json:"up"`
	IPv4  *windowsIPFamily `json:"ipv4,omitempty"`
	IPv6  *windowsIPFamily `json:"ipv6,omitempty"`
}

type windowsIPFamily struct {
	Connected            bool    `json:"connected"`
	DisableDefaultRoutes bool    `json:"disableDefaultRoutes"`
	Metric               uint32  `json:"metric"`
	AutomaticMetric      bool    `json:"automaticMetric"`
	Forwarding           bool    `json:"forwarding"`
	WeakHostSend         bool    `json:"weakHostSend"`
	Addresses            int     `json:"addresses"`
	DefaultRouteMetric   *uint32 `json:"defaultRouteMetric,omitempty"`
}

// systemInterfaces lists the IP interfaces that are up or have a default route, by index.
func systemInterfaces() any {
	interfaces := map[winipcfg.LUID]*windowsInterface{}
	for _, family := range []winipcfg.AddressFamily{windows.AF_INET, windows.AF_INET6} {
		rows, err := winipcfg.GetIPInterfaceTable(family)
		if err != nil {
			continue
		}
		states := map[winipcfg.LUID]*windowsIPFamily{}
		for _, row := range rows {
			state := &windowsIPFamily{
				Connected:            row.Connected,
				DisableDefaultRoutes: row.DisableDefaultRoutes,
				Metric:               row.Metric,
				AutomaticMetric:      row.UseAutomaticMetric,
				Forwarding:           row.ForwardingEnabled,
				WeakHostSend:         row.WeakHostSend,
			}
			states[row.InterfaceLUID] = state
			iface := interfaces[row.InterfaceLUID]
			if iface == nil {
				iface = &windowsInterface{Index: row.InterfaceIndex}
				if ifRow, err := row.InterfaceLUID.Interface(); err == nil {
					iface.Name = ifRow.Alias()
					iface.Type = uint32(ifRow.Type)
					iface.Up = ifRow.OperStatus == winipcfg.IfOperStatusUp
				}
				interfaces[row.InterfaceLUID] = iface
			}
			if family == windows.AF_INET {
				iface.IPv4 = state
			} else {
				iface.IPv6 = state
			}
		}
		if addresses, err := winipcfg.GetUnicastIPAddressTable(family); err == nil {
			for _, address := range addresses {
				if state := states[address.InterfaceLUID]; state != nil {
					state.Addresses++
				}
			}
		}
		if routes, err := winipcfg.GetIPForwardTable2(family); err == nil {
			for _, route := range routes {
				state := states[route.InterfaceLUID]
				if state == nil || route.DestinationPrefix.PrefixLength != 0 {
					continue
				}
				if state.DefaultRouteMetric == nil || route.Metric < *state.DefaultRouteMetric {
					metric := route.Metric
					state.DefaultRouteMetric = &metric
				}
			}
		}
	}
	list := make([]*windowsInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Type == winipcfg.IfTypeSoftwareLoopback {
			continue
		}
		if iface.Up || hasDefaultRoute(iface.IPv4) || hasDefaultRoute(iface.IPv6) {
			list = append(list, iface)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Index < list[j].Index })
	return list
}

func hasDefaultRoute(state *windowsIPFamily) bool {
	return state != nil && state.DefaultRouteMetric != nil
}
