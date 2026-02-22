package config

import (
	"fmt"
	reflect "reflect"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
	dns "github.com/sagernet/sing-dns"
)

type HiddifyOptions struct {
	EnableFullConfig        bool   `json:"enable-full-config,omitempty" overridable:"true"`
	LogLevel                string `json:"log-level,omitempty"`
	LogFile                 string `json:"log-file,omitempty"`
	EnableClashApi          bool   `json:"enable-clash-api,omitempty"`
	ClashApiPort            uint16 `json:"clash-api-port,omitempty"`
	ClashApiSecret          string `json:"web-secret,omitempty"`
	Region                  string `json:"region,omitempty"`
	BlockAds                bool   `json:"block-ads,omitempty" overridable:"true"`
	UseXrayCoreWhenPossible bool   `json:"use-xray-core-when-possible,omitempty" overridable:"true"`
	BalancerStrategy        string `json:"balancer-strategy,omitempty" overridable:"true"`
	// GeoIPPath        string      `json:"geoip-path"`
	// GeoSitePath      string      `json:"geosite-path"`
	Rules     []Rule      `json:"rules,omitempty" overridable:"true"`
	Warp      WarpOptions `json:"warp,omitempty"`
	Warp2     WarpOptions `json:"warp2,omitempty"`
	Mux       MuxOptions  `json:"mux,omitempty" overridable:"true"`
	TLSTricks TLSTricks   `json:"tls-tricks,omitempty"`
	EnableNTP bool        `json:"enable-ntp,omitempty"`

	DNSOptions
	InboundOptions
	URLTestOptions
	RouteOptions
}

type DNSOptions struct {
	RemoteDnsAddress        string                `json:"remote-dns-address,omitempty" overridable:"true"`
	RemoteDnsDomainStrategy option.DomainStrategy `json:"remote-dns-domain-strategy,omitempty" overridable:"true"`
	DirectDnsAddress        string                `json:"direct-dns-address,omitempty" overridable:"true"`
	DirectDnsDomainStrategy option.DomainStrategy `json:"direct-dns-domain-strategy,omitempty" overridable:"true"`
	IndependentDNSCache     bool                  `json:"independent-dns-cache,omitempty"`
	EnableFakeDNS           bool                  `json:"enable-fake-dns,omitempty"`
	// EnableDNSRouting        bool                  `json:"enable-dns-routing,omitempty"`
}

type InboundOptions struct {
	EnableTun        bool   `json:"enable-tun,omitempty"`
	EnableTunService bool   `json:"enable-tun-service,omitempty"`
	SetSystemProxy   bool   `json:"set-system-proxy,omitempty"`
	MixedPort        uint16 `json:"mixed-port,omitempty"`
	TProxyPort       uint16 `json:"tproxy-port,omitempty"`
	RedirectPort     uint16 `json:"redirect-port,omitempty"`
	DirectPort       uint16 `json:"direct-port,omitempty"`
	MTU              uint32 `json:"mtu,omitempty"`
	StrictRoute      bool   `json:"strict-route,omitempty"`
	TUNStack         string `json:"tun-implementation,omitempty"`
}

type URLTestOptions struct {
	ConnectionTestUrl  string            `json:"connection-test-url,omitempty" overridable:"true"`
	ConnectionTestUrls []string          `json:"connection-test-urls,omitempty" overridable:"true"`
	URLTestInterval    DurationInSeconds `json:"url-test-interval,omitempty" overridable:"true"`
	// URLTestIdleTimeout DurationInSeconds `json:"url-test-idle-timeout"`
}

type RouteOptions struct {
	ResolveDestination     bool                  `json:"resolve-destination,omitempty"`
	IPv6Mode               option.DomainStrategy `json:"ipv6-mode,omitempty"`
	BypassLAN              bool                  `json:"bypass-lan,omitempty"`
	AllowConnectionFromLAN bool                  `json:"allow-connection-from-lan,omitempty"`
	BlockQuic              bool                  `json:"block-quic,omitempty"`
}

type TLSTricks struct {
	EnableFragment bool   `json:"enable-fragment,omitempty" overridable:"true"`
	FragmentSize   string `json:"fragment-size,omitempty" overridable:"true"`
	FragmentSleep  string `json:"fragment-sleep,omitempty" overridable:"true"`
	MixedSNICase   bool   `json:"mixed-sni-case,omitempty" overridable:"true"`
	EnablePadding  bool   `json:"enable-padding,omitempty" overridable:"true"`
	PaddingSize    string `json:"padding-size,omitempty" overridable:"true"`
}

type MuxOptions struct {
	Enable     bool   `json:"enable,omitempty" overridable:"true"`
	Padding    bool   `json:"padding,omitempty" overridable:"true"`
	MaxStreams int    `json:"max-streams,omitempty" overridable:"true"`
	Protocol   string `json:"protocol,omitempty" overridable:"true"`
}

type WarpOptions struct {
	Id                 string              `json:"id,omitempty"`
	EnableWarp         bool                `json:"enable,omitempty"`
	Mode               string              `json:"mode,omitempty"`
	WireguardConfigStr string              `json:"wireguard-config,omitempty"`
	WireguardConfig    WarpWireguardConfig `json:"wireguardConfig,omitempty"` // TODO check
	FakePackets        string              `json:"noise,omitempty"`
	FakePacketSize     string              `json:"noise-size,omitempty"`
	FakePacketDelay    string              `json:"noise-delay,omitempty"`
	FakePacketMode     string              `json:"noise-mode,omitempty"`
	CleanIP            string              `json:"clean-ip,omitempty"`
	CleanPort          uint16              `json:"clean-port,omitempty"`
	Account            WarpAccount
}

func DefaultHiddifyOptions() *HiddifyOptions {
	return &HiddifyOptions{
		EnableNTP: true,
		DNSOptions: DNSOptions{
			RemoteDnsAddress:        "1.1.1.1",
			RemoteDnsDomainStrategy: option.DomainStrategy(dns.DomainStrategyAsIS),
			DirectDnsAddress:        "1.1.1.1",
			DirectDnsDomainStrategy: option.DomainStrategy(dns.DomainStrategyAsIS),
			IndependentDNSCache:     false,
			EnableFakeDNS:           false,
			// EnableDNSRouting:        false,
		},
		InboundOptions: InboundOptions{
			EnableTun:      false,
			SetSystemProxy: false,
			MixedPort:      12334,
			TProxyPort:     12335,
			RedirectPort:   12336,
			DirectPort:     12337,
			MTU:            9000,
			StrictRoute:    true,
			TUNStack:       "mixed",
		},
		URLTestOptions: URLTestOptions{
			ConnectionTestUrl: "http://cp.cloudflare.com/",
			URLTestInterval:   DurationInSeconds(600),
			// URLTestIdleTimeout: DurationInSeconds(6000),
		},
		RouteOptions: RouteOptions{
			ResolveDestination:     false,
			IPv6Mode:               option.DomainStrategy(dns.DomainStrategyAsIS),
			BypassLAN:              false,
			AllowConnectionFromLAN: false,
		},
		LogLevel: "warn",
		// LogFile:        "/dev/null",
		LogFile:        "data/box.log",
		Region:         "other",
		EnableClashApi: true,

		ClashApiPort:   16756,
		ClashApiSecret: "",
		// GeoIPPath:      "geoip.db",
		// GeoSitePath:    "geosite.db",
		Rules: []Rule{},
		Mux: MuxOptions{
			Enable:     false,
			Padding:    true,
			MaxStreams: 8,
			Protocol:   "h2mux",
		},
		TLSTricks: TLSTricks{
			EnableFragment: false,
			FragmentSize:   "10-100",
			FragmentSleep:  "50-200",
			MixedSNICase:   false,
			EnablePadding:  false,
			PaddingSize:    "1200-1500",
		},
		UseXrayCoreWhenPossible: false,
	}
}

// Recursively set the fields marked as overridable
func setOverridableFields(v reflect.Value, t reflect.Type, overrides map[string]interface{}) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Check if the field has an "overridable" tag set to "true"
		overridableTag := fieldType.Tag.Get("overridable")
		if overridableTag == "true" {
			// Get the field's JSON tag name
			jsonTag := strings.Split(fieldType.Tag.Get("json"), ",")[0]
			if jsonTag == "" {
				continue
			}

			// Check if an override exists for this field
			if overrideValue, ok := overrides[jsonTag]; ok {
				// Ensure the override value can be set to the field type
				var parsedValue reflect.Value
				switch field.Kind() {
				case reflect.Bool:
					if boolVal, err := parseBool(overrideValue); err == nil {
						parsedValue = reflect.ValueOf(boolVal)
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if intVal, err := parseInt(overrideValue); err == nil {
						parsedValue = reflect.ValueOf(intVal)
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					if uintVal, err := parseUint(overrideValue); err == nil {
						parsedValue = reflect.ValueOf(uintVal)
					}
				case reflect.String:
					parsedValue = reflect.ValueOf(overrideValue.(string))
					// Add more cases for other types as needed
				}

				// Set the field if we have a parsed value
				if parsedValue.IsValid() && parsedValue.Type().AssignableTo(field.Type()) {
					field.Set(parsedValue)
				}
			}
		}

		// If the field is a nested struct, recurse into it
		if field.Kind() == reflect.Struct {
			jsonTag := strings.Split(fieldType.Tag.Get("json"), ",")[0]

			data := overrides
			if jsonTag != "" {
				data1 := overrides[jsonTag]
				if data1 == nil {
					continue
				}
				data = data1.(map[string]interface{})
			}
			neastedType := fieldType.Type
			if data != nil {
				setOverridableFields(field, neastedType, data)
			}

		}
	}
}

// Helper functions for parsing
func parseBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseBool(v)
	case bool:
		return v, nil
	}
	return false, fmt.Errorf("invalid bool value")
}

func parseInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseInt(v, 10, 64)
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int(), nil
	}
	return 0, fmt.Errorf("invalid int value")
}

func parseUint(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseUint(v, 10, 64)
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint(), nil
	}
	return 0, fmt.Errorf("invalid uint value")
}

func GetOverridableHiddifyOptions(overrides map[string][]string) *HiddifyOptions {
	overrideHiddify := HiddifyOptions{}

	// Convert flat overrides to nested structure
	nestedOverrides := convertFlatToNested(overrides)

	// Use reflection to iterate over the fields of HiddifyOptions
	v := reflect.ValueOf(&overrideHiddify).Elem()
	t := reflect.TypeOf(overrideHiddify)

	// Recursively set the fields that are marked as overridable
	setOverridableFields(v, t, nestedOverrides)

	return &overrideHiddify
}

// Converts the flat overrides map to a nested structure without removing underscores
func convertFlatToNested(overrides map[string][]string) map[string]interface{} {
	nested := make(map[string]interface{})
	for key, value := range overrides {
		keys := strings.Split(key, ".")
		current := nested

		for i, k := range keys {
			if i == len(keys)-1 {
				// Set the final value with underscores preserved
				current[k] = value[0]
			} else {
				// Create nested maps if they do not exist
				if _, exists := current[k]; !exists {
					current[k] = make(map[string]interface{})
				}
				current = current[k].(map[string]interface{})
			}
		}
	}
	return nested
}
