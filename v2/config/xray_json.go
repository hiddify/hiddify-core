package config

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/xtls/libxray/share"
)

// An Xray JSON config, or a list of them (an Xray subscription), is read through share links: libxray writes a
// link for each proxy outbound and ray2sing reads it like any other link.

var xrayShareProtocols = map[string]bool{"vless": true, "vmess": true, "trojan": true, "shadowsocks": true, "socks": true, "hysteria": true}

// xrayJSONShareLinks returns the share links of parsed (decoded JSON), and whether it is Xray JSON.
func xrayJSONShareLinks(parsed any) (string, bool) {
	var configs []map[string]any
	switch value := parsed.(type) {
	case map[string]any:
		configs = []map[string]any{value}
	case []any:
		for _, item := range value {
			config, ok := item.(map[string]any)
			if !ok {
				return "", false
			}
			configs = append(configs, config)
		}
	default:
		return "", false
	}
	var links []string
	seen := map[string]bool{}
	isXray := false
	for _, config := range configs {
		outbounds, _ := config["outbounds"].([]any)
		for _, item := range outbounds {
			outbound, _ := item.(map[string]any)
			protocol, _ := outbound["protocol"].(string)
			if protocol == "" || outbound["type"] != nil {
				continue
			}
			isXray = true
			if !xrayShareProtocols[protocol] {
				continue
			}
			link := xrayOutboundShareLink(outbound)
			if link != "" && !seen[link] {
				seen[link] = true
				links = append(links, link)
			}
		}
	}
	return strings.Join(links, "\n"), isXray && len(links) > 0
}

func xrayOutboundShareLink(outbound map[string]any) string {
	flattened := make(map[string]any, len(outbound))
	for key, value := range outbound {
		flattened[key] = value
	}
	if settings, ok := outbound["settings"].(map[string]any); ok {
		flattened["settings"] = flattenXraySettings(settings)
	}
	if stream, ok := outbound["streamSettings"].(map[string]any); ok {
		flattened["streamSettings"] = normalizeXrayStream(stream)
	}
	content, err := json.Marshal(map[string]any{"outbounds": []any{flattened}})
	if err != nil {
		return ""
	}
	links, err := share.ConvertXrayJsonToShareLinks(content)
	if err != nil {
		return ""
	}
	link := withCipherSuites(strings.TrimSpace(links), outbound)
	if strings.HasPrefix(link, "vmess://") {
		return vmessURIToV2rayN(link)
	}
	return link
}

// withCipherSuites adds the outbound's TLS cipherSuites to its link, which libxray leaves out (links have no field
// for them); ray2sing reads it back.
func withCipherSuites(link string, outbound map[string]any) string {
	stream, _ := outbound["streamSettings"].(map[string]any)
	tlsSettings, _ := stream["tlsSettings"].(map[string]any)
	suites, _ := tlsSettings["cipherSuites"].(string)
	if link == "" || suites == "" {
		return link
	}
	link, fragment, hasFragment := strings.Cut(link, "#")
	separator := "?"
	if strings.Contains(link, "?") {
		separator = "&"
	}
	link += separator + "cipherSuites=" + url.QueryEscape(suites)
	if hasFragment {
		link += "#" + fragment
	}
	return link
}

// vmessURIToV2rayN rewrites libxray's vmess://id@host:port?... link as the v2rayN JSON link ray2sing reads.
func vmessURIToV2rayN(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	fields := map[string]string{"v": "2", "add": u.Hostname(), "port": u.Port(), "id": u.User.Username(), "aid": "0", "ps": u.Fragment}
	for key, values := range u.Query() {
		if len(values) > 0 {
			fields[strings.ToLower(key)] = values[0]
		}
	}
	fields["net"], fields["type"] = fields["type"], fields["headertype"]
	fields["tls"], fields["scy"] = fields["security"], fields["encryption"]
	if fields["net"] == "grpc" && fields["path"] == "" {
		fields["path"] = fields["servicename"]
	}
	for _, key := range []string{"headertype", "security", "encryption"} {
		delete(fields, key)
	}
	content, err := json.Marshal(fields)
	if err != nil {
		return ""
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(content)
}

// normalizeXrayStream moves the older names of a stream to the ones libxray writes links from:
//   - a transport's deprecated Host header (ws, httpupgrade, xhttp) to its "host": a CDN reached by address refuses
//     a request without it;
//   - "tcp" / "tcpSettings" to "raw" / "rawSettings": else the HTTP header of raw TCP is lost;
//   - REALITY's "publicKey" to "password".
func normalizeXrayStream(stream map[string]any) map[string]any {
	copied := make(map[string]any, len(stream))
	for key, value := range stream {
		copied[key] = value
	}
	if network, _ := stream["network"].(string); network == "" || network == "tcp" {
		if raw, ok := stream["tcpSettings"]; ok && stream["rawSettings"] == nil {
			copied["network"] = "raw"
			copied["rawSettings"] = raw
			delete(copied, "tcpSettings")
		}
	}
	if reality, ok := stream["realitySettings"].(map[string]any); ok && reality["password"] == nil && reality["publicKey"] != nil {
		fixed := make(map[string]any, len(reality)+1)
		for field, value := range reality {
			fixed[field] = value
		}
		fixed["password"] = reality["publicKey"]
		copied["realitySettings"] = fixed
	}
	for _, key := range []string{"wsSettings", "httpupgradeSettings", "xhttpSettings", "splithttpSettings"} {
		transport, ok := stream[key].(map[string]any)
		if !ok || transport["host"] != nil {
			continue
		}
		headers, _ := transport["headers"].(map[string]any)
		for name, value := range headers {
			if host, ok := value.(string); ok && strings.EqualFold(name, "host") {
				fixed := make(map[string]any, len(transport)+1)
				for field, v := range transport {
					fixed[field] = v
				}
				fixed["host"] = host
				copied[key] = fixed
			}
		}
	}
	return copied
}

// flattenXraySettings moves the first server of the "vnext" / "servers" form (VLESS, VMess / Trojan, Shadowsocks,
// SOCKS) to the flat form, the only one libxray writes links from.
func flattenXraySettings(settings map[string]any) map[string]any {
	flat := make(map[string]any, len(settings))
	for key, value := range settings {
		flat[key] = value
	}
	for _, key := range []string{"vnext", "servers"} {
		list, _ := settings[key].([]any)
		if len(list) == 0 {
			continue
		}
		server, _ := list[0].(map[string]any)
		delete(flat, key)
		for field, value := range server {
			if field != "users" {
				flat[field] = value
			}
		}
		users, _ := server["users"].([]any)
		if len(users) > 0 {
			if user, ok := users[0].(map[string]any); ok {
				for field, value := range user {
					if field == "user" || field == "pass" {
						continue
					}
					flat[field] = value
				}
				if name, ok := user["user"]; ok { // SOCKS
					flat["user"] = name
					flat["pass"] = user["pass"]
				}
			}
		}
	}
	return flat
}
