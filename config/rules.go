package config

import (
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
)

type Rule struct {
	RuleSetUrl string `json:"rule-set-url"`
	Domains    string `json:"domains"`
	IP         string `json:"ip"`
	Port       string `json:"port"`
	Network    string `json:"network"`
	Protocol   string `json:"protocol"`
	Outbound   string `json:"outbound"`
}

func (r *Rule) MakeRule() option.DefaultRule {
	rule := option.DefaultRule{}
	if len(r.Domains) > 0 {
		rule = makeDomainRule(rule, strings.Split(r.Domains, ","))
	}
	if len(r.IP) > 0 {
		rule = makeIpRule(rule, strings.Split(r.IP, ","))
	}
	if len(r.Port) > 0 {
		rule = makePortRule(rule, strings.Split(r.Port, ","))
	}
	if len(r.Network) > 0 {
		rule.Network = append(rule.Network, r.Network)
	}
	if len(r.Protocol) > 0 {
		rule.Protocol = append(rule.Protocol, strings.Split(r.Protocol, ",")...)
	}
	return rule
}

func (r *Rule) MakeDNSRule() option.DefaultDNSRule {
	rule := option.DefaultDNSRule{}
	domains := strings.Split(r.Domains, ",")
	for _, item := range domains {
		if strings.HasPrefix(item, "geosite:") {
			rule.Geosite = append(rule.Geosite, strings.TrimPrefix(item, "geosite:"))
		} else if strings.HasPrefix(item, "full:") {
			rule.Domain = append(rule.Domain, strings.ToLower(strings.TrimPrefix(item, "full:")))
		} else if strings.HasPrefix(item, "domain:") {
			rule.DomainSuffix = append(rule.DomainSuffix, strings.ToLower(strings.TrimPrefix(item, "domain:")))
		} else if strings.HasPrefix(item, "regexp:") {
			rule.DomainRegex = append(rule.DomainRegex, strings.ToLower(strings.TrimPrefix(item, "regexp:")))
		} else if strings.HasPrefix(item, "keyword:") {
			rule.DomainKeyword = append(rule.DomainKeyword, strings.ToLower(strings.TrimPrefix(item, "keyword:")))
		}
	}
	return rule
}

func makeDomainRule(options option.DefaultRule, list []string) option.DefaultRule {
	for _, item := range list {
		if strings.HasPrefix(item, "geosite:") {
			options.Geosite = append(options.Geosite, strings.TrimPrefix(item, "geosite:"))
		} else if strings.HasPrefix(item, "full:") {
			options.Domain = append(options.Domain, strings.ToLower(strings.TrimPrefix(item, "full:")))
		} else if strings.HasPrefix(item, "domain:") {
			options.DomainSuffix = append(options.DomainSuffix, strings.ToLower(strings.TrimPrefix(item, "domain:")))
		} else if strings.HasPrefix(item, "regexp:") {
			options.DomainRegex = append(options.DomainRegex, strings.ToLower(strings.TrimPrefix(item, "regexp:")))
		} else if strings.HasPrefix(item, "keyword:") {
			options.DomainKeyword = append(options.DomainKeyword, strings.ToLower(strings.TrimPrefix(item, "keyword:")))
		}
	}
	return options
}

func makeIpRule(options option.DefaultRule, list []string) option.DefaultRule {
	for _, item := range list {
		if strings.HasPrefix(item, "geoip:") {
			options.GeoIP = append(options.GeoIP, strings.TrimPrefix(item, "geoip:"))
		} else {
			options.IPCIDR = append(options.IPCIDR, item)
		}
	}
	return options
}

func makePortRule(options option.DefaultRule, list []string) option.DefaultRule {
	for _, item := range list {
		if strings.Contains(item, ":") {
			options.PortRange = append(options.PortRange, item)
		} else if i, err := strconv.Atoi(item); err == nil {
			options.Port = append(options.Port, uint16(i))
		}
	}
	return options
}
