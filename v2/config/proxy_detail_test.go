package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// A share link's transport and security are what the app shows after the protocol: a REALITY
// config and a TLS one, a TCP one and an XHTTP one, look different in the list.
func TestProxyDetailOfLinks(t *testing.T) {
	links := map[string]string{
		"vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443?security=reality&pbk=7j3hmlHrEDk3Kjx-dFG3E3jf0c6PZF45trIPFsuKWmI&fp=chrome&type=tcp&flow=xtls-rprx-vision&sni=dl.google.com#reality-tcp": "TCP · REALITY",
		"vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443?security=reality&pbk=7j3hmlHrEDk3Kjx-dFG3E3jf0c6PZF45trIPFsuKWmI&fp=chrome&type=xhttp&path=%2F&sni=dl.google.com#reality-xhttp":          "XHTTP · REALITY",
		"vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443?security=reality&pbk=7j3hmlHrEDk3Kjx-dFG3E3jf0c6PZF45trIPFsuKWmI&fp=chrome&type=grpc&serviceName=x&sni=dl.google.com#reality-grpc":       "gRPC · REALITY",
		"vless://11111111-1111-1111-1111-111111111111@example.com:443?security=tls&type=ws&path=%2Fws&sni=example.com#tls-ws":                                                                              "WS · TLS",
		"vless://11111111-1111-1111-1111-111111111111@1.2.3.4:80?security=none&type=ws&path=%2Fws#plain-ws":                                                                                                "WS",
		"trojan://secret@example.com:443?security=tls&sni=example.com#trojan":                                                                                                                              "TCP · TLS",
		"ss://YWVzLTI1Ni1nY206c2VjcmV0@1.2.3.4:8388#ss": "",
	}
	for link, want := range links {
		options, err := ParseConfig(context.Background(), &ReadOptions{Content: link}, false, nil, false)
		require.NoError(t, err, link)
		details := ProxyDetails(options)
		found := false
		for _, out := range options.Outbounds {
			if detail, ok := details[out.Tag]; ok {
				require.Equal(t, want, detail, link)
				found = true
			}
		}
		require.Equal(t, want != "", found, link)
	}
}
