package config

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
)

// Diagnostics leave the device unencrypted, so the configuration in them keeps its structure
// (types, tags, ports, routes, DNS and TLS settings) and loses what identifies the user or
// their servers: addresses, names, users and secrets.

// sensitiveConfigKey matches the keys whose values are replaced. A key that ends in _port is
// kept: a port says nothing without its server.
var sensitiveConfigKey = regexp.MustCompile(`(?i)(^|_)(server|host|hosts|sni|address|addresses|password|passwd|secret|token|auth|auth_str|user|users|username|uuid|key|keys|cert|certificate|path|short_id|obfs|reserved|cookie|license|dns_stamp|url)($|_)`)

const redactedConfigValue = "<redacted>"

// SanitizeConfigJSON returns content with the values of sensitive keys replaced, at any depth,
// by "<redacted>". Invalid JSON comes back as nothing: it is safer to leave the configuration
// out than to share it as it is.
func SanitizeConfigJSON(content []byte) []byte {
	var root any
	if err := json.Unmarshal(content, &root); err != nil {
		return nil
	}
	sanitizeConfigValue(root)
	var result bytes.Buffer
	encoder := json.NewEncoder(&result)
	encoder.SetEscapeHTML(false) // "<redacted>" as written, not \u003credacted\u003e
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(root); err != nil {
		return nil
	}
	return result.Bytes()
}

func sanitizeConfigValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isSensitiveConfigKey(key) {
				typed[key] = redactedConfigValue
				continue
			}
			sanitizeConfigValue(child)
		}
	case []any:
		for _, child := range typed {
			sanitizeConfigValue(child)
		}
	}
}

func isSensitiveConfigKey(key string) bool {
	lower := strings.ToLower(key)
	if strings.HasSuffix(lower, "_port") || lower == "listen" || lower == "listen_port" {
		return false
	}
	return sensitiveConfigKey.MatchString(lower)
}
