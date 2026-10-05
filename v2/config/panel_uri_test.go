package config

// Every URI variant hiddifypanel emits (proxy_v3/proxy_templates/sublink/uri and sublink/vmess)
// must convert through ray2sing into an outbound/endpoint that sing-box accepts, i.e. that does
// not end up as an hinvalid placeholder.

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/url"
	"strings"
	"testing"

	"github.com/hiddify/ray2sing/ray2sing"
	box "github.com/sagernet/sing-box"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	"golang.org/x/crypto/ssh"
)

const (
	pUUID   = "8bb28661-a350-4360-b47b-bc55dd0b57d6"
	pServer = "1.2.3.4"
	pDomain = "a.example.com"
	pPath   = "Zx9kPath"
	pPBK    = "yanlaRXp_Qaive33liPQzYJbIsh5rFZmJ8Bd-Iy_wjM"
	pSID    = "1bf7"
	pPCS    = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

type panelCase struct{ name, link string }

func qs(pairs ...string) string {
	v := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		v.Add(pairs[i], pairs[i+1])
	}
	return v.Encode()
}

func key32(b byte) string {
	k := make([]byte, 32)
	for i := range k {
		k[i] = b + byte(i)
	}
	return base64.StdEncoding.EncodeToString(k)
}

// security variants from sublink/uri/security/tls_http.j2
func panelSecurities() map[string][]string {
	return map[string][]string{
		"none":     {"security", "none"},
		"tls":      {"security", "tls", "fp", "chrome", "sni", pDomain, "alpn", "h2,http/1.1"},
		"tls-h3":   {"security", "tls", "fp", "chrome", "sni", pDomain, "alpn", "h3"},
		"insecure": {"security", "tls", "fp", "chrome", "sni", pDomain, "alpn", "h2,http/1.1", "allowInsecure", "true", "insecure", "true", "pcs", pPCS},
		"fragment": {"security", "tls", "fp", "chrome", "sni", pDomain, "alpn", "http/1.1", "fragment", "10-100,1-10,tlshello"},
		"reality":  {"security", "reality", "fp", "chrome", "sni", "www.google.com", "pbk", pPBK, "sid", pSID},
		// "unsafe" (Xray): no uTLS, standard Go TLS ClientHello
		"tls-unsafe-fp": {"security", "tls", "fp", "unsafe", "sni", pDomain, "alpn", "h2,http/1.1"},
	}
}

// transport variants from sublink/uri/transport/*.j2
func panelTransports() map[string][]string {
	downloadSettings := `{"downloadSettings":{"address":"dl.example.com","port":443,"network":"xhttp","security":"tls","tlsSettings":{"serverName":"dl.example.com","fingerprint":"chrome","alpn":["h2","http/1.1"],"pinnedPeerCertSha256":"` + pPCS + `"},"xhttpSettings":{"path":"/dl` + pPath + `","host":"` + pDomain + `","mode":"auto","headers":{"User-Agent":"Mozilla/5.0"}}},"headers":{"User-Agent":"Mozilla/5.0"}}`
	return map[string][]string{
		"tcp":         {"type", "tcp", "headerType", "none"},
		"http":        {"type", "tcp", "path", "/" + pPath, "host", pDomain, "headerType", "http"},
		"ws":          {"type", "ws", "path", "/" + pPath, "host", pDomain},
		"httpupgrade": {"type", "httpupgrade", "path", "/" + pPath, "host", pDomain},
		"grpc":        {"type", "grpc", "serviceName", pPath, "mode", "multigun", "authority", pDomain},
		"xhttp":       {"type", "xhttp", "mode", "auto", "path", "/" + pPath, "host", pDomain},
		"xhttp-extra": {"type", "xhttp", "mode", "auto", "path", "/" + pPath, "host", pDomain, "extra", downloadSettings},
	}
}

func panelVmess(name string, fields map[string]any) panelCase {
	data := map[string]any{
		"v": "2", "add": pServer, "port": 443, "id": pUUID, "aid": 0, "scy": "auto",
		"net": "tcp", "type": "none", "host": pDomain, "alpn": "h2,http/1.1", "path": "/" + pPath,
		"sni": pDomain, "fp": "chrome", "ps": name,
	}
	for k, v := range fields {
		data[k] = v
	}
	b, _ := json.Marshal(data)
	return panelCase{name, "vmess://" + base64.StdEncoding.EncodeToString(b)}
}

func panelCases(t *testing.T) []panelCase {
	var cases []panelCase
	for secName, sec := range panelSecurities() {
		for trName, tr := range panelTransports() {
			if secName == "reality" && (trName == "ws" || trName == "httpupgrade" || trName == "http") {
				continue // panel does not combine reality with these transports
			}
			params := append(append([]string{"hiddify", "1"}, sec...), tr...)
			name := secName + "-" + trName
			vlessParams := append(append([]string{}, params...), "encryption", "none")
			if secName == "reality" && trName == "tcp" {
				vlessParams = append(vlessParams, "flow", "xtls-rprx-vision")
			}
			cases = append(cases,
				panelCase{"vless " + name, "vless://" + pUUID + "@" + pServer + ":443/?" + qs(vlessParams...) + "#vless%20" + name},
				panelCase{"trojan " + name, "trojan://" + pUUID + "@" + pServer + ":443/?" + qs(params...) + "#trojan%20" + name},
			)
		}
	}

	// vmess (sublink/vmess)
	tlsFields := map[string]any{"tls": "tls"}
	cases = append(cases,
		panelVmess("vmess tcp", map[string]any{"net": "tcp", "type": "none"}),
		panelVmess("vmess http", map[string]any{"net": "tcp", "type": "http"}),
		panelVmess("vmess ws tls", map[string]any{"net": "ws", "type": "none", "tls": "tls"}),
		panelVmess("vmess httpupgrade tls", map[string]any{"net": "httpupgrade", "type": "none", "tls": "tls"}),
		panelVmess("vmess grpc tls", map[string]any{"net": "grpc", "type": "gun", "path": pPath, "tls": "tls"}),
		panelVmess("vmess xhttp tls", map[string]any{"net": "xhttp", "type": "auto", "tls": "tls"}),
		panelVmess("vmess xhttp tls h3", map[string]any{"net": "xhttp", "type": "auto", "tls": "tls", "alpn": "h3"}),
		panelVmess("vmess xhttp extra", map[string]any{"net": "xhttp", "type": "auto", "tls": "tls", "extra": `{"downloadSettings":{"address":"dl.example.com","port":443,"network":"xhttp","security":"tls","tlsSettings":{"serverName":"dl.example.com"},"xhttpSettings":{"path":"/p","host":"a.example.com","mode":"auto"}}}`}),
		panelVmess("vmess tls pcs", map[string]any{"net": "ws", "type": "none", "tls": "tls", "pcs": pPCS}),
		panelVmess("vmess tls insecure", map[string]any{"net": "ws", "type": "none", "tls": "tls", "allowInsecure": true, "insecure": true}),
		panelVmess("vmess reality", map[string]any{"net": "tcp", "type": "none", "tls": "reality", "pbk": pPBK, "sid": pSID, "sni": "www.google.com"}),
	)
	_ = tlsFields

	// ss (sublink/uri/ss.j2): userinfo = b64(method:b64(secret32):b64(uuidhex32))
	ssPass := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32))) + ":" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	ssUser := base64.StdEncoding.EncodeToString([]byte("2022-blake3-aes-256-gcm:" + ssPass))
	cases = append(cases,
		panelCase{"ss 2022", "ss://" + ssUser + "@" + pServer + ":443/?" + qs("uot", "true", "hiddify", "1") + "#ss"},
		panelCase{"ss v2ray-plugin", "ss://" + ssUser + "@" + pServer + ":443/?" + qs("uot", "true", "hiddify", "1", "plugin", "v2ray-plugin;mode=websocket;path="+pPath+";host="+pDomain+";tls") + "#ss-v2ray"},
		panelCase{"ss obfs faketls", "ss://" + ssUser + "@" + pServer + ":443/?" + qs("uot", "true", "hiddify", "1", "plugin", "obfs-local;obfs=tls;obfs-host=www.bing.com") + "#ss-faketls"},
	)

	// ssh (sublink/uri/ssh.j2)
	pub, priv, _ := ed25519.GenerateKey(nil)
	privPEM, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	hostPub, _ := ssh.NewPublicKey(pub)
	hostKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(hostPub)))
	pk := string(pem.EncodeToMemory(privPEM))
	cases = append(cases, panelCase{"ssh", "ssh://" + pUUID + ":pass@" + pServer + ":2222/?" + qs("pk", pk, "private_key", pk, "hk", hostKey+","+hostKey) + "#ssh"})

	wgPeer := key32(1)
	cases = append(cases,
		panelCase{"anytls", "anytls://" + pUUID + "@" + pServer + ":443/?" + qs("sni", pDomain, "insecure", "0") + "#anytls"},
		panelCase{"anytls insecure", "anytls://" + pUUID + "@" + pServer + ":443/?" + qs("sni", pDomain, "insecure", "1") + "#anytls-insecure"},
		panelCase{"hysteria", "hysteria://" + pServer + ":443/?" + qs("auth", pUUID, "protocol", "udp", "alpn", "h3", "upmbps", "100", "downmbps", "500", "sni", pDomain, "peer", pDomain, "insecure", "1", "obfs", pPath) + "#hy"},
		panelCase{"hysteria port range", "hysteria://" + pServer + ":20000-30000/?" + qs("auth", pUUID, "protocol", "udp", "alpn", "h3", "upmbps", "100", "downmbps", "500", "sni", pDomain) + "#hy-range"},
		panelCase{"hy2", "hy2://" + pUUID + "@" + pServer + ":443/?" + qs("sni", pDomain, "insecure", "1", "obfs", "salamander", "obfs-password", pPath) + "#hy2"},
		panelCase{"hy2 no obfs", "hy2://" + pUUID + "@" + pServer + ":443/?" + qs("sni", pDomain, "insecure", "", "obfs", "", "obfs-password", "") + "#hy2-plain"},
		panelCase{"hy2 port range", "hy2://" + pUUID + "@" + pServer + ":20000-30000,40000-40010/?" + qs("sni", pDomain) + "#hy2-range"},
		panelCase{"mierus tcp+udp", "mierus://" + pUUID + ":h@" + pServer + "/?" + qs("profile", "default", "mtu", "1400", "multiplexing", "MULTIPLEXING_HIGH", "handshake-mode", "HANDSHAKE_NO_WAIT", "port", "45043", "protocol", "TCP", "port", "13528", "protocol", "UDP") + "#mieru"},
		panelCase{"mierus ranges", "mierus://" + pUUID + ":h@" + pServer + "/?" + qs("profile", "default", "mtu", "1400", "multiplexing", "MULTIPLEXING_LOW", "handshake-mode", "HANDSHAKE_STANDARD", "port", "20000-20010", "protocol", "TCP") + "#mieru-range"},
		panelCase{"naive", "naive://" + pUUID + ":" + pPath + "@" + pServer + ":443/?" + qs("padding", "true", "extra-headers", "X-API-Key:"+pPath, "host", pDomain, "sni", pDomain) + "#naive"},
		panelCase{"naive+https", "naive+https://" + pUUID + ":" + pPath + "@" + pServer + ":443/?" + qs("padding", "true", "extra-headers", "X-API-Key:"+pPath, "host", pDomain, "sni", pDomain) + "#naive-https"},
		panelCase{"naive+quic", "naive+quic://" + pUUID + ":" + pPath + "@" + pServer + ":443/?" + qs("padding", "true", "extra-headers", "X-API-Key:"+pPath, "host", pDomain, "sni", pDomain) + "#naive-quic"},
		panelCase{"snell", "snell://" + pPath + "@" + pServer + ":443/?" + qs("version", "4") + "#snell"},
		panelCase{"socks", "socks://" + pUUID + ":h@" + pServer + ":1080#socks"},
		panelCase{"tuic", "tuic://" + pUUID + ":" + pUUID + "@" + pServer + ":443/?" + qs("alpn", "h3", "congestion_control", "cubic", "udp_relay_mode", "native", "sni", pDomain, "allow_insecure", "1") + "#tuic"},
		panelCase{"wg", "wg://" + pServer + ":51820/?" + qs("pk", key32(9), "local_address", "10.90.0.12/32", "peer_pk", wgPeer, "pre_shared_key", key32(5), "mtu", "1380", "reserved", "0,0,0", "ifp", "5-10") + "#wg"},
		panelCase{"wg no ifp", "wg://" + pServer + ":51820/?" + qs("pk", key32(9), "local_address", "10.90.0.12/32", "peer_pk", wgPeer, "pre_shared_key", key32(5), "mtu", "1380", "reserved", "0,0,0", "ifp", "") + "#wg-plain"},
	)
	return cases
}

func panelCheck(t *testing.T, link string) string {
	ctx := libbox.BaseContext(nil)
	options, err := ray2sing.Ray2SingboxOptions(ctx, link, false)
	if err != nil {
		return "ray2sing: " + err.Error()
	}
	if len(options.Outbounds)+len(options.Endpoints) == 0 {
		return "ray2sing: no outbound produced"
	}
	// round trip like a saved profile, so option validation (e.g. transport checks) runs
	content, err := options.MarshalJSONContext(ctx)
	if err != nil {
		return "marshal: " + err.Error()
	}
	var reparsed option.Options
	if err := reparsed.UnmarshalJSONContext(ctx, content); err != nil {
		return "unmarshal: " + err.Error()
	}
	for _, out := range reparsed.Outbounds {
		if inv, ok := out.Options.(*option.HInvalidOptions); ok {
			return "option parse (" + inv.OriginalType + "): " + inv.Err.Error()
		}
	}
	for _, ep := range reparsed.Endpoints {
		if inv, ok := ep.Options.(*option.HInvalidOptions); ok {
			return "endpoint parse (" + inv.OriginalType + "): " + inv.Err.Error()
		}
	}
	instance, err := box.New(box.Options{Context: ctx, Options: reparsed})
	if err != nil {
		return "box: " + err.Error()
	}
	defer instance.Close()
	for _, out := range instance.Outbound().Outbounds() {
		if out.Type() == C.TypeHInvalidConfig {
			return "outbound init [" + out.Tag() + "] invalid"
		}
	}
	for _, ep := range instance.Endpoint().Endpoints() {
		if ep.Type() == C.TypeHInvalidConfig {
			return "endpoint init [" + ep.Tag() + "] invalid"
		}
	}
	return ""
}

func TestPanelURIsConvert(t *testing.T) {
	for _, c := range panelCases(t) {
		t.Run(c.name, func(t *testing.T) {
			if problem := panelCheck(t, c.link); problem != "" {
				t.Errorf("%s\n  link: %.300s", problem, c.link)
			}
		})
	}
}

// fields that must survive the conversion (not just be accepted)
func TestPanelURIFields(t *testing.T) {
	pinB64 := `"certificate_sha256":"47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="`
	expect := map[string][]string{
		"vless insecure-grpc":   {pinB64, `"insecure":true`},
		"vmess tls pcs":         {pinB64},
		"vless fragment-ws":     {`"fragment":true`},
		"trojan fragment-tcp":   {`"fragment":true`},
		"vless reality-tcp":     {`"flow":"xtls-rprx-vision"`, `"public_key":"` + pPBK + `"`, `"short_id":"` + pSID + `"`},
		"vless tls-xhttp-extra": {`"download":{"host":"` + pDomain + `","path":"/dl` + pPath + `"`, `"server":"dl.example.com"`, pinB64, `"User-Agent":"Mozilla/5.0"`},
		"tuic":                  {`"alpn":"h3"`, `"congestion_control":"cubic"`, `"udp_relay_mode":"native"`, `"insecure":true`},
		"hysteria":              {`"obfs":"` + pPath + `"`, `"server_name":"` + pDomain + `"`, `"up_mbps":100`, `"down_mbps":500`},
		"hysteria port range":   {`"server_ports":"20000:30000"`},
		"hy2":                   {`"type":"salamander"`, `"password":"` + pPath + `"`, `"insecure":true`},
		"hy2 port range":        {`"server_ports":["20000:30000","40000:40010"]`},
		"naive+quic":            {`"quic":true`, `"X-API-Key":"` + pPath + `"`},
		"naive+https":           {`"X-API-Key":"` + pPath + `"`, `"server_name":"` + pDomain + `"`},
		"ss v2ray-plugin":       {`"plugin":"v2ray-plugin"`, `"plugin_opts":"mode=websocket;path=` + pPath + `;host=` + pDomain + `;tls"`, `"udp_over_tcp":true`},
		"ss obfs faketls":       {`"plugin":"obfs-local"`, `"plugin_opts":"obfs=tls;obfs-host=www.bing.com"`},
		"anytls insecure":       {`"type":"anytls"`, `"insecure":true`, `"password":"` + pUUID + `"`},
		"snell":                 {`"type":"snell"`, `"version":4`, `"psk":"` + pPath + `"`},
		"wg":                    {`"public_key":"` + key32(1) + `"`, `"pre_shared_key":"` + key32(5) + `"`, `"count":"5-10"`},
		"mierus tcp+udp":        {`{"protocol":"TCP","port":45043}`, `{"protocol":"UDP","port":13528}`, `"multiplexing":"MULTIPLEXING_HIGH"`},
		"socks":                 {`"type":"socks"`, `"username":"` + pUUID + `"`, `"password":"h"`},
	}
	ctx := libbox.BaseContext(nil)
	found := map[string]bool{}
	for _, c := range panelCases(t) {
		wants, ok := expect[c.name]
		if !ok {
			continue
		}
		found[c.name] = true
		options, err := ray2sing.Ray2SingboxOptions(ctx, c.link, false)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		content, _ := options.MarshalJSONContext(ctx)
		compact := strings.Join(strings.Fields(string(content)), "")
		for _, want := range wants {
			if !strings.Contains(compact, want) {
				t.Errorf("%s: missing %s\n  got: %.700s", c.name, want, compact)
			}
		}
	}
	for name := range expect {
		if !found[name] {
			t.Errorf("no panel case named %q", name)
		}
	}
}
