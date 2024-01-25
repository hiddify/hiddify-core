package config

import (
	"time"

	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

type ConfigOptions struct {
	ExecuteAsIs             bool                  `json:"execute-config-as-is"`
	LogLevel                string                `json:"log-level"`
	ResolveDestination      bool                  `json:"resolve-destination"`
	IPv6Mode                option.DomainStrategy `json:"ipv6-mode"`
	RemoteDnsAddress        string                `json:"remote-dns-address"`
	RemoteDnsDomainStrategy option.DomainStrategy `json:"remote-dns-domain-strategy"`
	DirectDnsAddress        string                `json:"direct-dns-address"`
	DirectDnsDomainStrategy option.DomainStrategy `json:"direct-dns-domain-strategy"`
	MixedPort               uint16                `json:"mixed-port"`
	LocalDnsPort            uint16                `json:"local-dns-port"`
	MTU                     uint32                `json:"mtu"`
	StrictRoute             bool                  `json:"strict-route"`
	TUNStack                string                `json:"tun-stack"`
	ConnectionTestUrl       string                `json:"connection-test-url"`
	URLTestInterval         option.Duration       `json:"url-test-interval"`
	URLTestIdleTimeout      option.Duration       `json:"url-test-idle-timeout"`
	EnableClashApi          bool                  `json:"enable-clash-api"`
	ClashApiPort            uint16                `json:"clash-api-port"`
	EnableTun               bool                  `json:"enable-tun"`
	SetSystemProxy          bool                  `json:"set-system-proxy"`
	BypassLAN               bool                  `json:"bypass-lan"`
	AllowConnectionFromLAN  bool                  `json:"allow-connection-from-lan"`
	EnableFakeDNS           bool                  `json:"enable-fake-dns"`
	EnableDNSRouting        bool                  `json:"enable-dns-routing"`
	IndependentDNSCache     bool                  `json:"independent-dns-cache"`
	GeoIPPath               string                `json:"geoip-path"`
	GeoSitePath             string                `json:"geosite-path"`
	Rules                   []Rule                `json:"rules"`
	MuxOptions
	TLSTricks
}

type TLSTricks struct {
	EnableFragment     bool   `json:"enable-tls-fragment"`
	FragmentSize       string `json:"tls-fragment-size"`
	FragmentSleep      string `json:"tls-fragment-sleep"`
	EnableMixedSNICase bool   `json:"enable-tls-mixed-sni-case"`
	EnablePadding      bool   `json:"enable-tls-padding"`
	PaddingSize        string `json:"tls-padding-size"`
}

type MuxOptions struct {
	EnableMux   bool   `json:"enable-mux"`
	MuxPadding  bool   `json:"mux-padding"`
	MaxStreams  int    `json:"mux-max-streams"`
	MuxProtocol string `json:"mux-protocol"`
}

func DefaultConfigOptions() *ConfigOptions {
	return &ConfigOptions{
		ExecuteAsIs:             false,
		LogLevel:                "info",
		ResolveDestination:      false,
		IPv6Mode:                option.DomainStrategy(dns.DomainStrategyAsIS),
		RemoteDnsAddress:        "1.1.1.1",
		RemoteDnsDomainStrategy: option.DomainStrategy(dns.DomainStrategyAsIS),
		DirectDnsAddress:        "1.1.1.1",
		DirectDnsDomainStrategy: option.DomainStrategy(dns.DomainStrategyAsIS),
		MixedPort:               2334,
		LocalDnsPort:            6450,
		MTU:                     9000,
		StrictRoute:             true,
		TUNStack:                "mixed",
		ConnectionTestUrl:       "http://cp.cloudflare.com/",
		URLTestInterval:         option.Duration(10 * time.Minute),
		URLTestIdleTimeout:      option.Duration(100 * time.Minute),
		EnableClashApi:          true,
		ClashApiPort:            6756,
		EnableTun:               true,
		SetSystemProxy:          true,
		BypassLAN:               false,
		AllowConnectionFromLAN:  false,
		EnableFakeDNS:           false,
		EnableDNSRouting:        false,
		IndependentDNSCache:     false,
		GeoIPPath:               "geoip.db",
		GeoSitePath:             "geosite.db",
		Rules:                   []Rule{},
		MuxOptions: MuxOptions{
			EnableMux:   true,
			MuxPadding:  true,
			MaxStreams:  8,
			MuxProtocol: "h2mux",
		},
		TLSTricks: TLSTricks{
			EnableFragment:     false,
			FragmentSize:       "10-100",
			FragmentSleep:      "50-200",
			EnableMixedSNICase: false,
			EnablePadding:      false,
			PaddingSize:        "100-200",
		},
	}
}
