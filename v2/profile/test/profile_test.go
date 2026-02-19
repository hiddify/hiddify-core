package test

import (
	"fmt"
	"testing"

	"github.com/hiddify/hiddify-core/v2/profile"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func TestAddByContent(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	entity, err := profile.AddByUrl(ctx, "https://raw.githubusercontent.com/hiddify/hiddify-next/refs/heads/main/test.configs/warp", "", false)
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	fmt.Printf("entity: %v\n", entity)
	// Check if the content has been added correctly
	profileTitle := entity.Name
	expectedTitle := "🔥 WARP 🔥" // The Base64 decoded title
	if profileTitle != expectedTitle {
		t.Errorf("expected profile title to be %v, got %v", expectedTitle, profileTitle)
	}

	// Check subscription userinfo
	userInfo := entity.SubInfo
	if userInfo.Upload != 0 || userInfo.Download != 0 || userInfo.Total != 10737418240000000 || userInfo.Expire != 2546249531 {
		t.Errorf("subscription userinfo not parsed correctly, got: %v", userInfo)
	}

	// Check URLs
	supportURL := entity.SubInfo.SupportUrl
	if supportURL != "https://t.me/hiddify" {
		t.Errorf("expected support URL to be https://t.me/hiddify, got %v", supportURL)
	}

	profileWebPageURL := entity.SubInfo.WebPageUrl
	if profileWebPageURL != "https://hiddify.com" {
		t.Errorf("expected profile web page URL to be https://hiddify.com, got %v", profileWebPageURL)
	}
	profile.DeleteById(entity.Id)
	// You can further assert individual fields of warp configurations
}
