package hutils

import (
	"fmt"
	"net"
)

func IsPortInUse(port uint16) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return true
	}
	defer listener.Close()
	return false
}
