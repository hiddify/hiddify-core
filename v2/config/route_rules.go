package config

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
	"google.golang.org/protobuf/encoding/protojson"
)

// RouteRuleOptions is the app's rule list ("route-rule"), sent as the proto3 JSON of RouteRule.
type RouteRuleOptions struct {
	*RouteRule
}

func (r *RouteRuleOptions) UnmarshalJSON(content []byte) error {
	routeRule := &RouteRule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(content, routeRule); err != nil {
		return fmt.Errorf("route-rule: %w", err)
	}
	r.RouteRule = routeRule
	return nil
}

func (r RouteRuleOptions) MarshalJSON() ([]byte, error) {
	if r.RouteRule == nil {
		return []byte("null"), nil
	}
	return protojson.Marshal(r.RouteRule)
}

// appRouteRules turns the app's enabled rules, in their order, into route rules, DNS rules and the
// remote rule-sets they use. A rule-set URL may hold "{region}"; it is dropped while the region is
// unset or "other", and a rule left without any condition is dropped too (it would match everything).
type appRouteRules struct {
	region   string
	fakeDNS  bool
	strategy option.DomainStrategy

	routeRules []option.Rule
	dnsRules   []option.DefaultDNSRule
	ruleSets   []option.RuleSet
	tagByURL   map[string]string
	usedTags   map[string]bool
}

func buildAppRouteRules(hopt *HiddifyOptions) *appRouteRules {
	b := &appRouteRules{
		region:   hopt.Region,
		fakeDNS:  hopt.EnableFakeDNS,
		strategy: hopt.DirectDnsDomainStrategy,
		tagByURL: map[string]string{},
		usedTags: map[string]bool{},
	}
	if hopt.RouteRule == nil || hopt.RouteRule.RouteRule == nil {
		return b
	}
	for _, rule := range sortedRules(hopt.RouteRule.GetRules()) {
		if rule.GetEnabled() {
			b.add(rule)
		}
	}
	return b
}

func sortedRules(rules []*Rule) []*Rule {
	sorted := append([]*Rule(nil), rules...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].GetListOrder() < sorted[j-1].GetListOrder(); j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return sorted
}

func (b *appRouteRules) add(rule *Rule) {
	var ruleSets, dnsRuleSets []string
	for _, url := range rule.GetRuleSets() {
		tag, ok := b.ruleSet(url)
		if !ok {
			continue
		}
		ruleSets = append(ruleSets, tag)
		// IP rule-sets mean nothing to a DNS query
		if !strings.HasPrefix(tag, "geoip-") {
			dnsRuleSets = append(dnsRuleSets, tag)
		}
	}
	ports, portRanges := splitPorts(rule.GetPortRanges())
	sourcePorts, sourcePortRanges := splitPorts(rule.GetSourcePortRanges())
	raw := option.RawDefaultRule{
		Domain:          rule.GetDomains(),
		DomainSuffix:    rule.GetDomainSuffixes(),
		DomainKeyword:   rule.GetDomainKeywords(),
		DomainRegex:     rule.GetDomainRegexes(),
		IPCIDR:          rule.GetIpCidrs(),
		SourceIPCIDR:    rule.GetSourceIpCidrs(),
		Port:            ports,
		PortRange:       portRanges,
		SourcePort:      sourcePorts,
		SourcePortRange: sourcePortRanges,
		ProcessName:     rule.GetProcessNames(),
		ProcessPath:     rule.GetProcessPaths(),
		PackageName:     rule.GetPackageNames(),
		RuleSet:         ruleSets,
	}
	if rule.GetNetwork() != Network_all {
		raw.Network = []string{rule.GetNetwork().String()}
	}
	for _, protocol := range rule.GetProtocols() {
		raw.Protocol = append(raw.Protocol, protocol.String())
	}
	// a rule whose only destination was a dropped rule-set would otherwise match every destination
	if len(rule.GetRuleSets()) > 0 && len(ruleSets) == 0 &&
		len(raw.Domain)+len(raw.DomainSuffix)+len(raw.DomainKeyword)+len(raw.DomainRegex)+len(raw.IPCIDR) == 0 {
		return
	}
	if !hasCondition(raw) {
		return
	}

	action := option.RuleAction{Action: C.RuleActionTypeRoute}
	switch rule.GetOutbound() {
	case Outbound_direct:
		action.RouteOptions.Outbound = OutboundDirectTag
	case Outbound_direct_with_fragment:
		action.RouteOptions.Outbound = OutboundDirectFragmentTag
	case Outbound_block:
		action = option.RuleAction{
			Action:        C.RuleActionTypeReject,
			RejectOptions: option.RejectActionOptions{Method: C.RuleActionRejectMethodDefault},
		}
	default:
		action.RouteOptions.Outbound = OutboundMainDetour
	}
	b.routeRules = append(b.routeRules, option.Rule{
		Type:           C.RuleTypeDefault,
		DefaultOptions: option.DefaultRule{RawDefaultRule: raw, RuleAction: action},
	})

	b.addDNSRule(rule, raw, dnsRuleSets)
}

// addDNSRule resolves the names a rule sends direct with the direct DNS (and refuses the ones it
// blocks), so a direct site isn't looked up through the proxy. Only rules that match on names alone
// get one: a port, network, protocol or IP condition can't be checked on a DNS query.
func (b *appRouteRules) addDNSRule(rule *Rule, raw option.RawDefaultRule, dnsRuleSets []string) {
	if len(raw.Network) > 0 || len(raw.Protocol) > 0 || len(raw.Port) > 0 || len(raw.PortRange) > 0 ||
		len(raw.SourcePort) > 0 || len(raw.SourcePortRange) > 0 || len(raw.IPCIDR) > 0 || len(raw.SourceIPCIDR) > 0 {
		return
	}
	dnsRaw := option.RawDefaultDNSRule{
		Domain:        raw.Domain,
		DomainSuffix:  raw.DomainSuffix,
		DomainKeyword: raw.DomainKeyword,
		DomainRegex:   raw.DomainRegex,
		ProcessName:   raw.ProcessName,
		ProcessPath:   raw.ProcessPath,
		PackageName:   raw.PackageName,
		RuleSet:       dnsRuleSets,
	}
	// a rule-set of IPs only leaves nothing to match on
	if len(raw.RuleSet) > 0 && len(dnsRuleSets) == 0 {
		return
	}
	if len(dnsRaw.Domain)+len(dnsRaw.DomainSuffix)+len(dnsRaw.DomainKeyword)+len(dnsRaw.DomainRegex)+
		len(dnsRaw.ProcessName)+len(dnsRaw.ProcessPath)+len(dnsRaw.PackageName)+len(dnsRaw.RuleSet) == 0 {
		return
	}
	var action option.DNSRuleAction
	switch rule.GetOutbound() {
	case Outbound_direct, Outbound_direct_with_fragment:
		action = option.DNSRuleAction{
			Action: C.RuleActionTypeRoute,
			RouteOptions: option.DNSRouteActionOptions{
				Server: DNSMultiDirectTag,
				AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
					Strategy:   b.strategy,
					RewriteTTL: &DEFAULT_DNS_TTL,
				},
			},
		}
	case Outbound_block:
		rcode := option.DNSRCode(5) // REFUSED
		action = option.DNSRuleAction{
			Action:            C.RuleActionTypePredefined,
			PredefinedOptions: option.DNSRouteActionPredefined{Rcode: &rcode},
		}
	default:
		// the remote DNS is already the default; a proxy rule only needs its own DNS rule to win
		// over a later direct one. With fake DNS the fake server must still answer, so leave it.
		if b.fakeDNS {
			return
		}
		action = option.DNSRuleAction{
			Action:       C.RuleActionTypeRoute,
			RouteOptions: option.DNSRouteActionOptions{Server: DNSMultiRemoteTag},
		}
	}
	b.dnsRules = append(b.dnsRules, option.DefaultDNSRule{RawDefaultDNSRule: dnsRaw, DNSRuleAction: action})
}

// ruleSet returns the tag of the remote rule-set at url, adding it once. The tag is the file name
// without its extension (e.g. geosite-ir), so the rule-set's cache survives a rebuild.
func (b *appRouteRules) ruleSet(url string) (string, bool) {
	url = strings.TrimSpace(url)
	if strings.Contains(url, "{region}") {
		if b.region == "" || b.region == "other" {
			return "", false
		}
		url = strings.ReplaceAll(url, "{region}", b.region)
	}
	if url == "" {
		return "", false
	}
	if tag, ok := b.tagByURL[url]; ok {
		return tag, true
	}
	name := path.Base(strings.SplitN(strings.SplitN(url, "?", 2)[0], "#", 2)[0])
	format := C.RuleSetFormatBinary
	switch ext := path.Ext(name); ext {
	case ".json":
		format = C.RuleSetFormatSource
		name = strings.TrimSuffix(name, ext)
	case ".srs":
		name = strings.TrimSuffix(name, ext)
	}
	if name == "" || name == "." || name == "/" {
		name = "rule-set"
	}
	tag := name
	for i := 2; b.usedTags[tag]; i++ {
		tag = name + "-" + strconv.Itoa(i)
	}
	b.usedTags[tag] = true
	b.tagByURL[url] = tag
	b.ruleSets = append(b.ruleSets, option.RuleSet{
		Type:   C.RuleSetTypeRemote,
		Tag:    badoption.Listable[string]{tag},
		Format: format,
		RemoteOptions: option.RemoteRuleSet{
			URL:            url,
			UpdateInterval: badoption.Duration(5 * time.Hour * 24),
			DownloadDetour: OutboundSelectTag,
		},
	})
	return tag, true
}

// splitPorts separates single ports ("443") from ranges ("1000:2000").
func splitPorts(values []string) (ports badoption.Listable[uint16], ranges badoption.Listable[string]) {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if strings.Contains(value, ":") {
			ranges = append(ranges, value)
		} else if port, err := strconv.ParseUint(value, 10, 16); err == nil {
			ports = append(ports, uint16(port))
		}
	}
	return
}

func hasCondition(raw option.RawDefaultRule) bool {
	return len(raw.Domain)+len(raw.DomainSuffix)+len(raw.DomainKeyword)+len(raw.DomainRegex)+
		len(raw.IPCIDR)+len(raw.SourceIPCIDR)+len(raw.Port)+len(raw.PortRange)+len(raw.SourcePort)+
		len(raw.SourcePortRange)+len(raw.ProcessName)+len(raw.ProcessPath)+len(raw.PackageName)+
		len(raw.RuleSet)+len(raw.Network)+len(raw.Protocol) > 0
}
