package ezytel

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// frontProvider fetches Telegram public-channel pages and images through
// some outage-resilient front, hiding the concrete mechanism (currently
// Google Translate domain fronting) from EzytelService. Implementing this
// lets a future provider (e.g. a different CDN or translation service) be
// swapped in without touching the HTML-parsing code.
type frontProvider interface {
	// FetchPage retries transient timeouts internally, mirroring the
	// original PHP curl_auto(). params is the t.me path plus any query
	// string, e.g. "channelname" or "channelname?before=123".
	FetchPage(ctx context.Context, params string) (string, error)
	// Download fetches src (an "https://..." URL) into cacheDir/name,
	// no-op if the file already exists there.
	Download(ctx context.Context, cacheDir, src, name string) error
}

// googleTranslateFront is the default frontProvider: it fronts requests
// through the "translate.goog" domain so they appear as Google Translate
// traffic rather than direct connections to Telegram.
type googleTranslateFront struct {
	client *http.Client
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"

var googleDomains = []string{
	"safebrowsing.google.com",
	"images.google.com",
	"maps.google.com",
	"news.google.com",
	"scholar.google.com",
	"meet.google.com",
	"mail.google.com",
	"drive.google.com",
}

// FetchPage retries up to twice on timeout, like the PHP curl_auto().
func (g *googleTranslateFront) FetchPage(ctx context.Context, params string) (string, error) {
	body, err := g.curlGet(ctx, params, 0)
	if err == nil {
		return body, nil
	}
	for try := 1; try <= 2; try++ {
		if !isTimeout(err) {
			break
		}
		body, err = g.curlGet(ctx, params, try)
		if err == nil {
			return body, nil
		}
	}
	return "", err
}

func (g *googleTranslateFront) curlGet(ctx context.Context, params string, try int) (string, error) {
	host := domainPick(try > 1)
	sep := "?"
	if strings.Contains(params, "?") {
		sep = "&"
	}
	dst := fmt.Sprintf("https://%s/s/%s%s_x_tr_sl=el&_x_tr_tl=en&_x_tr_hl=en&_x_tr_pto=wapp", host, params, sep)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dst, nil)
	if err != nil {
		return "", err
	}
	req.Host = "t-me.translate.goog"
	req.Header.Set("Host", "t-me.translate.goog")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Download mirrors the image-download fronting in libs.php: rewrite the
// host into <host-with-dashes>.translate.goog and fetch via
// www.google.com so the SNI stays Google.
func (g *googleTranslateFront) Download(ctx context.Context, cacheDir, src, name string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(cacheDir, name)
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	parsed, err := url.Parse(src)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid url: %s", src)
	}
	host := parsed.Host
	dst := strings.Replace(src, host, "www.google.com", 1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dst, nil)
	if err != nil {
		return err
	}
	frontHost := strings.ReplaceAll(host, ".", "-") + ".translate.goog"
	req.Host = frontHost
	req.Header.Set("Host", frontHost)
	req.Header.Set("User-Agent", userAgent)

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline")
}

func domainPick(rnd bool) string {
	if !rnd {
		return "www.google.com"
	}
	return googleDomains[randInt(len(googleDomains))]
}

var rng = struct {
	mu sync.Mutex
	r  *rand.Rand
}{r: rand.New(rand.NewSource(time.Now().UnixNano()))}

func randInt(n int) int {
	rng.mu.Lock()
	defer rng.mu.Unlock()
	return rng.r.Intn(n)
}
