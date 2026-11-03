package config

import (
	"slices"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
)

const keepTopProfile = `{"outbounds":[
	{"type":"socks","tag":"a","server":"127.0.0.1","server_port":1081},
	{"type":"socks","tag":"b","server":"127.0.0.1","server_port":1082,"detour":"c"},
	{"type":"socks","tag":"c","server":"127.0.0.1","server_port":1083,"detour":"d"},
	{"type":"socks","tag":"d","server":"127.0.0.1","server_port":1084},
	{"type":"socks","tag":"e","server":"127.0.0.1","server_port":1085},
	{"type":"socks","tag":"hidden §hide§","server":"127.0.0.1","server_port":1086},
	{"type":"direct","tag":"direct"},
	{"type":"selector","tag":"group","outbounds":["a","b"]}
]}`

func TestUserProxyTagsAreTheSelectableProxies(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	input, err := ReadSingOptions(ctx, &ReadOptions{Content: keepTopProfile})
	if err != nil {
		t.Fatal(err)
	}
	if tags := UserProxyTags(input); !slices.Equal(tags, []string{"a", "b", "c", "d", "e"}) {
		t.Fatalf("tags = %v", tags)
	}
}

func TestFingerprintsChangeWithTheProxy(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	before, _ := ReadSingOptions(ctx, &ReadOptions{Content: keepTopProfile})
	after, _ := ReadSingOptions(ctx, &ReadOptions{Content: keepTopProfile})
	after.Outbounds[0].Options = before.Outbounds[4].Options // a now points where e does
	old, new := ProxyFingerprints(ctx, before), ProxyFingerprints(ctx, after)
	if old["a"] == new["a"] {
		t.Fatal("a changed, its fingerprint did not")
	}
	if old["d"] != new["d"] || old["d"] == "" {
		t.Fatal("d is the same, its fingerprint must be too")
	}
}

func TestKeepOnlyProxiesKeepsDetoursAndHidden(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	input, _ := ReadSingOptions(ctx, &ReadOptions{Content: keepTopProfile})
	dropped := KeepOnlyProxies(ctx, input, map[string]bool{"b": true})
	if dropped != 2 {
		t.Fatalf("dropped %d, want 2 (a and e)", dropped)
	}
	var tags []string
	for _, out := range input.Outbounds {
		tags = append(tags, out.Tag)
	}
	// b goes through c, c through d; the hidden proxy, direct and the group are not the profile's proxies
	want := []string{"b", "c", "d", "hidden §hide§", "direct", "group"}
	if !slices.Equal(tags, want) {
		t.Fatalf("kept %v, want %v", tags, want)
	}

	opts := chainTestOptions(t, `{"chain-status":"off"}`)
	options, err := BuildConfig(ctx, opts, &ReadOptions{Options: input})
	if err != nil {
		t.Fatal(err)
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejected the config: %v", err)
	}
	if items := selectorItems(options); slices.Contains(items, "a") || !slices.Contains(items, "b") {
		t.Fatalf("select = %v", items)
	}
}
