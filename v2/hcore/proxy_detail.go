package hcore

import (
	"strings"
	"sync/atomic"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// runningProxyDetails are the details (config.ProxyDetail) of the proxies of the config the
// engine runs, by tag: read from its options, so a proxy keep top leaves at rest, which has no
// real outbound, shows them as well as one in use.
var runningProxyDetails atomic.Pointer[map[string]string]

// rememberProxyDetails keeps the details of the config the engine runs now: after a start, or a
// hot reload, that applied it.
func rememberProxyDetails(options *option.Options) {
	details := config.ProxyDetails(options)
	runningProxyDetails.Store(&details)
}

// displayTypeWithDetail is the type the app shows for outbound: the display type, with the
// proxy's transport and security after its protocol's name, "VLESS · XHTTP · REALITY", before
// anything the display type adds (why it failed).
func displayTypeWithDetail(outbound adapter.Outbound) string {
	display := adapter.DisplayType(outbound)
	details := runningProxyDetails.Load()
	if details == nil {
		return display
	}
	detail := (*details)[outbound.Tag()]
	if detail == "" {
		return display
	}
	name := C.ProxyDisplayName(outbound.Type())
	if !strings.HasPrefix(display, name) {
		return display
	}
	return name + " · " + detail + display[len(name):]
}
