package config

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// With keep-top, a profile with many proxies runs only its fastest few once their first
// pings are in (hcore picks them). These helpers find a profile's proxies and drop the others from
// its options before the config is built; the extra security and unblocker hops are not the
// profile's, so they always stay.

// UserProxyTags are the proxies of the profile that end up in the select group: what setOutbounds
// lists, before the chain hop is added.
func UserProxyTags(input *option.Options) []string {
	var tags []string
	for _, out := range input.Outbounds {
		if isUserProxy(out.Tag, out.Type) && !isDNSOutbound(out) {
			tags = append(tags, out.Tag)
		}
	}
	for _, end := range input.Endpoints {
		if isUserProxy(end.Tag, end.Type) {
			tags = append(tags, end.Tag)
		}
	}
	return tags
}

// ChosenProxies are the tags of chosen that are the profile's proxies (UserProxyTags), to keep.
func ChosenProxies(input *option.Options, chosen []string) map[string]bool {
	keep := make(map[string]bool)
	for _, tag := range UserProxyTags(input) {
		if slices.Contains(chosen, tag) {
			keep[tag] = true
		}
	}
	return keep
}

func isUserProxy(tag, typ string) bool {
	if contains(PredefinedOutboundTags, tag) || contains([]string{"direct", "bypass", "block"}, tag) {
		return false
	}
	if strings.Contains(tag, "§hide§") {
		return false
	}
	switch typ {
	case C.TypeBlock, C.TypeSelector, C.TypeURLTest, C.TypeCustom, C.TypeDNS:
		return false
	}
	return true
}

// KeepOnlyProxies drops the profile's proxies that are not in keep, except those a kept one goes
// through (its detour, and theirs), and returns how many were dropped.
func KeepOnlyProxies(ctx context.Context, input *option.Options, keep map[string]bool) int {
	needed := make(map[string]bool, len(keep))
	for tag := range keep {
		needed[tag] = true
	}
	looked := make(map[string]bool)
	for grew := true; grew; {
		grew = false
		for i := range input.Outbounds {
			if tag := input.Outbounds[i].Tag; needed[tag] && !looked[tag] {
				looked[tag], grew = true, true
				content, _ := input.Outbounds[i].MarshalJSONContext(ctx)
				addDetours(content, needed)
			}
		}
		for i := range input.Endpoints {
			if tag := input.Endpoints[i].Tag; needed[tag] && !looked[tag] {
				looked[tag], grew = true, true
				content, _ := input.Endpoints[i].MarshalJSONContext(ctx)
				addDetours(content, needed)
			}
		}
	}
	dropped := 0
	outbounds := input.Outbounds[:0]
	for _, out := range input.Outbounds {
		if isUserProxy(out.Tag, out.Type) && !isDNSOutbound(out) && !needed[out.Tag] {
			dropped++
			continue
		}
		outbounds = append(outbounds, out)
	}
	input.Outbounds = outbounds
	endpoints := input.Endpoints[:0]
	for _, end := range input.Endpoints {
		if isUserProxy(end.Tag, end.Type) && !needed[end.Tag] {
			dropped++
			continue
		}
		endpoints = append(endpoints, end)
	}
	input.Endpoints = endpoints
	return dropped
}

// addDetours adds every "detour" an outbound's options name, at any depth, to needed.
func addDetours(content []byte, needed map[string]bool) {
	var value any
	if json.Unmarshal(content, &value) != nil {
		return
	}
	walkJSON(value, func(object map[string]any, key string, field any) bool {
		if tag, ok := field.(string); ok && key == "detour" && tag != "" {
			needed[tag] = true
		}
		return true
	})
}

// walkJSON visits every field of every object in a JSON tree, depth first: visit gets the object,
// the key and the field, and says whether to go on into the field.
func walkJSON(value any, visit func(object map[string]any, key string, field any) bool) {
	switch v := value.(type) {
	case map[string]any:
		for key, field := range v {
			if visit(v, key, field) {
				walkJSON(field, visit)
			}
		}
	case []any:
		for _, field := range v {
			walkJSON(field, visit)
		}
	}
}
