package config

import (
	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

type HiddifyOptions struct {
	EnableFullConfig        bool   `json:"enable-full-config"`
	LogLevel                string `json:"log-level"`
	LogFile                 string `json:"log-file"`
	EnableClashApi          bool   `json:"enable-clash-api"`
	ClashApiPort            uint16 `json:"clash-api-port"`
	ClashApiSecret          string `json:"web-secret"`
	ClashApiHost            string `json:"clash-api-host"`
	Region                  string `json:"region"`
	BlockAds                bool   `json:"block-ads"`
	UseXrayCoreWhenPossible bool   `json:"use-xray-core-when-possible"`
	// GeoIPPath        string      `json:"geoip-path"`
	// GeoSitePath      string      `json:"geosite-path"`
	Rules     []Rule      `json:"rules"`
	Warp      WarpOptions `json:"warp"`
	Warp2     WarpOptions `json:"warp2"`
	Mux       MuxOptions  `json:"mux"`
	TLSTricks TLSTricks   `json:"tls-tricks"`
	GeoRulesBaseURL        string   `json:"geo-rules-base-url"`
	DNSOptions
	InboundOptions
	URLTestOptions
	RouteOptions
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
    TProxyPort       uint16 `json:"tproxy-port"`
    LocalDnsPort     uint16 `json:"local-dns-port"`
    MTU              uint32 `json:"mtu"`
    StrictRoute      bool   `json:"strict-route"`
    TUNStack         string `json:"tun-implementation"`
}

type URLTestOptions struct {
    ConnectionTestUrl string            `json:"connection-test-url"`
    URLTestInterval   DurationInSeconds `json:"url-test-interval"`
    // URLTestIdleTimeout DurationInSeconds `json:"url-test-idle-timeout"`
}

type RouteOptions struct {
    ResolveDestination     bool                  `json:"resolve-destination"`
    IPv6Mode               option.DomainStrategy `json:"ipv6-mode"`
    BypassLAN              bool                  `json:"bypass-lan"`
    AllowConnectionFromLAN bool                  `json:"allow-connection-from-lan"`
    ExtraBypassCIDRs       []string              `json:"extra-bypass-cidrs"`
}

type TLSTricks struct {
    EnableFragment   bool   `json:"enable-fragment"`
    FragmentSize     string `json:"fragment-size"`
    FragmentSleep    string `json:"fragment-sleep"`
    MixedSNICase     bool   `json:"mixed-sni-case"`
    EnablePadding    bool   `json:"enable-padding"`
    PaddingSize      string `json:"padding-size"`
    EnableECH        bool   `json:"enable-ech"`
    ECHConfig        string `json:"ech-config"`
    ECHConfigPath    string `json:"ech-config-path"`
    EnableReality    bool   `json:"enable-reality"`
    RealityPublicKey string `json:"reality-public-key"`
    RealityShortID   string `json:"reality-short-id"`
}

type MuxOptions struct {
    Enable     bool   `json:"enable"`
    Padding    bool   `json:"padding"`
    MaxStreams int    `json:"max-streams"`
    Protocol   string `json:"protocol"`
}

type WarpOptions struct {
    Id                 string              `json:"id"`
    EnableWarp         bool                `json:"enable"`
    Mode               string              `json:"mode"`
    WireguardConfigStr string              `json:"wireguard-config"`
    WireguardConfig    WarpWireguardConfig `json:"wireguardConfig"`
    FakePackets        string              `json:"noise"`
    FakePacketSize     string              `json:"noise-size"`
    FakePacketDelay    string              `json:"noise-delay"`
    FakePacketMode     string              `json:"noise-mode"`
    CleanIP            string              `json:"clean-ip"`
    CleanPort          uint16              `json:"clean-port"`
    Account            WarpAccount         `json:"-"`
}

func DefaultHiddifyOptions() *HiddifyOptions {
    return &HiddifyOptions{
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
            EnableTun:      false,
            SetSystemProxy: false,
            MixedPort:      12334,
            TProxyPort:     12335,
            LocalDnsPort:   16450,
            MTU:            9000,
            StrictRoute:    true,
            TUNStack:       "mixed",
        },
        URLTestOptions: URLTestOptions{
            ConnectionTestUrl: "http://cp.cloudflare.com/",
            URLTestInterval:   DurationInSeconds(600),
        },
        RouteOptions: RouteOptions{
            ResolveDestination:     false,
            IPv6Mode:               option.DomainStrategy(dns.DomainStrategyAsIS),
            BypassLAN:              false,
            AllowConnectionFromLAN: false,
            ExtraBypassCIDRs:       []string{},
        },
        LogLevel: "warn",
        LogFile:  "box.log",
        Region:   "other",
        EnableClashApi: true,
        ClashApiPort:   16756,
        ClashApiSecret: "",
        ClashApiHost:   "127.0.0.1",
        GeoRulesBaseURL: "https://raw.githubusercontent.com/hiddify/hiddify-geo",
        Rules: []Rule{},
        Mux: MuxOptions{
            Enable:     false,
            Padding:    true,
            MaxStreams: 8,
            Protocol:   "h2mux",
        },
        TLSTricks: TLSTricks{
            EnableFragment:  false,
            FragmentSize:    "10-100",
            FragmentSleep:   "50-200",
            MixedSNICase:    false,
            EnablePadding:   false,
            PaddingSize:     "1200-1500",
            EnableECH:       false,
            ECHConfig:       "",
            ECHConfigPath:   "",
            EnableReality:   false,
            RealityPublicKey: "",
            RealityShortID:   "",
        },
        UseXrayCoreWhenPossible: false,
    }
}
