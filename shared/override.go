package shared

import (
	"fmt"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

type ConfigOverrides struct {
	ExcludeTunInbound   bool
	IncludeMixedInbound bool
	IncludeLogOutput    bool
	IncludeLogTimestamp bool
	LogLevel            string
	ClashApiPort        int
}

func ApplyOverrides(options option.Options, overrides ConfigOverrides) option.Options {
	options.Log = &option.LogOptions{
		Disabled:     false,
		Timestamp:    overrides.IncludeLogTimestamp,
		DisableColor: true,
	}
	if overrides.LogLevel != "" {
		options.Log.Level = overrides.LogLevel
	}
	if overrides.IncludeLogOutput {
		options.Log.Output = "box.log"
	}

	options.Experimental.ClashAPI = &option.ClashAPIOptions{
		ExternalController: fmt.Sprintf("%s:%d", "127.0.0.1", overrides.ClashApiPort),
		StoreSelected:      true,
		Secret:             "",
	}

	var inbounds []option.Inbound
	for _, inb := range options.Inbounds {
		if overrides.ExcludeTunInbound && inb.Type == C.TypeTun {
			continue
		}
		if overrides.IncludeMixedInbound && inb.Type == C.TypeMixed {
			inb.MixedOptions.SetSystemProxy = true
			inbounds = append(inbounds, inb)
			continue
		}
		inbounds = append(inbounds, inb)
	}
	options.Inbounds = inbounds

	hasSelector := false
	hasUrlTest := false
	var selectable []option.Outbound
	var urlTests []option.Outbound
	for _, out := range options.Outbounds {
		if out.Type == C.TypeSelector {
			hasSelector = true
		} else if out.Type == C.TypeURLTest {
			hasUrlTest = true
			urlTests = append(urlTests, out)
		}
		switch out.Type {
		case C.TypeDirect, C.TypeBlock, C.TypeDNS:
			continue
		}
		selectable = append(selectable, out)
	}
	var generatedUrlTest *option.Outbound
	if !hasUrlTest {
		var urlSelectOuts []string
		for _, out := range selectable {
			urlSelectOuts = append(urlSelectOuts, out.Tag)
		}
		generatedUrlTest = &option.Outbound{Type: C.TypeURLTest, Tag: "urltest", URLTestOptions: option.URLTestOutboundOptions{Outbounds: urlSelectOuts}}
		urlTests = append(urlTests, *generatedUrlTest)
	}
	if !hasSelector {
		var selectorOuts []string
		for _, out := range selectable {
			selectorOuts = append(selectorOuts, out.Tag)
		}
		for _, out := range urlTests {
			selectorOuts = append(selectorOuts, out.Tag)
		}

		defaultSelector := option.Outbound{Type: C.TypeSelector, Tag: "select", SelectorOptions: option.SelectorOutboundOptions{Outbounds: selectorOuts}}
		if generatedUrlTest != nil {
			defaultSelector.SelectorOptions.Default = generatedUrlTest.Tag
		}
		options.Outbounds = append([]option.Outbound{defaultSelector}, options.Outbounds...)
	}
	if generatedUrlTest != nil {
		options.Outbounds = append(options.Outbounds, *generatedUrlTest)
	}

	return options
}
