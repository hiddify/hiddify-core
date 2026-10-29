package config

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	E "github.com/sagernet/sing/common/exceptions"
)

// A profile can list other subscriptions, one URL per line (Mahsa's profiles do): their content
// takes the place of the URL. Only one level is read, so a subscription cannot list itself.

const (
	subscriptionLinksMax     = 32
	subscriptionFetchTimeout = 20 * time.Second
	subscriptionSizeMax      = 16 << 20
)

// isSubscriptionLink reports a line that is the URL of a subscription, not an HTTP proxy: http or
// https with a path, and without a user or a name (fragment), which proxy links have.
func isSubscriptionLink(line string) bool {
	if !strings.HasPrefix(line, "http://") && !strings.HasPrefix(line, "https://") || strings.ContainsAny(line, " \t") {
		return false
	}
	u, err := url.Parse(line)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return false
	}
	return strings.Trim(u.Path, "/") != ""
}

// expandSubscriptionLinks replaces the subscription URLs of content with their content. A
// subscription that cannot be read is left out; content without subscription URLs is returned
// as is.
func expandSubscriptionLinks(ctx context.Context, content []byte) []byte {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	var links []int
	for i, line := range lines {
		if isSubscriptionLink(strings.TrimSpace(line)) && len(links) < subscriptionLinksMax {
			links = append(links, i)
		}
	}
	if len(links) == 0 {
		return content
	}
	var wait sync.WaitGroup
	for _, i := range links {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			body, err := fetchSubscription(ctx, strings.TrimSpace(lines[i]))
			if err != nil {
				body = ""
			}
			lines[i] = body
		}(i)
	}
	wait.Wait()
	return []byte(strings.Join(lines, "\n"))
}

func fetchSubscription(ctx context.Context, link string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, subscriptionFetchTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "HiddifyNext/4.0.0("+runtime.GOOS+") like ClashMeta v2ray sing-box")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", E.New("subscription ", link, ": ", response.Status)
	}
	// a base64 subscription is one line, which the share link parser decodes
	body, err := io.ReadAll(io.LimitReader(response.Body, subscriptionSizeMax))
	return strings.TrimSpace(string(body)), err
}
