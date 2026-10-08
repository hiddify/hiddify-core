package config

import (
	"encoding/json"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

const geoURL = "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set"

// the rule list as the app sends it (proto3 JSON: default enums such as proxy/all are left out)
const appRouteRule = `"route-rule":{"rules":[
	{"list_order":0,"enabled":true,"name":"builtin:bypassLan","outbound":"direct","ip_cidr":["10.0.0.0/8","192.168.0.0/16"]},
	{"list_order":1,"enabled":false,"name":"builtin:blockAds","outbound":"block","rule_set":["` + geoURL + `/block/geosite-category-ads-all.srs"]},
	{"list_order":2,"enabled":true,"name":"builtin:bypassRegion","outbound":"direct","rule_set":["` + geoURL + `/country/geosite-{region}.srs","` + geoURL + `/country/geoip-{region}.srs"]},
	{"list_order":3,"enabled":true,"name":"ports","outbound":"block","network":"udp","port_range":["443","1000:2000"]},
	{"list_order":4,"enabled":true,"name":"sites","domain_suffix":["example.com"],"package_name":["org.example"]}
]}`

func routeOf(t *testing.T, options *option.Options, name string) []option.Rule {
	t.Helper()
	var found []option.Rule
	for _, rule := range options.Route.Rules {
		raw := rule.DefaultOptions.RawDefaultRule
		switch name {
		case "lan":
			if len(raw.IPCIDR) == 2 && raw.IPCIDR[0] == "10.0.0.0/8" {
				found = append(found, rule)
			}
		case "region":
			if strings.Join(raw.RuleSet, ",") == "geosite-ir,geoip-ir" {
				found = append(found, rule)
			}
		case "ports":
			if len(raw.PortRange) > 0 {
				found = append(found, rule)
			}
		case "sites":
			if len(raw.DomainSuffix) == 1 && raw.DomainSuffix[0] == "example.com" {
				found = append(found, rule)
			}
		}
	}
	return found
}

func ruleSetTags(options *option.Options) []string {
	var tags []string
	for _, rs := range options.Route.RuleSet {
		tags = append(tags, rs.Tag...)
	}
	return tags
}

func TestAppRouteRules(t *testing.T) {
	options := buildChainConfig(t, `{"region":"ir","chain-status":"off",`+appRouteRule+`}`)

	if r := routeOf(t, options, "lan"); len(r) != 1 || r[0].DefaultOptions.RouteOptions.Outbound != OutboundDirectTag {
		t.Fatalf("bypass LAN rule: %+v", r)
	}
	region := routeOf(t, options, "region")
	if len(region) != 1 || region[0].DefaultOptions.RouteOptions.Outbound != OutboundDirectTag {
		t.Fatalf("region rule: %+v", region)
	}
	ports := routeOf(t, options, "ports")
	if len(ports) != 1 || ports[0].DefaultOptions.Action != C.RuleActionTypeReject {
		t.Fatalf("ports rule: %+v", ports)
	}
	raw := ports[0].DefaultOptions.RawDefaultRule
	if len(raw.Port) != 1 || raw.Port[0] != 443 || raw.PortRange[0] != "1000:2000" || raw.Network[0] != "udp" {
		t.Fatalf("ports rule conditions: %+v", raw)
	}
	sites := routeOf(t, options, "sites")
	if len(sites) != 1 || sites[0].DefaultOptions.RouteOptions.Outbound != OutboundMainDetour {
		t.Fatalf("a rule without an outbound goes through the proxy: %+v", sites)
	}

	// only the enabled rules' rule-sets, with the region filled in
	if got := strings.Join(ruleSetTags(options), ","); got != "geosite-ir,geoip-ir" {
		t.Fatalf("rule-sets: %s", got)
	}
	// the region's names resolve with the direct DNS, but not with the IP-only rule-set
	foundDNS := false
	for _, rule := range options.DNS.Rules {
		rs := rule.DefaultOptions.RuleSet
		if strings.Join(rs, ",") == "geosite-ir" {
			foundDNS = rule.DefaultOptions.RouteOptions.Server == DNSMultiDirectTag
		}
		for _, tag := range rs {
			if strings.HasPrefix(tag, "geoip-") {
				t.Fatalf("a DNS rule uses the IP rule-set %s", tag)
			}
		}
	}
	if !foundDNS {
		t.Fatal("no direct DNS rule for the region")
	}
}

func TestAppRouteRulesReplaceLegacyOptions(t *testing.T) {
	// the app's list decides: its region rule is off, so the legacy region rules must not come back
	settings := `{"region":"ir","block-ads":true,"bypass-lan":true,"chain-status":"off","route-rule":{"rules":[
		{"list_order":0,"enabled":false,"name":"builtin:bypassRegion","outbound":"direct","rule_set":["` + geoURL + `/country/geoip-{region}.srs"]}]}}`
	options := buildChainConfig(t, settings)
	if tags := ruleSetTags(options); len(tags) != 0 {
		t.Fatalf("rule-sets added beside the app's list: %v", tags)
	}
	for _, rule := range options.Route.Rules {
		raw := rule.DefaultOptions.RawDefaultRule
		if raw.IPIsPrivate || len(raw.DomainSuffix) > 0 && raw.DomainSuffix[0] == ".ir" {
			t.Fatalf("legacy rule added: %+v", raw)
		}
	}
}

func TestAppRouteRulesRegionOther(t *testing.T) {
	options := buildChainConfig(t, `{"region":"other","chain-status":"off",`+appRouteRule+`}`)
	if r := routeOf(t, options, "region"); len(r) != 0 {
		t.Fatal("the region rule must be dropped with the region other")
	}
	for _, rule := range options.Route.Rules {
		if rule.DefaultOptions.RouteOptions.Outbound == OutboundDirectTag && !hasCondition(rule.DefaultOptions.RawDefaultRule) {
			t.Fatal("a rule without conditions sends everything direct")
		}
	}
	if tags := ruleSetTags(options); len(tags) != 0 {
		t.Fatalf("rule-sets: %v", tags)
	}
}

// without route-rule (CLI, older apps) the legacy options still apply
func TestLegacyRoutingWithoutAppRules(t *testing.T) {
	options := buildChainConfig(t, `{"region":"ir","chain-status":"off"}`)
	if got := strings.Join(ruleSetTags(options), ","); got != "geoip-ir,geosite-ir" {
		t.Fatalf("legacy region rule-sets: %s", got)
	}
}

func TestRouteRuleOptionsJSON(t *testing.T) {
	opts := chainTestOptions(t, `{`+appRouteRule+`}`)
	if n := len(opts.RouteRule.GetRules()); n != 5 {
		t.Fatalf("decoded %d rules", n)
	}
	if opts.RouteRule.GetRules()[4].GetOutbound() != Outbound_proxy {
		t.Fatal("a missing outbound must decode as proxy")
	}
	out, err := json.Marshal(opts)
	if err != nil {
		t.Fatal(err)
	}
	again := chainTestOptions(t, string(out))
	if len(again.RouteRule.GetRules()) != 5 {
		t.Fatal("route-rule did not survive a round trip")
	}
}
