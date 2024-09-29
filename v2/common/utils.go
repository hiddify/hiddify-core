package common

import (
	"net"
	"net/netip"
	"time"
)

func CanConnectIPv6Addr(remoteAddr netip.AddrPort) bool {
	dialer := net.Dialer{
		Timeout: 1 * time.Second,
	}

	conn, err := dialer.Dial("tcp6", remoteAddr.String())
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}

func CanConnectIPv6() bool {
	return CanConnectIPv6Addr(netip.MustParseAddrPort("[2001:4860:4860::8888]:80"))
}
