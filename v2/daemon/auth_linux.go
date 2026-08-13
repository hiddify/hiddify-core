//go:build linux

package daemon

import (
	"net"
	"os"
	"syscall"
)

func socketMode(allowedUID int) os.FileMode {
	if allowedUID >= 0 {
		return 0o600
	}
	return 0o660
}

type authorizedListener struct {
	net.Listener
	allowedUID int
}

func authorizeListener(listener net.Listener, allowedUID int) net.Listener {
	if allowedUID < 0 {
		return listener
	}
	return &authorizedListener{Listener: listener, allowedUID: allowedUID}
}

func (listener *authorizedListener) Accept() (net.Conn, error) {
	for {
		connection, err := listener.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if unixConnection, ok := connection.(*net.UnixConn); ok && peerAllowed(unixConnection, listener.allowedUID) {
			return connection, nil
		}
		connection.Close()
	}
}

func peerAllowed(connection *net.UnixConn, allowedUID int) bool {
	raw, err := connection.SyscallConn()
	if err != nil {
		return false
	}
	var credentials *syscall.Ucred
	var controlErr error
	err = raw.Control(func(fd uintptr) {
		credentials, controlErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil || controlErr != nil || credentials == nil {
		return false
	}
	return credentials.Uid == 0 || int(credentials.Uid) == allowedUID
}
