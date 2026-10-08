package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
)

// The same settings and profile must build the same config: hot reload compares the running
// config with the new one and refuses anything but outbound/endpoint changes.
func TestBuildConfigIsDeterministic(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	profile := `{"outbounds":[
		{"type":"socks","tag":"proxy-a","server":"127.0.0.1","server_port":1080},
		{"type":"vless","tag":"proxy-b","server":"example.com","server_port":443,"uuid":"25da296e-1d96-48ae-9867-4342796cd742"}
	]}`
	build := func() []byte {
		options, err := BuildConfig(ctx, DefaultHiddifyOptions(), &ReadOptions{Content: profile})
		if err != nil {
			t.Fatal(err)
		}
		content, err := options.MarshalJSONContext(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return content
	}
	first := build()
	for i := 0; i < 50; i++ {
		if next := build(); !bytes.Equal(first, next) {
			t.Fatalf("build %d differs from the first one:\n%s", i, firstDifference(first, next))
		}
	}
}

func firstDifference(a, b []byte) string {
	var ma, mb map[string]json.RawMessage
	json.Unmarshal(a, &ma)
	json.Unmarshal(b, &mb)
	for key, value := range ma {
		if !bytes.Equal(value, mb[key]) {
			i := 0
			for i < len(value) && i < len(mb[key]) && value[i] == mb[key][i] {
				i++
			}
			start := max(0, i-200)
			return fmt.Sprintf("%s: ...%s\n  vs ...%s", key, value[start:min(len(value), i+100)], mb[key][start:min(len(mb[key]), i+100)])
		}
	}
	return "?"
}
