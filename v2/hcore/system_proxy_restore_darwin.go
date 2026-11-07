//go:build darwin && !ios

package hcore

import (
	"net"
	"os/exec"
	"strings"
)

// restoreSystemProxy turns off each proxy of a network service that still points at address
// (the inbound sets the web, secure web and SOCKS proxies of the active service); a proxy the
// user pointed elsewhere since is left alone.
func restoreSystemProxy(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, service := range lines {
		// the first line explains that "*" marks a disabled service
		if i == 0 || service == "" || strings.HasPrefix(service, "*") {
			continue
		}
		for _, kind := range []string{"webproxy", "securewebproxy", "socksfirewallproxy"} {
			out, err := exec.Command("networksetup", "-get"+kind, service).Output()
			if err != nil || !proxyPointsAt(string(out), host, port) {
				continue
			}
			if err := exec.Command("networksetup", "-set"+kind+"state", service, "off").Run(); err != nil {
				return err
			}
		}
	}
	return nil
}

// proxyPointsAt reads networksetup's "Enabled: Yes / Server: … / Port: …" lines.
func proxyPointsAt(info string, host string, port string) bool {
	values := map[string]string{}
	for _, line := range strings.Split(info, "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return values["Enabled"] == "Yes" && values["Server"] == host && values["Port"] == port
}
