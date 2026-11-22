package config

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

const proxyFilterProfile = `{"outbounds":[
{"type":"vless","tag":"🇩🇪 Germany [1]","server":"192.0.2.1","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"},
{"type":"vless","tag":"🇳🇱 Netherlands","server":"192.0.2.2","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"},
{"type":"vless","tag":"🇩🇪 Германия","server":"192.0.2.3","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"}]}`

// buildFiltered builds profile with the app's rules, after setup changes the settings, and returns
// the config, after sing-box has checked it.
func buildFiltered(t *testing.T, profile string, setup func(*HiddifyOptions), rules ...*Rule) *option.Options {
	t.Helper()
	hopts := DefaultHiddifyOptions()
	if setup != nil {
		setup(hopts)
	}
	for i, rule := range rules {
		rule.ListOrder, rule.Enabled = uint32(i), true
	}
	hopts.RouteRule = &RouteRuleOptions{RouteRule: &RouteRule{Rules: rules}}
	options, err := BuildConfig(libbox.BaseContext(nil), hopts, &ReadOptions{Content: profile})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejects the config: %v", err)
	}
	return options
}

func buildWithRules(t *testing.T, rules ...*Rule) *option.Options {
	t.Helper()
	return buildFiltered(t, proxyFilterProfile, nil, rules...)
}

// routeRule returns the route rule for domain suffix, the IPv6 reject of a filtered rule aside.
func routeRule(t *testing.T, options *option.Options, suffix string) option.DefaultRule {
	t.Helper()
	for _, rule := range options.Route.Rules {
		if rule.DefaultOptions.IPVersion == 0 && slices.Equal(rule.DefaultOptions.DomainSuffix, []string{suffix}) {
			return rule.DefaultOptions
		}
	}
	t.Fatalf("no route rule for %s", suffix)
	return option.DefaultRule{}
}

func ruleOutbound(t *testing.T, options *option.Options, suffix string) string {
	t.Helper()
	return routeRule(t, options, suffix).RouteOptions.Outbound
}

// balancerMembers returns the members of the balancer tag, or nil when tag is no balancer.
func balancerMembers(options *option.Options, tag string) []string {
	for _, out := range options.Outbounds {
		if out.Tag == tag && out.Type == C.TypeBalancer {
			return out.Options.(*option.BalancerOutboundOptions).Outbounds
		}
	}
	return nil
}

// filterBalancers counts the balancers of the proxy filters.
func filterBalancers(options *option.Options) int {
	n := 0
	for _, out := range options.Outbounds {
		if out.Type == C.TypeBalancer && out.Tag != OutboundURLTestTag && out.Tag != OutboundRoundRobinTag {
			n++
		}
	}
	return n
}

func proxyRule(suffix, filter string) *Rule {
	return &Rule{Outbound: Outbound_proxy, DomainSuffixes: []string{suffix}, ProxyFilter: filter}
}

func TestProxyFilterPicksTheMatchingProxies(t *testing.T) {
	options := buildWithRules(t, proxyRule("claude.ai", " 🇩🇪 "))
	tag := ruleOutbound(t, options, "claude.ai")
	want := []string{"🇩🇪 Germany [1]", "🇩🇪 Германия"}
	if got := balancerMembers(options, tag); !reflect.DeepEqual(got, want) {
		t.Fatalf("rule goes to %q with %v, want a balancer of %v", tag, got, want)
	}
}

func TestProxyFilterIgnoresCase(t *testing.T) {
	options := buildWithRules(t, proxyRule("claude.ai", "GERMANY|нидерл"))
	want := []string{"🇩🇪 Germany [1]"}
	if got := balancerMembers(options, ruleOutbound(t, options, "claude.ai")); !reflect.DeepEqual(got, want) {
		t.Fatalf("members %v, want %v", got, want)
	}
}

// Repeated names are numbered ("x § 2") and shown without the number: the filter sees them so.
func TestProxyFilterMatchesTheShownName(t *testing.T) {
	profile := `{"outbounds":[
{"type":"vless","tag":"Germany","server":"192.0.2.1","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"},
{"type":"vless","tag":"Germany § 2","server":"192.0.2.2","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"},
{"type":"vless","tag":"Germany West","server":"192.0.2.3","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"}]}`
	options := buildFiltered(t, profile, nil, proxyRule("claude.ai", "^germany$"))
	want := []string{"Germany", "Germany § 2"}
	if got := balancerMembers(options, ruleOutbound(t, options, "claude.ai")); !reflect.DeepEqual(got, want) {
		t.Fatalf("members %v, want %v", got, want)
	}
}

func TestRulesWithOneFilterShareABalancer(t *testing.T) {
	options := buildWithRules(t, proxyRule("claude.ai", "🇩🇪"), proxyRule("anthropic.com", "🇩🇪"),
		proxyRule("example.nl", "🇳🇱"))
	de, de2, nl := ruleOutbound(t, options, "claude.ai"), ruleOutbound(t, options, "anthropic.com"),
		ruleOutbound(t, options, "example.nl")
	if de != de2 || de == nl {
		t.Fatalf("outbounds %q, %q, %q: want the first two the same and the third another", de, de2, nl)
	}
	if got := balancerMembers(options, nl); !reflect.DeepEqual(got, []string{"🇳🇱 Netherlands"}) {
		t.Fatalf("members of %q: %v", nl, got)
	}
}

func TestProxyFilterFallsBackToTheMainProxy(t *testing.T) {
	for name, filter := range map[string]string{
		"no filter":                  "",
		"matching no proxy":          "🇺🇸",
		"not a valid regexp":         "(🇩🇪",
		"only spaces":                "   ",
		"anchored at a name's start": "^Germany", // the names start with a flag
	} {
		t.Run(name, func(t *testing.T) {
			options := buildWithRules(t, proxyRule("claude.ai", filter))
			if got := ruleOutbound(t, options, "claude.ai"); got != OutboundSelectTag {
				t.Fatalf("rule goes to %q, want %q", got, OutboundSelectTag)
			}
			if n := filterBalancers(options); n != 0 {
				t.Fatalf("%d extra balancers", n)
			}
		})
	}
}

func TestProxyFilterOnAProfileOfOneProxy(t *testing.T) {
	profile := `{"outbounds":[{"type":"vless","tag":"🇩🇪 Germany","server":"192.0.2.1","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"}]}`
	options := buildFiltered(t, profile, nil, proxyRule("claude.ai", "🇩🇪"))
	if got := balancerMembers(options, ruleOutbound(t, options, "claude.ai")); !reflect.DeepEqual(got, []string{"🇩🇪 Germany"}) {
		t.Fatalf("members %v", got)
	}
}

func TestProxyFilterLeavesDirectAndBlockRules(t *testing.T) {
	options := buildWithRules(t,
		&Rule{Outbound: Outbound_direct, DomainSuffixes: []string{"example.ru"}, ProxyFilter: "🇩🇪"},
		&Rule{Outbound: Outbound_direct_with_fragment, DomainSuffixes: []string{"example.ir"}, ProxyFilter: "🇩🇪"},
		&Rule{Outbound: Outbound_block, DomainSuffixes: []string{"ads.example"}, ProxyFilter: "🇩🇪"})
	if got := ruleOutbound(t, options, "example.ru"); got != OutboundDirectTag {
		t.Fatalf("direct rule goes to %q", got)
	}
	if got := ruleOutbound(t, options, "example.ir"); got != OutboundDirectFragmentTag {
		t.Fatalf("direct with fragment rule goes to %q", got)
	}
	if got := routeRule(t, options, "ads.example").Action; got != C.RuleActionTypeReject {
		t.Fatalf("block rule action %q", got)
	}
	if n := filterBalancers(options); n != 0 {
		t.Fatalf("%d balancers for rules that don't proxy", n)
	}
}

// A profile proxy may carry the tag the core would give a filter's balancer.
func TestProxyFilterAvoidsATakenTag(t *testing.T) {
	profile := `{"outbounds":[
{"type":"vless","tag":"🇩🇪 Germany","server":"192.0.2.1","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"},
{"type":"vless","tag":"rule-proxies-1 §hide§","server":"192.0.2.2","server_port":443,"uuid":"bf000d23-0752-40b4-affe-68f7707a9661"}]}`
	options := buildFiltered(t, profile, nil, proxyRule("claude.ai", "🇩🇪"))
	if tag := ruleOutbound(t, options, "claude.ai"); tag == "rule-proxies-1 §hide§" || balancerMembers(options, tag) == nil {
		t.Fatalf("rule goes to %q", tag)
	}
}

func TestExtraSecurityChainKeepsTheMainProxy(t *testing.T) {
	options := buildFiltered(t, proxyFilterProfile, func(h *HiddifyOptions) {
		h.ChainStatus = ChainStatusExtraSecurity
		h.ExtraSecurity.Mode = ChainModePsiphon
	}, proxyRule("claude.ai", "🇩🇪"))
	hop := ChainHopTag(ChainStatusExtraSecurity, ChainModePsiphon)
	if got := ruleOutbound(t, options, "claude.ai"); got != hop {
		t.Fatalf("rule goes to %q, want the chain hop %q", got, hop)
	}
	if n := filterBalancers(options); n != 0 {
		t.Fatalf("%d balancers around the chain hop", n)
	}
}

// With the unblocker every proxy dials through the hop, the filter's members too.
func TestUnblockerChainKeepsTheFilter(t *testing.T) {
	options := buildFiltered(t, proxyFilterProfile, func(h *HiddifyOptions) {
		h.ChainStatus = ChainStatusUnblocker
		h.Unblocker.Mode = ChainModePsiphon
	}, proxyRule("claude.ai", "🇩🇪"))
	members := balancerMembers(options, ruleOutbound(t, options, "claude.ai"))
	if len(members) != 2 {
		t.Fatalf("members %v", members)
	}
	hop := ChainHopTag(ChainStatusUnblocker, ChainModePsiphon)
	for _, out := range options.Outbounds {
		if slices.Contains(members, out.Tag) {
			if detour := out.Options.(*option.VLESSOutboundOptions).Detour; detour != hop {
				t.Fatalf("%q dials through %q, want the hop %q", out.Tag, detour, hop)
			}
		}
	}
}

// With IPv6 for the apps, an IPv6 connection of a filtered rule is refused when the filter's
// proxies cannot carry it, before the rule routes it.
func TestProxyFilterRefusesIPv6ItsProxiesLack(t *testing.T) {
	options := buildFiltered(t, proxyFilterProfile, func(h *HiddifyOptions) {
		h.IPv6Mode = option.DomainStrategy(C.DomainStrategyPreferIPv4)
	}, proxyRule("claude.ai", "🇩🇪"))
	group := ruleOutbound(t, options, "claude.ai")
	for i, rule := range options.Route.Rules {
		r := rule.DefaultOptions
		if !slices.Equal(r.DomainSuffix, []string{"claude.ai"}) {
			continue
		}
		if r.IPVersion != 6 || r.IPv6UnavailableVia != group || r.Action != C.RuleActionTypeReject {
			t.Fatalf("first claude.ai rule (#%d) is %+v, want the IPv6 reject via %q", i, r, group)
		}
		return
	}
	t.Fatal("no claude.ai rule")
}

func TestProxyFilterAddsNoIPv6RuleWithoutIPv6(t *testing.T) {
	options := buildWithRules(t, proxyRule("claude.ai", "🇩🇪"))
	for _, rule := range options.Route.Rules {
		if rule.DefaultOptions.IPv6UnavailableVia != "" && slices.Equal(rule.DefaultOptions.DomainSuffix, []string{"claude.ai"}) {
			t.Fatal("an IPv6 rule while the apps get no IPv6 (the main reject refuses all of it)")
		}
	}
}

// With keep top, the filter's proxies are pinned: at rest none of them would answer.
func TestKeepTopPinsTheFiltersProxies(t *testing.T) {
	var hopts *HiddifyOptions
	options := buildFiltered(t, proxyFilterProfile, func(h *HiddifyOptions) {
		h.KeepTop, h.KeepTopPinned = 1, []string{"🇳🇱 Netherlands"}
		hopts = h
	}, proxyRule("claude.ai", "🇩🇪"))
	if ApplyKeepTop(options, hopts, selectableProxies(options)) == 0 {
		t.Fatal("keep top is off")
	}
	want := []string{"🇳🇱 Netherlands", "🇩🇪 Germany [1]", "🇩🇪 Германия"}
	if got := []string(options.Experimental.Monitoring.KeepTop.Pinned); !reflect.DeepEqual(got, want) {
		t.Fatalf("pinned %v, want %v", got, want)
	}
}

func TestProxyFilterReadsTheAppsJSON(t *testing.T) {
	var hopts HiddifyOptions
	content := `{"route-rule":{"rules":[{"enabled":true,"outbound":"proxy","domain_suffix":["claude.ai"],"proxy_filter":"🇩🇪"}]}}`
	if err := json.Unmarshal([]byte(content), &hopts); err != nil {
		t.Fatal(err)
	}
	if got := hopts.RouteRule.GetRules()[0].GetProxyFilter(); got != "🇩🇪" {
		t.Fatalf("proxy_filter %q", got)
	}
}
