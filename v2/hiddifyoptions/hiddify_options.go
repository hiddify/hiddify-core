package hiddifyoptions

import (
	"fmt"
	reflect "reflect"
	"strconv"
	"strings"
)

func DefaultHiddifyOptions() *HiddifyOptions {
	return &HiddifyOptions{
		DnsOptions: &DNSOptions{
			RemoteDnsAddress:        "1.1.1.1",
			RemoteDnsDomainStrategy: DomainStrategy_as_is,
			DirectDnsAddress:        "1.1.1.1",
			DirectDnsDomainStrategy: DomainStrategy_as_is,
			IndependentDnsCache:     false,
			EnableFakeDns:           false,
			EnableDnsRouting:        false,
		},
		InboundOptions: &InboundOptions{
			EnableTun:      false,
			SetSystemProxy: false,
			MixedPort:      12334,
			TproxyPort:     12335,
			LocalDnsPort:   16450,
			Mtu:            9000,
			StrictRoute:    true,
			TunStack:       "mixed",
		},
		UrlTestOptions: &URLTestOptions{
			ConnectionTestUrl: "http://cp.cloudflare.com/",
			UrlTestInterval:   600 * 1000,
			// URLTestIdleTimeout: DurationInSeconds(6000),
		},
		RouteOptions: &RouteOptions{
			ResolveDestination:     false,
			Ipv6Mode:               DomainStrategy_as_is,
			BypassLan:              false,
			AllowConnectionFromLan: false,
		},
		LogLevel: "warn",
		// LogFile:        "/dev/null",
		LogFile:        "data/box.log",
		Region:         "other",
		EnableClashApi: true,
		ClashApiPort:   16756,
		WebSecret:      "",
		// GeoIPPath:      "geoip.db",
		// GeoSitePath:    "geosite.db",

		TlsTricks: &TLSTricks{
			EnableFragment: false,
			FragmentSize:   &IntRange{From: 10, To: 100},
			FragmentSleep:  &IntRange{From: 50, To: 200},
			MixedSniCase:   false,
			EnablePadding:  false,
			PaddingSize:    &IntRange{From: 1200, To: 1500},
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
