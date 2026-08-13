//go:build !linux

package daemon

import (
	"net"
	"os"
)

func socketMode(allowedUID int) os.FileMode {
	if allowedUID >= 0 {
		return 0o600
	}
	return 0o660
}

func authorizeListener(listener net.Listener, allowedUID int) net.Listener {
	return listener
}
