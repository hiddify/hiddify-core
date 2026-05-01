package ezytel

import (
	"context"
	"encoding/hex"
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
