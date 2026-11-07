//go:build windows || (linux && !android)

package hcore

import (
	"context"

	"github.com/sagernet/sing-box/common/settings"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
)

// restoreSystemProxy turns the system proxy off, as the inbound that set it does when it closes.
func restoreSystemProxy(address string) error {
	server := M.ParseSocksaddr(address)
	if !server.IsValid() {
		return E.New("invalid address")
	}
	proxy, err := settings.NewSystemProxy(context.Background(), server, true, nil)
	if err != nil {
		return err
	}
	defer proxy.Close()
	return proxy.Disable()
}
