package config

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// With keep-top, a profile with many proxies uses only its fastest few. The core finds them
// itself (sing-box's keep top, experimental.monitoring.keep_top): the config keeps every proxy,
// the ones keep top manages as lazy outbounds, which the core creates when a connection or a test
// goes through them and closes when nothing does, so no config changes while it picks. These
// helpers find a profile's proxies and turn keep top on; for keep chosen they drop the others
// from its options before the config is built. The extra security and unblocker hops are not the
// profile's, so they always run.

// KeepTopAnswering is the keep-top that keeps every proxy answering its first ping and stops
// the ones that do not.
const KeepTopAnswering = -2

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

// lazyTypes are the proxies the core can create when they are used and close when they are not: a
// connection through one needs nothing that takes long to set up, and nothing else in the core
// holds it. Not the endpoints (WireGuard, WARP, ...), which the router holds as interfaces, nor
// the outbounds that search before they work (Psiphon, Tor, the DNS tunnels): those run for the
// whole run, and keep top leaves them alone.
var lazyTypes = map[string]bool{
	C.TypeVLESS: true, C.TypeVMess: true, C.TypeTrojan: true, C.TypeShadowsocks: true,
	C.TypeSOCKS: true, C.TypeHTTP: true, C.TypeHysteria: true, C.TypeHysteria2: true,
	C.TypeTUIC: true, C.TypeAnyTLS: true, C.TypeShadowTLS: true, C.TypeSSH: true,
	C.TypeNaive: true, C.TypeMieru: true, C.TypeSnell: true, C.TypeTrustTunnel: true,
	C.TypeSudoku: true,
}

// LazyProxyTags are the proxies of the profile that keep top manages: the ones of UserProxyTags
// that can be lazy outbounds, in the profile's order.
func LazyProxyTags(input *option.Options) []string {
	var tags []string
	for _, out := range input.Outbounds {
		if isUserProxy(out.Tag, out.Type) && !isDNSOutbound(out) && lazyTypes[out.Type] {
			tags = append(tags, out.Tag)
		}
	}
	return tags
}

// ApplyKeepTop turns keep top on in a built config, when the settings ask for it: the proxies it
// manages (candidates, LazyProxyTags) become lazy outbounds, and the monitoring gets the keep top
// options, by which the core finds the fastest of them and leaves the others at rest. It returns
// how many it manages; 0 when keep top is not on, and then nothing in the config is lazy.
//
// The proxies are lazy whatever the count, also when the profile has no more of them than it
// keeps (then none rests): the count is only an option of the monitoring, so changing it changes
// no outbound, and the running core takes it without a restart or a cut connection.
func ApplyKeepTop(options *option.Options, hopt *HiddifyOptions, candidates []string) int {
	if options.Experimental == nil || options.Experimental.Monitoring == nil {
		return 0 // nothing would test them
	}
	count := hopt.KeepTop
	answering := count == KeepTopAnswering
	var countries []string
	for _, code := range hopt.KeepTopCountries {
		if code = strings.ToUpper(strings.TrimSpace(code)); len(code) == 2 && !slices.Contains(countries, code) {
			countries = append(countries, code)
		}
	}
	slices.Sort(countries)
	if count <= 0 && !answering && len(countries) == 0 {
		return 0
	}
	wanted := make(map[string]bool, len(candidates))
	for _, tag := range candidates {
		wanted[tag] = true
	}
	var managed []*option.Outbound
	for i := range options.Outbounds {
		if out := &options.Outbounds[i]; wanted[out.Tag] && lazyTypes[out.Type] {
			managed = append(managed, out)
		}
	}
	if len(managed) == 0 {
		return 0
	}
	for _, out := range managed {
		out.Lazy = true
	}
	options.Experimental.Monitoring.KeepTop = &option.KeepTopOptions{
		Count:     max(count, 0),
		Answering: answering,
		Countries: countries,
		Batch:     hopt.KeepTopBatch,
		Quick:     hopt.KeepTopQuick,
		Pinned:    hopt.KeepTopPinned,
	}
	return len(managed)
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
