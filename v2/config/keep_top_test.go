package config

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/sagernet/sing-box/option"

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
	fingerprints := func(options *option.Options) map[string]string {
		items, _, err := StableProxyTags(ctx, options, nil)
		if err != nil {
			t.Fatal(err)
		}
		return Fingerprints(items)
	}
	old, new := fingerprints(before), fingerprints(after)
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

func TestChosenProxiesAreTheProfilesOnly(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	input, _ := ReadSingOptions(ctx, &ReadOptions{Content: keepTopProfile})
	keep := ChosenProxies(input, []string{"b", "e", "gone", "direct", "hidden §hide§"})
	if len(keep) != 2 || !keep["b"] || !keep["e"] {
		t.Fatalf("kept %v, want b and e", keep)
	}
	if len(ChosenProxies(input, []string{"gone"})) != 0 {
		t.Fatal("a choice none of whose proxies is left kept something")
	}
}

func TestKeepTagsReadFromTheSettings(t *testing.T) {
	var options HiddifyOptions
	if err := json.Unmarshal([]byte(`{"keep-tags":["a","b"],"keep-top":5}`), &options); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(options.KeepTags, []string{"a", "b"}) || options.KeepTop != 5 {
		t.Fatalf("read %v, %d", options.KeepTags, options.KeepTop)
	}
}

const keepTopEndpointsProfile = `{"outbounds":[
	{"type":"socks","tag":"a","server":"127.0.0.1","server_port":1081},
	{"type":"socks","tag":"b","server":"127.0.0.1","server_port":1082}
],"endpoints":[
	{"type":"wireguard","tag":"wg","address":["172.16.0.2/32"],"private_key":"YNXtAzepDqRv9H52osJVDQnznT5AL11eGFTK9ZPVtWU=",
		"peers":[{"address":"127.0.0.1","port":51820,"public_key":"Z1XXLsKYkYxuiYjJIkRvtIKFepCYHTgON+GwPq7SOV4=","allowed_ips":["0.0.0.0/0"]}]},
	{"type":"wireguard","tag":"wg-server","listen_port":51821,"address":["172.16.0.3/32"],"private_key":"YNXtAzepDqRv9H52osJVDQnznT5AL11eGFTK9ZPVtWU=",
		"peers":[{"public_key":"Z1XXLsKYkYxuiYjJIkRvtIKFepCYHTgON+GwPq7SOV4=","allowed_ips":["172.16.0.4/32"]}]},
	{"type":"openvpn-client","tag":"ovpn","server":"127.0.0.1","server_port":1194},
	{"type":"warp","tag":"warp"}
]}`

// Keep top manages the client endpoints as it does the outbounds: they are lazy in the config the
// core runs, WARP too. A WireGuard that listens is a server: it runs.
func TestKeepTopMakesClientEndpointsLazy(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	input, err := ReadSingOptions(ctx, &ReadOptions{Content: keepTopEndpointsProfile})
	if err != nil {
		t.Fatal(err)
	}
	candidates := LazyProxyTags(input)
	if want := []string{"a", "b", "wg", "ovpn", "warp"}; !slices.Equal(candidates, want) {
		t.Fatalf("candidates = %v, want %v", candidates, want)
	}
	opts := chainTestOptions(t, `{"chain-status":"off","keep-top":1}`)
	options, err := BuildConfig(ctx, opts, &ReadOptions{Options: input})
	if err != nil {
		t.Fatal(err)
	}
	if managed := ApplyKeepTop(options, opts, candidates); managed != 5 {
		t.Fatalf("managed %d, want 5", managed)
	}
	lazy := map[string]bool{}
	for _, end := range options.Endpoints {
		lazy[end.Tag] = end.Lazy
	}
	for _, out := range options.Outbounds {
		if out.Lazy {
			lazy[out.Tag] = true
		}
	}
	want := map[string]bool{"a": true, "b": true, "wg": true, "ovpn": true, "warp": true, "wg-server": false}
	for tag, wantLazy := range want {
		if got, found := lazy[tag]; (found || wantLazy) && got != wantLazy {
			t.Errorf("%s lazy = %v, want %v", tag, got, wantLazy)
		}
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		t.Fatalf("sing-box rejected the config: %v", err)
	}
}
