package ezytel

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EzytelService is a port of the original PHP ezytel app to Go, exposed
// over gRPC. It fetches t.me public-channel pages through the
// translate.goog domain front and rewrites the HTML so the embedded
// images flow through ProxyImage instead of going to telegram.org directly.
type EzytelService struct {
	UnimplementedEzytelServer

	cacheDir string
	client   *http.Client
	mu       sync.Mutex
	infoMu   sync.Mutex
}

// NewEzytelService constructs a service rooted at cacheDir. The directory
// is created on first use; pass "" to fall back to <tempdir>/ezytel-cache.
func NewEzytelService(cacheDir string) *EzytelService {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "ezytel-cache")
	}
	return &EzytelService{
		cacheDir: cacheDir,
		client: &http.Client{
			Timeout: 35 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
	}
}

// GetCacheDir returns the directory where avatars and channel JSON snapshots
// are written. Useful for callers that want to surface AvatarPath as a
// concrete file path.
func (s *EzytelService) GetCacheDir() string { return s.cacheDir }

func (s *EzytelService) ensureCacheDir() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.MkdirAll(s.cacheDir, 0o755)
}

// ---------------------------------------------------------------------------
// RPC: GetChannelMessages — port of html_parse() in libs.php.
// ---------------------------------------------------------------------------

func (s *EzytelService) GetChannelMessages(ctx context.Context, in *ChannelMessagesRequest) (*ChannelMessagesResponse, error) {
	chid := strings.TrimSpace(in.ChannelId)
	if chid == "" {
		return nil, fmt.Errorf("channel_id is required")
	}
	path := chid
	if in.Before != 0 {
		path = fmt.Sprintf("%s?before=%d", chid, in.Before)
	}
	html, err := s.curlAuto(ctx, path)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(strings.ToLower(html), "</header>") || !strings.Contains(strings.ToLower(html), "</main>") {
		return nil, fmt.Errorf("upstream page missing header/main markers")
	}
	// disable telegram internal pics
	html = caseReplace(html, "background-image:url('//", "background-temp:url('//")
	// rewrite background-image:url('https://...')
	html = rewriteBackgroundImages(html)
	// rewrite <img src="https://...">
	html = rewriteImgSources(html)
	// fix date and time blocks
	html = rewriteTimes(html)

	chanPic := strFind(html, []string{"tgme_page_photo_image", "src=\""}, "\"")
	content := strFind(html, []string{"</header>"}, "</main>")
	if content == "" {
		return nil, fmt.Errorf("could not isolate <main> content")
	}
	content += "</main>"

	// rewrite "load older" link
	if strings.Contains(strings.ToLower(content), "messages_more_wrap") {
		link := strFind(content, []string{"messages_more_wrap", ">"}, "</div>")
		data := strFind(content, []string{"messages_more_wrap", "data-before=\""}, "\"")
		if link != "" && data != "" {
			content = caseReplace(content,
				link,
				fmt.Sprintf(`<a class="tme_messages_more" href="#" onclick="load_more(%s,'%s',this)"></a>`, data, chid))
		}
	}
	// drop the "after" link (newer messages) — we always render bottom-up
	if strings.Contains(strings.ToLower(content), "data-after=") {
		raw := strFind(content, []string{"data-after=", "*<a"}, "</a>")
		if raw != "" {
			content = caseReplace(content, "<a "+raw+"</a>", "")
		}
	}

	resp := &ChannelMessagesResponse{ChannelAvatar: chanPic}

	// On the first page, prepend the channel header card and capture last_post_id.
	if in.Before == 0 {
		subscribers := strFind(html, []string{`<div class="tgme_header_counter">`}, "</div>")
		channelName := strFind(html, []string{`<div class="tgme_header_title">`}, "</div>")
		header := fmt.Sprintf(
			`<header class="tgme_header search_collapsed"><div class="menu_channel_header"><div class="menu_channel_avatar"><img src="%s"></div><div class="menu_channel_info"><p>%s</p><span>%s</span></div></div></header>`,
			chanPic, stripTags(channelName), subscribers,
		)
		content = header + content

		lastPostRaw := strFind(html, []string{`*data-post="`, "/"}, "\"")
		if id, err := strconv.ParseInt(strings.TrimSpace(lastPostRaw), 10, 64); err == nil {
			resp.LastPostId = id
		}
	}

	resp.Html = content
	return resp, nil
}

// ---------------------------------------------------------------------------
// RPC: GetChannelInfo — port of json_info() in libs.php.
// ---------------------------------------------------------------------------

func (s *EzytelService) GetChannelInfo(ctx context.Context, in *ChannelInfoRequest) (*ChannelInfo, error) {
	chid := strings.TrimSpace(in.ChannelId)
	if chid == "" {
		return nil, fmt.Errorf("channel_id is required")
	}

	cacheFile := filepath.Join(s.cacheDir, chid+".json")
	html, err := s.curlAuto(ctx, chid)
	ok := err == nil && strings.Contains(html, `<meta property="og:title" content="`)

	if !ok {
		// fall back to last good snapshot
		if data, rerr := os.ReadFile(cacheFile); rerr == nil {
			var snap ChannelInfo
			if jerr := json.Unmarshal(data, &snap); jerr == nil {
				snap.Newmsg = "OFF"
				snap.Ok = false
				return &snap, nil
			}
		}
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("channel page malformed")
	}

	info := &ChannelInfo{
		Name: strFind(html, []string{`<meta property="og:title" content="`}, "\""),
		Ok:   true,
	}

	pic := strFind(html, []string{`<meta property="og:image" content="https://`}, "\"")
	if pic != "" {
		hash := md5sum(pic)
		if err := s.curlDownload(ctx, "https://"+pic, hash+".jpg"); err == nil {
			info.AvatarPath = "cache/" + hash + ".jpg"
		}
	}

	if strings.Contains(html, `data-post="`) {
		lastPost := strFind(html, []string{`*data-post="`}, "")
		lastCode, _ := strconv.ParseInt(strings.TrimSpace(strFind(lastPost, []string{"/"}, "\"")), 10, 64)
		info.LastPostId = lastCode

		if strings.Contains(strings.ToLower(lastPost), "tgme_widget_message_text") {
			info.Description = stripTags(strFind(lastPost, []string{"tgme_widget_message_text", ">"}, "</div>"))
		} else {
			info.Description = "فایل"
		}
		if strings.Contains(strings.ToLower(lastPost), "<time datetime=") {
			t := strFind(lastPost, []string{`<time datetime="`}, "\"")
			if parsed, perr := time.Parse(time.RFC3339, t); perr == nil {
				info.Date = parsed.Unix()
				converted := dateConvert(parsed)
				if len(converted) >= 5 {
					info.DateStr = converted[5:]
				}
			}
		}
		if in.LastRead > 0 && lastCode != in.LastRead {
			diff := lastCode - in.LastRead
			if diff < 0 {
				diff = 0
			}
			if diff > 99 {
				diff = 99
			}
			info.Newmsg = fmt.Sprintf("+%d", diff)
		}
	}

	if err := s.ensureCacheDir(); err == nil {
		if data, jerr := json.Marshal(info); jerr == nil {
			s.infoMu.Lock()
			_ = os.WriteFile(cacheFile, data, 0o644)
			s.infoMu.Unlock()
		}
	}
	return info, nil
}

// ---------------------------------------------------------------------------
// RPC: ProxyImage — port of proxy.php's image branch.
// ---------------------------------------------------------------------------

func (s *EzytelService) ProxyImage(ctx context.Context, in *ProxyImageRequest) (*ProxyImageResponse, error) {
	if in.HexUrl == "" {
		return nil, fmt.Errorf("hex_url is required")
	}
	raw, err := hex.DecodeString(in.HexUrl)
	if err != nil {
		return nil, fmt.Errorf("hex_url is not valid hex: %w", err)
	}
	pic := string(raw)
	hash := md5sum(pic)
	cacheName := hash + ".jpg"

	if err := s.curlDownload(ctx, "https://"+pic, cacheName); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(s.cacheDir, cacheName))
	if err != nil {
		return nil, err
	}
	return &ProxyImageResponse{
		Data:        data,
		ContentType: "image/jpeg",
		CacheName:   cacheName,
	}, nil
}

// ---------------------------------------------------------------------------
// RPC: ParseChannels — port of file_parse() in libs.php.
// ---------------------------------------------------------------------------

func (s *EzytelService) ParseChannels(_ context.Context, in *ParseChannelsRequest) (*ParseChannelsResponse, error) {
	raw := in.Raw
	for _, ch := range []string{"'", "\"", "@"} {
		raw = strings.ReplaceAll(raw, ch, "")
	}
	for _, p := range []string{"https://", "http://", "HTTPS://", "HTTP://"} {
		raw = strings.ReplaceAll(raw, p, "")
	}
	raw = strings.ReplaceAll(raw, "/s/", "")
	raw = strings.ReplaceAll(raw, "t.me", "")
	raw = strings.ReplaceAll(raw, "/", "")

	seen := map[string]struct{}{}
	out := []string{}
	for _, line := range strings.Split(raw, "\n") {
		name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(line), " ", ""))
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return &ParseChannelsResponse{ChannelIds: out}, nil
}

// ---------------------------------------------------------------------------
// HTTP plumbing — domain-fronted GET via translate.goog.
// ---------------------------------------------------------------------------

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

// curlAuto retries up to twice on timeout, like the PHP curl_auto().
func (s *EzytelService) curlAuto(ctx context.Context, params string) (string, error) {
	body, err := s.curlGet(ctx, params, 0)
	if err == nil {
		return body, nil
	}
	for try := 1; try <= 2; try++ {
		if !isTimeout(err) {
			break
		}
		body, err = s.curlGet(ctx, params, try)
		if err == nil {
			return body, nil
		}
	}
	return "", err
}

func (s *EzytelService) curlGet(ctx context.Context, params string, try int) (string, error) {
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

	resp, err := s.client.Do(req)
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

// curlDownload mirrors the image-download fronting in libs.php: rewrite
// the host into <host-with-dashes>.translate.goog and fetch via
// www.google.com so the SNI stays Google.
func (s *EzytelService) curlDownload(ctx context.Context, src, name string) error {
	if err := s.ensureCacheDir(); err != nil {
		return err
	}
	target := filepath.Join(s.cacheDir, name)
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

	resp, err := s.client.Do(req)
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

func domainPick(rand bool) string {
	if !rand {
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

// ---------------------------------------------------------------------------
// String / HTML helpers — ports of str_find, stripTags etc.
// ---------------------------------------------------------------------------

// strFind reproduces the PHP str_find(): walk a sequence of needles forward
// (or backward when prefixed with "*"), then slice up to `end`. When `end`
// is empty the rest of the string is returned. Returns "" on miss.
func strFind(s string, find []string, end string) string {
	lower := strings.ToLower(s)
	start := 0
	for i := 0; i < len(find); i++ {
		needle := find[i]
		var pos int
		if strings.HasPrefix(needle, "*") {
			n := strings.ToLower(needle[1:])
			if start == 0 {
				pos = strings.LastIndex(lower, n)
			} else {
				pos = strings.LastIndex(lower[:start], n)
			}
			if pos < 0 {
				return ""
			}
			start = pos + len(n)
		} else {
			n := strings.ToLower(needle)
			pos = indexFrom(lower, n, start)
			if pos < 0 {
				return ""
			}
			start = pos + len(n)
		}
	}
	if end == "" {
		return strings.TrimSpace(s[start:])
	}
	fin := indexFrom(strings.ToLower(s), strings.ToLower(end), start)
	if fin < 0 {
		return ""
	}
	return strings.TrimSpace(s[start:fin])
}

func indexFrom(s, sub string, from int) int {
	if from < 0 || from > len(s) {
		return -1
	}
	idx := strings.Index(s[from:], sub)
	if idx < 0 {
		return -1
	}
	return from + idx
}

// caseReplace is a case-insensitive str_ireplace().
func caseReplace(s, old, new string) string {
	if old == "" {
		return s
	}
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(old))
	if err != nil {
		return s
	}
	return re.ReplaceAllString(s, regexp.QuoteMeta(new))
}

func md5sum(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string {
	return tagRE.ReplaceAllString(s, "")
}

// rewriteBackgroundImages turns background-image:url('https://X')
// into background-image:url('proxy.php?url=<hex(X)>') — kept as-is so the
// HTML stays drop-in compatible with the original front-end. Clients that
// own the renderer can swap the URL prefix via post-processing.
func rewriteBackgroundImages(html string) string {
	prefix := "background-image:url('https://"
	for {
		idx := strings.Index(strings.ToLower(html), strings.ToLower(prefix))
		if idx < 0 {
			return html
		}
		start := idx + len(prefix)
		end := strings.Index(html[start:], ")")
		if end < 0 {
			return html
		}
		raw := strings.Trim(html[start:start+end], "'\"")
		full := "https://" + raw
		hexed := hex.EncodeToString([]byte(raw))
		html = strings.ReplaceAll(html, full, "proxy.php?url="+hexed)
	}
}

func rewriteImgSources(html string) string {
	prefix := `<img src="https://`
	for {
		idx := strings.Index(html, prefix)
		if idx < 0 {
			return html
		}
		start := idx + len(prefix)
		end := strings.Index(html[start:], `"`)
		if end < 0 {
			return html
		}
		raw := html[start : start+end]
		full := "https://" + raw
		hexed := hex.EncodeToString([]byte(raw))
		html = strings.ReplaceAll(html, full, "proxy.php?url="+hexed)
	}
}

func rewriteTimes(html string) string {
	for {
		i := strings.Index(html, `<time datetime="`)
		if i < 0 {
			return html
		}
		t := strFind(html, []string{`<time datetime="`}, `"`)
		if t == "" {
			return html
		}
		raw := strFind(html, []string{`datetime="`, `*<time `}, "</time>")
		if raw == "" {
			return html
		}
		var human string
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			human = dateEasyRead(parsed)
		} else {
			human = t
		}
		html = strings.Replace(html, raw, `class="time">`+human, 1)
	}
}

// ---------------------------------------------------------------------------
// Persian-calendar helpers — port of date_convert / date_easy_read.
// ---------------------------------------------------------------------------

var tehran *time.Location

func init() {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		// Tehran is UTC+3:30 with no DST since 2022.
		loc = time.FixedZone("IRST", 3*3600+1800)
	}
	tehran = loc
}

func dateEasyRead(t time.Time) string {
	full := dateConvert(t)
	today := dateConvert(time.Now())
	yesterday := dateConvert(time.Now().Add(-24 * time.Hour))
	if len(today) >= 10 && strings.Contains(full, today[:10]) {
		return strings.Replace(full, today[:10], "Today", 1)
	}
	if len(yesterday) >= 10 && strings.Contains(full, yesterday[:10]) {
		return strings.Replace(full, yesterday[:10], "Yesterday", 1)
	}
	return full
}

// dateConvert returns the Tehran-local Jalali (Persian) date plus HH:mm,
// matching the PHP version verbatim so cached snapshots stay comparable.
func dateConvert(t time.Time) string {
	dt := t.In(tehran)
	gy := dt.Year()
	gm := int(dt.Month())
	gd := dt.Day()
	hhmm := dt.Format("15:04")

	gDM := []int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	var jy int
	if gy > 1600 {
		jy = 979
		gy -= 1600
	} else {
		jy = 0
		gy -= 621
	}
	gy2 := gy
	if gm > 2 {
		gy2 = gy + 1
	}
	days := (365 * gy) + (gy2+3)/4 - (gy2+99)/100 + (gy2+399)/400 - 80 + gd + gDM[gm-1]
	jy += 33 * (days / 12053)
	days %= 12053
	jy += 4 * (days / 1461)
	days %= 1461
	if days > 365 {
		jy += (days - 1) / 365
		days = (days - 1) % 365
	}
	var jm, jd int
	if days < 186 {
		jm = 1 + days/31
		jd = 1 + days%31
	} else {
		jm = 7 + (days-186)/30
		jd = 1 + (days-186)%30
	}
	return fmt.Sprintf("%04d-%02d-%02d %s", jy, jm, jd, hhmm)
}
