package hcore

import (
	"context"
	"runtime"
	"sync"

	"github.com/hiddify/hiddify-core/v2/config"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/common/settings"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
)

// A core running as root for a user (the admin_bgservice on Windows and Linux) cannot set that
// user's system proxy: the user's desktop session (D-Bus, Wayland) and registry are out of its
// reach. It runs its config without set_system_proxy and reports the address; the app then asks
// its own core, which runs as the user, to point the system proxy there.
var systemProxyState struct {
	access sync.Mutex
	// reported by the root core: where the config's system proxy pointed
	delegated *config.SystemProxyTarget
	// set by the user's core on the app's request
	proxy settings.SystemProxy
}

func recordDelegatedSystemProxy(options *option.Options) {
	var target *config.SystemProxyTarget
	if hopts := static.HiddifyOptions; hopts != nil && hopts.DelegateSystemProxy {
		target = config.TakeSystemProxy(options)
	}
	systemProxyState.access.Lock()
	systemProxyState.delegated = target
	systemProxyState.access.Unlock()
}

// DelegatedSystemProxy is where the running config's system proxy points, for the user's core to
// set; nil when it has none, or when this core sets its own.
func DelegatedSystemProxy() *config.SystemProxyTarget {
	systemProxyState.access.Lock()
	defer systemProxyState.access.Unlock()
	if static.CoreState != CoreStates_STARTED {
		return nil
	}
	return systemProxyState.delegated
}

// SystemProxyDelegationSupported reports whether a root core here hands the system proxy to the
// user's core. On macOS the root daemon sets it itself: networksetup applies to every user.
func SystemProxyDelegationSupported() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "linux"
}

func (s *CoreService) SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest) (*hcommon.Response, error) {
	var err error
	if in.IsEnabled {
		err = EnableUserSystemProxy(in.Address, in.SupportSocks)
	} else {
		err = DisableUserSystemProxy()
	}
	if err != nil {
		return &hcommon.Response{Code: hcommon.ResponseCode_FAILED, Message: err.Error()}, nil
	}
	return &hcommon.Response{Code: hcommon.ResponseCode_OK}, nil
}

func (s *CoreService) GetSystemProxyStatus(ctx context.Context, empty *hcommon.Empty) (*SystemProxyStatus, error) {
	systemProxyState.access.Lock()
	defer systemProxyState.access.Unlock()
	proxy := systemProxyState.proxy
	return &SystemProxyStatus{
		Available: SystemProxyDelegationSupported(),
		Enabled:   proxy != nil && proxy.IsEnabled(),
	}, nil
}

// EnableUserSystemProxy points the system proxy of the user this core runs as at address.
func EnableUserSystemProxy(address string, supportSOCKS bool) error {
	if !SystemProxyDelegationSupported() {
		return E.New("set system proxy: not delegated on ", runtime.GOOS)
	}
	server := M.ParseSocksaddr(address)
	if !server.IsValid() {
		return E.New("set system proxy: invalid address ", address)
	}
	systemProxyState.access.Lock()
	defer systemProxyState.access.Unlock()
	disableUserSystemProxyLocked()
	proxy, err := settings.NewSystemProxy(context.Background(), server, supportSOCKS, nil)
	if err != nil {
		return E.Cause(err, "set system proxy")
	}
	if err := proxy.Enable(); err != nil {
		_ = proxy.Close()
		return E.Cause(err, "set system proxy")
	}
	systemProxyState.proxy = proxy
	return nil
}

// DisableUserSystemProxy restores the system proxy EnableUserSystemProxy set, if any.
func DisableUserSystemProxy() error {
	systemProxyState.access.Lock()
	defer systemProxyState.access.Unlock()
	return disableUserSystemProxyLocked()
}

func disableUserSystemProxyLocked() error {
	proxy := systemProxyState.proxy
	if proxy == nil {
		return nil
	}
	systemProxyState.proxy = nil
	err := proxy.Disable()
	_ = proxy.Close()
	return err
}
