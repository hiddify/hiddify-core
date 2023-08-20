package shared

import (
	"bytes"
	"text/template"
)

const base = `
	{
		"log": {
			{{if .IncludeLogOutput}}
			"disabled": false,
			"level": "info",
			"output": "box.log",
			{{end}}
		},
		"dns": {
			"servers": [
				{
					"tag": "remote",
					"address_resolver": "local",
					"address": "tcp://1.1.1.1",
					"strategy": "prefer_ipv4",
					"detour": "select"
				},
				{
					"tag": "local",
					"address": "local",
					"detour": "direct"
				}
			],
			"rules": [
				{
					"clash_mode": "global",
					"server": "remote"
				},
				{
					"clash_mode": "direct",
					"server": "local"
				},
				{
					"outbound": [
						"any"
					],
					"server": "local"
				},
				{
					"geosite": "ir",
					"server": "local"
				}
			],
			"strategy": "ipv4_only"
		},
		"inbounds": [
			{{if .IncludeTunInbound}}
			{
				"type": "tun",
				"inet4_address": "172.19.0.1/30",
				"sniff": true,
				"sniff_override_destination": true,
				"domain_strategy": "ipv4_only",
				"strict_route": true,
				"mtu": 9000,
				"endpoint_independent_nat": true,
				"auto_route": true
			},
			{{end}}
			{
				"type": "socks",
				"tag": "socks-in",
				"listen": "127.0.0.1",
				"sniff": true,
				"sniff_override_destination": true,
				"domain_strategy": "ipv4_only",
				"listen_port": 2333,
				"users": []
			}
			{{if .IncludeMixedInbound}}
			,{
				"type": "mixed",
				"tag": "mixed-in",
				"sniff": true,
				"sniff_override_destination": true,
				"domain_strategy": "ipv4_only",
				"listen": "127.0.0.1",
				"listen_port": 2334,
				"set_system_proxy": true,
				"users": []
			}
			{{end}}
		],
		"outbounds": [
			{
				"type": "direct",
				"tag": "direct"
			},
			{
				"type": "block",
				"tag": "block"
			},
			{
				"type": "dns",
				"tag": "dns-out"
			}
		],
		"route": {
			"rules": [
				{
					"geosite": "category-ads-all",
					"outbound": "block"
				},
				{
					"protocol": "dns",
					"outbound": "dns-out"
				},
				{
					"clash_mode": "direct",
					"outbound": "direct"
				},
				{
					"clash_mode": "global",
					"outbound": "select"
				},
				{
					"geoip": [
						"ir",
						"private"
					],
					"outbound": "direct"
				},
				// {
				// 	"geosite": "geolocation-!ir",
				// 	"outbound": "select"
				// },
				{
					"geosite": "ir",
					"outbound": "direct"
				}
			],
			"auto_detect_interface": true
		},
		"experimental": {
			"clash_api": {
				"external_controller": "127.0.0.1:9090",
				"store_selected": true,
				"secret": ""
			}
		}
	}`

type ConfigTemplateOptions struct {
	IncludeTunInbound   bool
	IncludeMixedInbound bool
	IncludeLogOutput    bool
}

func defaultTemplate(options ConfigTemplateOptions) []byte {
	var buffer bytes.Buffer
	t := template.New("baseConfig")
	t, _ = t.Parse(base)
	t.Execute(&buffer, options)
	return buffer.Bytes()
}
