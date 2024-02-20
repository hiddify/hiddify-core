package config

import (
	"github.com/bepass-org/wireguard-go/warp"
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

type ConfigOptions struct {
	LogLevel       string `json:"log-level"`
	EnableClashApi bool   `json:"enable-clash-api"`
	ClashApiPort   uint16 `json:"clash-api-port"`
	GeoIPPath      string `json:"geoip-path"`
	GeoSitePath    string `json:"geosite-path"`
	Rules          []Rule `json:"rules"`
	DNSOptions
	InboundOptions
	URLTestOptions
	RouteOptions
	MuxOptions
	TLSTricks
	*WarpOptions
}

type DNSOptions struct {
	RemoteDnsAddress        string                `json:"remote-dns-address"`
	RemoteDnsDomainStrategy option.DomainStrategy `json:"remote-dns-domain-strategy"`
	DirectDnsAddress        string                `json:"direct-dns-address"`
	DirectDnsDomainStrategy option.DomainStrategy `json:"direct-dns-domain-strategy"`
	IndependentDNSCache     bool                  `json:"independent-dns-cache"`
	EnableFakeDNS           bool                  `json:"enable-fake-dns"`
	EnableDNSRouting        bool                  `json:"enable-dns-routing"`
}

type InboundOptions struct {
	EnableTun        bool   `json:"enable-tun"`
	EnableTunService bool   `json:"enable-tun-service"`
	SetSystemProxy   bool   `json:"set-system-proxy"`
	MixedPort        uint16 `json:"mixed-port"`
	LocalDnsPort     uint16 `json:"local-dns-port"`
	MTU              uint32 `json:"mtu"`
	StrictRoute      bool   `json:"strict-route"`
	TUNStack         string `json:"tun-stack"`
}

type URLTestOptions struct {
	ConnectionTestUrl  string            `json:"connection-test-url"`
	URLTestInterval    DurationInSeconds `json:"url-test-interval"`
	URLTestIdleTimeout DurationInSeconds `json:"url-test-idle-timeout"`
}

type RouteOptions struct {
	ResolveDestination     bool                  `json:"resolve-destination"`
	IPv6Mode               option.DomainStrategy `json:"ipv6-mode"`
	BypassLAN              bool                  `json:"bypass-lan"`
	AllowConnectionFromLAN bool                  `json:"allow-connection-from-lan"`
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

type WarpOptions struct {
	Mode string `json:"mode"`
	WarpAccount
	warp.WireguardConfig
	FakePackets     string `json:"fake-packets"`
	FakePacketSize  string `json:"fake-packet-size"`
	FakePacketDelay string `json:"fake-packet-delay"`
	CleanIP         string `json:"clean-ip"`
	CleanPort       uint16 `json:"clean-port"`
}

func DefaultConfigOptions() *ConfigOptions {
	return &ConfigOptions{
		DNSOptions: DNSOptions{
			RemoteDnsAddress:        "1.1.1.1",
			RemoteDnsDomainStrategy: option.DomainStrategy(dns.DomainStrategyAsIS),
			DirectDnsAddress:        "1.1.1.1",
			DirectDnsDomainStrategy: option.DomainStrategy(dns.DomainStrategyAsIS),
			IndependentDNSCache:     false,
			EnableFakeDNS:           false,
			EnableDNSRouting:        false,
		},
		InboundOptions: InboundOptions{
			EnableTun:      true,
			SetSystemProxy: true,
			MixedPort:      2334,
			LocalDnsPort:   16450,
			MTU:            9000,
			StrictRoute:    true,
			TUNStack:       "mixed",
		},
		URLTestOptions: URLTestOptions{
			ConnectionTestUrl:  "http://cp.cloudflare.com/",
			URLTestInterval:    DurationInSeconds(600),
			URLTestIdleTimeout: DurationInSeconds(6000),
		},
		RouteOptions: RouteOptions{
			ResolveDestination:     false,
			IPv6Mode:               option.DomainStrategy(dns.DomainStrategyAsIS),
			BypassLAN:              false,
			AllowConnectionFromLAN: false,
		},
		LogLevel:       "info",
		EnableClashApi: true,
		ClashApiPort:   16756,
		GeoIPPath:      "geoip.db",
		GeoSitePath:    "geosite.db",
		Rules:          []Rule{},
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
			PaddingSize:        "1200-1500",
		},
	}
}
