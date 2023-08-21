package shared

import (
	"fmt"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

type ConfigOverrides struct {
	ClashApiPort   *int    `json:"clash-api-port"`
	EnableTun      *bool   `json:"enable-tun"`
	SetSystemProxy *bool   `json:"set-system-proxy"`
	LogLevel       *string `json:"log-level"`
	LogOutput      *string `json:"log-output"`
	DNSRemote      *string `json:"dns-remote"`
	MixedPort      *int    `json:"mixed-port"`
}

func ApplyOverrides(base option.Options, options option.Options, overrides ConfigOverrides) option.Options {
	clashApiPort := pointerOrDefaultInt(overrides.ClashApiPort, 9090)
	base.Experimental = &option.ExperimentalOptions{
		ClashAPI: &option.ClashAPIOptions{
			ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", clashApiPort),
			StoreSelected:      true,
		},
	}

	base.Log = &option.LogOptions{
		Level:        pointerOrDefaultString(overrides.LogLevel, "info"),
		Output:       pointerOrDefaultString(overrides.LogOutput, ""),
		Disabled:     false,
		Timestamp:    false,
		DisableColor: true,
	}

	var inbounds []option.Inbound
	for _, inb := range base.Inbounds {
		switch inb.Type {
		case C.TypeTun:
			if pointerOrDefaultBool(overrides.EnableTun, true) {
				inbounds = append(inbounds, inb)
			}
		default:
			inbounds = append(inbounds, inb)
		}
	}
	base.Inbounds = inbounds

	var outbounds []option.Outbound
	var tags []string
	for _, out := range options.Outbounds {
		switch out.Type {
		case C.TypeDirect, C.TypeBlock, C.TypeDNS:
			continue
		case C.TypeSelector, C.TypeURLTest:
			continue
		default:
			tags = append(tags, out.Tag)
		}
		outbounds = append(outbounds, out)
	}

	urlTest := option.Outbound{
		Type: C.TypeURLTest,
		Tag:  "auto",
		URLTestOptions: option.URLTestOutboundOptions{
			Outbounds: tags,
		},
	}

	selector := option.Outbound{
		Type: C.TypeSelector,
		Tag:  "select",
		SelectorOptions: option.SelectorOutboundOptions{
			Outbounds: append([]string{urlTest.Tag}, tags...),
			Default:   urlTest.Tag,
		},
	}

	outbounds = append([]option.Outbound{selector, urlTest}, outbounds...)

	base.Outbounds = append(
		outbounds,
		[]option.Outbound{
			{
				Tag:  "direct",
				Type: C.TypeDirect,
			},
			{
				Tag:  "block",
				Type: C.TypeBlock,
			},
			{
				Tag:  "dns-out",
				Type: C.TypeDNS,
			},
		}...,
	)

	return base
}
