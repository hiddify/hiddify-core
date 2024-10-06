package profile

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/hiddifyoptions"
)

const (
	infiniteTrafficThreshold = 9223372036854775807
	infiniteTimeThreshold    = 92233720368
)

// ProfileParser parses profile subscription URLs and headers for data
type ProfileParser struct{}

// Parse parses a URL and headers to create a RemoteProfileEntity
func (p *ProfileEntity) Parse(headers http.Header) {
	name := ""

	// Check profile-title header
	if titleHeaders, ok := headers["Profile-Title"]; ok && len(titleHeaders) > 0 {
		titleHeader := titleHeaders[0]
		if strings.HasPrefix(titleHeader, "base64:") {
			decodedTitle, _ := base64.StdEncoding.DecodeString(strings.Replace(titleHeader, "base64:", "", 1))
			name = string(decodedTitle)
		} else {
			name = strings.TrimSpace(titleHeader)
		}
	}

	// Check content-disposition header
	if contentDispositionHeaders, ok := headers["Content-Disposition"]; ok && len(contentDispositionHeaders) > 0 && name == "" {
		contentDispositionHeader := contentDispositionHeaders[0]
		re := regexp.MustCompile(`filename="([^"]*)"`)
		if match := re.FindStringSubmatch(contentDispositionHeader); len(match) > 1 {
			name = match[1]
		}
	}

	// Check URL fragment
	parsedURL, err := url.Parse(p.Url)
	if err == nil {
		if name == "" {
			name = parsedURL.Fragment
		}

		// Check URL file extension
		urlParts := strings.Split(parsedURL.Path, "/")
		if len(urlParts) > 0 && name == "" {
			lastPart := urlParts[len(urlParts)-1]
			re := regexp.MustCompile(`\.(json|yaml|yml|txt)[\s\S]*`)
			name = re.ReplaceAllString(lastPart, "")
		}
		if name == "" {
			name = parsedURL.Host
		}
	}
	if name == "" {
		name = "Remote Profile"
	}

	var options *ProfileOptions
	if updateIntervalHeaders, ok := headers["Profile-Update-Interval"]; ok && len(updateIntervalHeaders) > 0 {
		updateInterval, _ := time.ParseDuration(updateIntervalHeaders[0] + "h")
		options = &ProfileOptions{UpdateInterval: updateInterval.Milliseconds()}
	}

	var subInfo *SubscriptionInfo
	if subInfoHeaders, ok := headers["Subscription-Userinfo"]; ok && len(subInfoHeaders) > 0 {
		subInfo = parseSubscriptionInfo(subInfoHeaders[0])
	}

	if subInfo != nil {
		if profileWebPageURLHeaders, ok := headers["Profile-Web-Page-Url"]; ok && len(profileWebPageURLHeaders) > 0 && isURL(profileWebPageURLHeaders[0]) {
			subInfo.WebPageUrl = profileWebPageURLHeaders[0]
		}
		if profileSupportURLHeaders, ok := headers["Support-Url"]; ok && len(profileSupportURLHeaders) > 0 && isURL(profileSupportURLHeaders[0]) {
			subInfo.SupportUrl = profileSupportURLHeaders[0]
		}
	}

	if p.Name == "" {
		p.Name = name
	}

	p.Options = options
	p.SubInfo = subInfo
	p.OverrideHiddifyOptions = hiddifyoptions.GetOverridableHiddifyOptions(headers)
}

// parseSubscriptionInfo parses subscription info from a string
func parseSubscriptionInfo(subInfoStr string) *SubscriptionInfo {
	values := strings.Split(subInfoStr, ";")
	infoMap := map[string]int64{}
	for _, v := range values {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			num, _ := parseInt(value)
			infoMap[key] = num
		}
	}

	upload := infoMap["upload"]
	download := infoMap["download"]
	total := infoMap["total"]
	expire := infoMap["expire"]

	if total == 0 {
		total = infiniteTrafficThreshold
	}
	if expire == 0 {
		expire = infiniteTimeThreshold
	}

	return &SubscriptionInfo{
		Upload:   upload,
		Download: download,
		Total:    total,
		Expire:   expire,
	}
}

// isURL checks if a string is a valid URL
func isURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	return err == nil
}

// parseInt parses an integer from a string
func parseInt(s string) (int64, error) {
	return json.Number(s).Int64()
}

// safeDecodeBase64 decodes a base64-encoded string. Returns the decoded string.
func safeDecodeBase64(content string) string {
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return content
	}
	return string(decoded)
}

// parseHeadersFromContent parses headers from a given base64-encoded content string.
func parseHeadersFromContent(content string) http.Header {
	headers := make(map[string][]string)

	// Decode base64 content
	contentDecoded := safeDecodeBase64(content)

	lines := strings.SplitN(contentDecoded, "\n", 30)
	for i := 0; i < len(lines)-1; i++ {
		line := lines[i]
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			index := strings.Index(line, ":")
			if index == -1 {
				continue
			}
			if len(line) <= index+1 || line[index+1] == '/' {
				continue
			}
			key := strings.TrimSpace(strings.ToLower(strings.TrimPrefix(line[:index], "#")))
			key = strings.TrimSpace(strings.TrimPrefix(key, "//"))
			value := strings.TrimSpace(line[index+1:])

			if value != "" {
				headers[key] = []string{value}
			}
		}
	}

	return http.Header(headers)
}
