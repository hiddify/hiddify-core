package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// ProxyFingerprints tells each proxy of the profile by its options, so a proxy whose tag stayed
// but whose server or settings changed is seen as another one.
func ProxyFingerprints(ctx context.Context, input *option.Options) map[string]string {
	fingerprints := make(map[string]string)
	add := func(tag string, content []byte, err error) {
		if err != nil {
			return
		}
		sum := sha256.Sum256(content)
		fingerprints[tag] = hex.EncodeToString(sum[:])
	}
	for i := range input.Outbounds {
		out := &input.Outbounds[i]
		if isUserProxy(out.Tag, out.Type) && !isDNSOutbound(*out) {
			content, err := out.MarshalJSONContext(ctx)
			add(out.Tag, content, err)
		}
	}
	for i := range input.Endpoints {
		end := &input.Endpoints[i]
		if isUserProxy(end.Tag, end.Type) {
			content, err := end.MarshalJSONContext(ctx)
			add(end.Tag, content, err)
		}
	}
	return fingerprints
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
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for key, item := range v {
				if tag, ok := item.(string); ok && key == "detour" && tag != "" {
					needed[tag] = true
				} else {
					walk(item)
				}
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(value)
}
