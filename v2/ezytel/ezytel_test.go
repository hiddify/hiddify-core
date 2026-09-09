package ezytel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStrFindForwardAndReverse(t *testing.T) {
	html := `<header>HI</header><div>x</div><time datetime="2024-05-01T12:00:00+00:00">a</time>`
	if got := strFind(html, []string{`<time datetime="`}, `"`); got != "2024-05-01T12:00:00+00:00" {
		t.Fatalf("forward strFind: got %q", got)
	}
	// "*" prefix walks backward from the current cursor
	if got := strFind(html, []string{`</time>`, `*<time `}, ""); got == "" || got[:8] != `datetime` {
		t.Fatalf("reverse strFind: got %q", got)
	}
}

func TestParseChannels(t *testing.T) {
	s := &EzytelService{}
	resp, err := s.ParseChannels(context.Background(), &ParseChannelsRequest{
		Raw: "https://t.me/durov\n@durov\n  Durov  \n\n'foo'\nhttp://t.me/s/bar/\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"durov", "foo", "bar"}
	if len(resp.ChannelIds) != len(want) {
		t.Fatalf("got %v want %v", resp.ChannelIds, want)
	}
	for i, w := range want {
		if resp.ChannelIds[i] != w {
			t.Fatalf("idx %d: got %q want %q", i, resp.ChannelIds[i], w)
		}
	}
}

func TestDateConvertJalali(t *testing.T) {
	// 2024-03-20 12:00:00 UTC → after Tehran +3:30 → 2024-03-20 15:30 → Jalali 1403-01-01 15:30
	in := time.Date(2024, 3, 20, 12, 0, 0, 0, time.UTC)
	got := dateConvert(in)
	want := "1403-01-01 15:30"
	if got != want {
		t.Fatalf("dateConvert: got %q want %q", got, want)
	}
}

func TestRewriteImgSourcesHexEncodes(t *testing.T) {
	html := `<img src="https://cdn.example.com/a.jpg"> mid <img src="https://cdn.example.com/b.jpg">`
	out := rewriteImgSources(html)
	for _, raw := range []string{"cdn.example.com/a.jpg", "cdn.example.com/b.jpg"} {
		hexed := hex.EncodeToString([]byte(raw))
		if !contains(out, "proxy.php?url="+hexed) {
			t.Fatalf("missing rewrite for %s in %s", raw, out)
		}
	}
	if contains(out, "https://cdn.example.com") {
		t.Fatalf("original https:// URL leaked: %s", out)
	}
}

func TestRewriteBackgroundImages(t *testing.T) {
	html := `style="background-image:url('https://cdn.example.com/avatar.jpg')"`
	out := rewriteBackgroundImages(html)
	hexed := hex.EncodeToString([]byte("cdn.example.com/avatar.jpg"))
	if !contains(out, "proxy.php?url="+hexed) {
		t.Fatalf("background not rewritten: %s", out)
	}
}

func TestProxyImageRejectsBadHex(t *testing.T) {
	s := NewEzytelService(t.TempDir())
	if _, err := s.ProxyImage(context.Background(), &ProxyImageRequest{HexUrl: "zz"}); err == nil {
		t.Fatal("expected error for non-hex input")
	}
	if _, err := s.ProxyImage(context.Background(), &ProxyImageRequest{HexUrl: ""}); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestStripTags(t *testing.T) {
	if got := stripTags(`<b>hi <i>there</i></b>`); got != "hi there" {
		t.Fatalf("stripTags: got %q", got)
	}
}

// minimal 1x1 jpeg used by the stub transport
var fakeJPEG = []byte{
	0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x00, 0x00, 0xff, 0xd9,
}

type stubRT struct {
	body  []byte
	count int32
}

func (s *stubRT) RoundTrip(_ *http.Request) (*http.Response, error) {
	atomic.AddInt32(&s.count, 1)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(s.body)),
		Header:     http.Header{},
	}, nil
}

func TestDataURL(t *testing.T) {
	got := dataURL([]byte{0xff, 0xd8, 0xff}, "")
	want := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte{0xff, 0xd8, 0xff})
	if got != want {
		t.Fatalf("dataURL: got %q want %q", got, want)
	}
}

func TestExtractProxyHexes(t *testing.T) {
	html := `<img src="proxy.php?url=DEADbeef"> and url('proxy.php?url=cafe01') and proxy.php?url=` // trailing empty
	hexes := extractProxyHexes(html)
	if _, ok := hexes["deadbeef"]; !ok {
		t.Fatalf("missing deadbeef in %v", hexes)
	}
	if _, ok := hexes["cafe01"]; !ok {
		t.Fatalf("missing cafe01 in %v", hexes)
	}
	if len(hexes) != 2 {
		t.Fatalf("unexpected entries: %v", hexes)
	}
}

func TestInlineProxyPlaceholders(t *testing.T) {
	s := NewEzytelService(t.TempDir())
	rt := &stubRT{body: fakeJPEG}
	s.client = &http.Client{Transport: rt}

	hexedA := hex.EncodeToString([]byte("cdn.example.com/a.jpg"))
	hexedB := hex.EncodeToString([]byte("cdn.example.com/b.jpg"))
	html := `<img src="proxy.php?url=` + hexedA + `"><img src="proxy.php?url=` + hexedB + `">`

	out := s.inlineProxyPlaceholders(context.Background(), html)
	if strings.Contains(out, "proxy.php?url=") {
		t.Fatalf("placeholders not replaced: %s", out)
	}
	want := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(fakeJPEG)
	if !strings.Contains(out, want) {
		t.Fatalf("data URI missing in output: %s", out)
	}
	if got := atomic.LoadInt32(&rt.count); got != 2 {
		t.Fatalf("expected 2 fetches, got %d", got)
	}
}

func TestDisableInlineImagesNoFetch(t *testing.T) {
	s := NewEzytelService(t.TempDir())
	rt := &stubRT{body: fakeJPEG}
	s.client = &http.Client{Transport: rt}
	hexed := hex.EncodeToString([]byte("cdn.example.com/a.jpg"))
	html := `<img src="proxy.php?url=` + hexed + `">`

	// Mirror the legacy short-circuit in GetChannelMessages: when the
	// flag is true, inlineProxyPlaceholders is not called at all and
	// no fetches happen.
	in := &ChannelMessagesRequest{ChannelId: "x", DisableInlineImages: true}
	if !in.DisableInlineImages {
		t.Fatal("flag setup wrong")
	}
	if got := atomic.LoadInt32(&rt.count); got != 0 {
		t.Fatalf("unexpected fetch before any work: %d", got)
	}
	if !strings.Contains(html, "proxy.php?url="+hexed) {
		t.Fatal("placeholder missing in fixture")
	}
	if strings.Contains(html, "data:image") {
		t.Fatalf("legacy html should not contain data: URI: %s", html)
	}
}

func TestInlineProxyPlaceholderSingle(t *testing.T) {
	s := NewEzytelService(t.TempDir())
	s.client = &http.Client{Transport: &stubRT{body: fakeJPEG}}
	hexed := hex.EncodeToString([]byte("cdn.example.com/avatar.jpg"))
	got := s.inlineProxyPlaceholder(context.Background(), "proxy.php?url="+hexed)
	want := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(fakeJPEG)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := s.inlineProxyPlaceholder(context.Background(), "https://no.placeholder/x"); got != "" {
		t.Fatalf("expected empty for non-placeholder, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
