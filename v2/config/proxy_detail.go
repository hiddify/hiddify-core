package config

import (
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// ProxyDetail is what a proxy's options say beyond its protocol, as the app shows it after the
// protocol's name: the transport, then the security, "TCP · REALITY", "XHTTP · TLS", "WS". It is
// "" for a protocol that has no transport of its own to choose (Shadowsocks, Hysteria2, WARP, ...)
// and for anything that is not a proxy. Two configs of the same protocol that a filter treats
// differently differ here: a REALITY one and a TLS one, a TCP one and an XHTTP one.
func ProxyDetail(options any) string {
	switch o := options.(type) {
	case *option.VLESSOutboundOptions:
		return joinDetail(transportName(o.Transport), securityName(o.TLS))
	case *option.VMessOutboundOptions:
		return joinDetail(transportName(o.Transport), securityName(o.TLS))
	case *option.TrojanOutboundOptions:
		return joinDetail(transportName(o.Transport), securityName(o.TLS))
	}
	return ""
}

// ProxyDetails are the details (ProxyDetail) of the outbounds and endpoints of options that have
// one, by tag.
func ProxyDetails(options *option.Options) map[string]string {
	details := make(map[string]string)
	if options == nil {
		return details
	}
	for _, out := range options.Outbounds {
		if detail := ProxyDetail(out.Options); detail != "" {
			details[out.Tag] = detail
		}
	}
	for _, end := range options.Endpoints {
		if detail := ProxyDetail(end.Options); detail != "" {
			details[end.Tag] = detail
		}
	}
	return details
}

// transportName is how the app names a V2Ray transport; none is plain TCP.
func transportName(transport *option.V2RayTransportOptions) string {
	if transport == nil {
		return "TCP"
	}
	switch transport.Type {
	case "", C.V2RayTransportTypeRaw:
		return "TCP"
	case C.V2RayTransportTypeWebsocket:
		return "WS"
	case C.V2RayTransportTypeGRPC:
		return "gRPC"
	case C.V2RayTransportTypeHTTP:
		return "HTTP"
	case C.V2RayTransportTypeHTTPUpgrade:
		return "HTTPUpgrade"
	case C.V2RayTransportTypeXHTTP:
		return "XHTTP"
	case C.V2RayTransportTypeQUIC:
		return "QUIC"
	case C.V2RayTransportTypeKCP, C.V2RayTransportTypeKCPAlias:
		return "mKCP"
	case C.V2RayTransportTypeXDrive:
		return "XDrive"
	}
	return transport.Type
}

// securityName is REALITY, TLS, or nothing for a proxy without TLS.
func securityName(tls *option.OutboundTLSOptions) string {
	switch {
	case tls == nil || !tls.Enabled:
		return ""
	case tls.Reality != nil && tls.Reality.Enabled:
		return "REALITY"
	}
	return "TLS"
}

func joinDetail(parts ...string) string {
	detail := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if detail != "" {
			detail += " · "
		}
		detail += part
	}
	return detail
}
