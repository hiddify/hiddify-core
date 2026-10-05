package config

import (
	"net/url"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

const testFinalMask = `{"tcp":[{"type":"fragment","settings":{"packets":"tlshello","length":"1-104","delay":"0","maxSplit":"0"}},{"type":"fragment","settings":{"packets":"1-1","length":"1-114","delay":"1","maxSplit":"11"}}]}`

func finalMaskAfterBuild(t *testing.T, content string) *option.FinalMaskOptions {
	t.Helper()
	ctx := libbox.BaseContext(nil)
	// the import path: links are converted to sing-box options first
	profile, err := ParseConfig(ctx, &ReadOptions{Content: content}, false, DefaultHiddifyOptions(), false)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	profileJSON, err := profile.MarshalJSONContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	options, err := BuildConfig(ctx, DefaultHiddifyOptions(), &ReadOptions{Content: string(profileJSON)})
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejected the config: %v", err)
	}
	for _, out := range options.Outbounds {
		if wrapper, ok := out.Options.(option.DialerOptionsWrapper); ok {
			if fm := wrapper.TakeDialerOptions().FinalMask; fm != nil {
				return fm
			}
		}
	}
	return nil
}

func TestFinalMaskFromProfileJSON(t *testing.T) {
	fm := finalMaskAfterBuild(t, `{"outbounds":[{"type":"socks","tag":"proxy","server":"127.0.0.1","server_port":1080,"final_mask":`+testFinalMask+`}]}`)
	if fm == nil || len(fm.TCP) != 2 {
		t.Fatalf("final_mask lost: %+v", fm)
	}
}

func TestFinalMaskFromLink(t *testing.T) {
	link := "vless://25da296e-1d96-48ae-9867-4342796cd742@example.com:443?security=tls&sni=example.com&type=tcp&fm=" + url.QueryEscape(testFinalMask) + "#fm"
	fm := finalMaskAfterBuild(t, link)
	if fm == nil || len(fm.TCP) != 2 || fm.TCP[0].Type != "fragment" {
		t.Fatalf("final_mask lost: %+v", fm)
	}
}
