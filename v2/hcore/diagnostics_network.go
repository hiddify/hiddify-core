package hcore

import (
	"github.com/sagernet/sing/common/control"
)

// networkDiagnostics is network.json in the sensitive part of a report: where the core's own
// connections leave the device.
//
// With auto_detect_interface the core binds every outbound socket to the default interface its
// monitor picks, and says which one only in an info line of box.log, which a report made at the
// warn level does not have. A wrong pick fails every dial the same way a broken network does
// (hiddify-app#2611: "The requested address is not valid in its context" for every server, DNS
// server and direct connection), so the report always names the pick and, where the system tells
// (Windows), the state of every interface it was picked from.
func networkDiagnostics() map[string]any {
	report := map[string]any{"coreRunning": false}
	if box := static.Box(); box != nil {
		report["coreRunning"] = true
		if monitor := box.Network().InterfaceMonitor(); monitor == nil {
			report["defaultInterface"] = "none: the config does not bind to the default interface"
		} else if defaultInterface := monitor.DefaultInterface(); defaultInterface == nil {
			report["defaultInterface"] = "none: the monitor found no default route"
		} else {
			report["defaultInterface"] = interfaceSummary(defaultInterface)
		}
	}
	if interfaces := systemInterfaces(); interfaces != nil {
		report["interfaces"] = interfaces
	}
	return report
}

// interfaceSummary names an interface and counts its addresses, without the addresses.
func interfaceSummary(iface *control.Interface) map[string]any {
	ipv4, ipv6 := 0, 0
	for _, prefix := range iface.Addresses {
		if prefix.Addr().Unmap().Is4() {
			ipv4++
		} else {
			ipv6++
		}
	}
	return map[string]any{
		"name":  iface.Name,
		"index": iface.Index,
		"ipv4":  ipv4,
		"ipv6":  ipv6,
	}
}
