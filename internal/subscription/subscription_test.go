package subscription

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func b64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func b64std(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// ---- 各协议分享链接样例 ----

func TestParseSS_SIP002(t *testing.T) {
	link := "ss://" + b64("aes-256-gcm:pass123") + "@1.2.3.4:8388#" + url.PathEscape("🇭🇰 香港SS")
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "Shadowsocks" || n.Server != "1.2.3.4" || n.Name != "🇭🇰 香港SS" {
		t.Fatalf("节点基础字段错误: %+v", n)
	}
	raw := n.Raw
	if raw["cipher"] != "aes-256-gcm" || raw["password"] != "pass123" {
		t.Fatalf("ss 凭证字段错误: %v", raw)
	}
	if raw["port"] != 8388 || raw["type"] != "ss" || raw["name"] != "🇭🇰 香港SS" {
		t.Fatalf("ss raw 字段错误: %v", raw)
	}
}

func TestParseSS_Legacy(t *testing.T) {
	// 旧式：method:password@host:port 整体 base64
	link := "ss://" + b64std("aes-256-cfb:legacypass@2.3.4.5:443") + "#" + url.PathEscape("旧式SS")
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	raw := n.Raw
	if raw["cipher"] != "aes-256-cfb" || raw["password"] != "legacypass" ||
		raw["server"] != "2.3.4.5" || raw["port"] != 443 {
		t.Fatalf("旧式 ss 解析错误: %v", raw)
	}
	if n.Name != "旧式SS" {
		t.Fatalf("旧式 ss 名称错误: %q", n.Name)
	}
}

func TestParseSS_PlainUserinfo(t *testing.T) {
	// SIP002 明文 userinfo（未 base64）
	link := "ss://chacha20-ietf-poly1305:plainpass@3.4.5.6:1080#plain"
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Raw["cipher"] != "chacha20-ietf-poly1305" || n.Raw["password"] != "plainpass" {
		t.Fatalf("明文 userinfo ss 解析错误: %v", n.Raw)
	}
}

func TestParseSSR(t *testing.T) {
	payload := "4.5.6.7:12345:auth_aes128_md5:aes-256-cfb:tls1.2_ticket_auth:" + b64("ssrpass") +
		"/?obfsparam=" + b64("obfsp") + "&protoparam=" + b64("protop") + "&remarks=" + b64("SSR节点")
	n, err := ParseShareLink("ssr://" + b64(payload))
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "ShadowsocksR" || n.Name != "SSR节点" || n.Server != "4.5.6.7" {
		t.Fatalf("ssr 基础字段错误: %+v", n)
	}
	raw := n.Raw
	if raw["type"] != "ssr" || raw["port"] != 12345 ||
		raw["protocol"] != "auth_aes128_md5" || raw["cipher"] != "aes-256-cfb" ||
		raw["obfs"] != "tls1.2_ticket_auth" || raw["password"] != "ssrpass" ||
		raw["obfs-param"] != "obfsp" || raw["protocol-param"] != "protop" {
		t.Fatalf("ssr raw 字段错误: %v", raw)
	}
}

func TestParseVmess(t *testing.T) {
	jsonBody := `{"v":"2","ps":"Vmess节点","add":"vm.example.com","port":"443","id":"11111111-2222-3333-4444-555555555555","aid":0,"scy":"auto","net":"ws","type":"none","host":"cdn.example.com","path":"/ws","tls":"tls","sni":"sni.example.com","alpn":"h2,http/1.1"}`
	n, err := ParseShareLink("vmess://" + b64(jsonBody))
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "Vmess" || n.Name != "Vmess节点" || n.Server != "vm.example.com" {
		t.Fatalf("vmess 基础字段错误: %+v", n)
	}
	raw := n.Raw
	if raw["uuid"] != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("vmess uuid 错误: %v", raw["uuid"])
	}
	if raw["port"] != 443 || raw["alterId"] != 0 || raw["cipher"] != "auto" {
		t.Fatalf("vmess 端口/alterId/cipher 错误: %v", raw)
	}
	if raw["tls"] != true || raw["servername"] != "sni.example.com" {
		t.Fatalf("vmess tls 字段错误: %v", raw)
	}
	if raw["network"] != "ws" {
		t.Fatalf("vmess network 错误: %v", raw["network"])
	}
	ws, ok := raw["ws-opts"].(map[string]any)
	if !ok || ws["path"] != "/ws" {
		t.Fatalf("vmess ws-opts 错误: %v", raw["ws-opts"])
	}
	headers, ok := ws["headers"].(map[string]any)
	if !ok || headers["Host"] != "cdn.example.com" {
		t.Fatalf("vmess ws Host 错误: %v", ws["headers"])
	}
	alpn, ok := raw["alpn"].([]string)
	if !ok || len(alpn) != 2 || alpn[0] != "h2" {
		t.Fatalf("vmess alpn 错误: %v", raw["alpn"])
	}
}

func TestParseVless_Reality(t *testing.T) {
	link := "vless://aaaa-bbbb@vl.example.com:443?encryption=none&security=reality&flow=xtls-rprx-vision" +
		"&pbk=PUBLIC_KEY&sid=ab12&fp=chrome&sni=www.microsoft.com&type=tcp#" + url.PathEscape("Vless-Reality")
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "Vless" || n.Name != "Vless-Reality" {
		t.Fatalf("vless 基础字段错误: %+v", n)
	}
	raw := n.Raw
	if raw["uuid"] != "aaaa-bbbb" || raw["flow"] != "xtls-rprx-vision" || raw["tls"] != true {
		t.Fatalf("vless uuid/flow/tls 错误: %v", raw)
	}
	reality, ok := raw["reality-opts"].(map[string]any)
	if !ok || reality["public-key"] != "PUBLIC_KEY" || reality["short-id"] != "ab12" {
		t.Fatalf("vless reality-opts 错误: %v", raw["reality-opts"])
	}
	if raw["client-fingerprint"] != "chrome" || raw["servername"] != "www.microsoft.com" {
		t.Fatalf("vless fp/servername 错误: %v", raw)
	}
	if _, hasNetwork := raw["network"]; hasNetwork {
		t.Fatalf("type=tcp 不应设置 network: %v", raw)
	}
}

func TestParseVless_WS(t *testing.T) {
	link := "vless://uuid-1@ws.example.com:8443?security=tls&type=ws&path=%2Fray&host=h.example.com&sni=s.example.com#ws"
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	raw := n.Raw
	if raw["network"] != "ws" || raw["tls"] != true || raw["port"] != 8443 {
		t.Fatalf("vless ws 字段错误: %v", raw)
	}
	ws := raw["ws-opts"].(map[string]any)
	if ws["path"] != "/ray" || ws["headers"].(map[string]any)["Host"] != "h.example.com" {
		t.Fatalf("vless ws-opts 错误: %v", ws)
	}
}

func TestParseTrojan(t *testing.T) {
	link := "trojan://p%40ss@tj.example.com:443?sni=t.example.com&allowInsecure=1&alpn=h2%2Chttp%2F1.1&type=grpc&serviceName=svc#Trojan节点"
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "Trojan" || n.Server != "tj.example.com" {
		t.Fatalf("trojan 基础字段错误: %+v", n)
	}
	raw := n.Raw
	if raw["password"] != "p@ss" {
		t.Fatalf("trojan 密码未正确 URL 解码: %v", raw["password"])
	}
	if raw["sni"] != "t.example.com" || raw["skip-cert-verify"] != true {
		t.Fatalf("trojan sni/skip-cert-verify 错误: %v", raw)
	}
	if raw["network"] != "grpc" {
		t.Fatalf("trojan network 错误: %v", raw["network"])
	}
	grpc := raw["grpc-opts"].(map[string]any)
	if grpc["grpc-service-name"] != "svc" {
		t.Fatalf("trojan grpc-opts 错误: %v", grpc)
	}
}

func TestParseHysteria2(t *testing.T) {
	link := "hy2://hy2pass@hy.example.com:8443?sni=hy.example.com&insecure=1&obfs=salamander&obfs-password=obfspw#Hy2节点"
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "Hysteria2" || n.Name != "Hy2节点" {
		t.Fatalf("hysteria2 基础字段错误: %+v", n)
	}
	raw := n.Raw
	if raw["type"] != "hysteria2" || raw["password"] != "hy2pass" || raw["port"] != 8443 {
		t.Fatalf("hysteria2 基本字段错误: %v", raw)
	}
	if raw["sni"] != "hy.example.com" || raw["skip-cert-verify"] != true ||
		raw["obfs"] != "salamander" || raw["obfs-password"] != "obfspw" {
		t.Fatalf("hysteria2 sni/obfs 字段错误: %v", raw)
	}
}

func TestParseTuic(t *testing.T) {
	link := "tuic://uu-id-1:tuicpass@tuic.example.com:10443?sni=t.example.com&congestion_control=bbr&udp_relay_mode=native&alpn=h3#Tuic节点"
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "Tuic" {
		t.Fatalf("tuic 展示名错误: %v", n.Type)
	}
	raw := n.Raw
	if raw["uuid"] != "uu-id-1" || raw["password"] != "tuicpass" || raw["port"] != 10443 {
		t.Fatalf("tuic 凭证字段错误: %v", raw)
	}
	if raw["congestion-controller"] != "bbr" || raw["udp-relay-mode"] != "native" {
		t.Fatalf("tuic 传输字段错误: %v", raw)
	}
}

func TestParseSnell(t *testing.T) {
	link := "snell://snellpsk@sn.example.com:6666?version=3#Snell节点"
	n, err := ParseShareLink(link)
	if err != nil {
		t.Fatalf("ParseShareLink: %v", err)
	}
	if n.Type != "Snell" {
		t.Fatalf("snell 展示名错误: %v", n.Type)
	}
	raw := n.Raw
	if raw["psk"] != "snellpsk" || raw["version"] != 3 || raw["port"] != 6666 {
		t.Fatalf("snell 字段错误: %v", raw)
	}
}

// ---- Clash YAML ----

const clashYAML = `proxies:
  - name: "🇭🇰 Clash-Hy2"
    type: hysteria2
    server: c-hy2.example.com
    port: 443
    password: pw1
    sni: c-hy2.example.com
  - name: "Clash-Vless"
    type: vless
    server: c-vl.example.com
    port: 8443
    uuid: u-u-i-d
    network: ws
    tls: true
  - name: "DIRECT占位"
    type: direct
  - name: "REJECT占位"
    type: reject
proxy-groups:
  - name: PROXY
    type: select
    proxies: ["🇭🇰 Clash-Hy2"]
`

func TestParseClash(t *testing.T) {
	nodes, err := parse([]byte(clashYAML), 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("应解析出 2 个节点（跳过 direct/reject），实际 %d: %+v", len(nodes), nodes)
	}
	if nodes[0].Index != 1 || nodes[1].Index != 2 {
		t.Fatalf("Index 编号错误: %d, %d", nodes[0].Index, nodes[1].Index)
	}
	n0 := nodes[0]
	if n0.Name != "🇭🇰 Clash-Hy2" || n0.Type != "Hysteria2" || n0.Server != "c-hy2.example.com" {
		t.Fatalf("clash 节点基础字段错误: %+v", n0)
	}
	if n0.Raw["type"] != "hysteria2" || n0.Raw["port"] != 443 || n0.Raw["password"] != "pw1" {
		t.Fatalf("clash Raw 未保留原 map: %v", n0.Raw)
	}
	if nodes[1].Raw["uuid"] != "u-u-i-d" {
		t.Fatalf("clash vless uuid 错误: %v", nodes[1].Raw)
	}
}

// ---- 订阅级解析（分享链接 + 坏行跳过 + base64 包装）----

var sampleLinks = []string{
	"ss://" + b64("aes-256-gcm:p1") + "@1.1.1.1:1001#n1",
	"vmess://" + b64(`{"ps":"n2","add":"2.2.2.2","port":1002,"id":"u2","aid":0,"net":"tcp","tls":""}`),
	"vless://u3@3.3.3.3:1003?security=reality&flow=xtls-rprx-vision&pbk=K&sid=s#n3",
	"trojan://p4@4.4.4.4:1004#n4",
	"hysteria2://p5@5.5.5.5:1005?obfs=salamander&obfs-password=o5#n5",
	"tuic://u6:p6@6.6.6.6:1006#n6",
	"snell://p7@7.7.7.7:1007?version=4#n7",
	"ssr://" + b64("8.8.8.8:1008:origin:aes-256-cfb:plain:"+b64("p8")+"/?remarks="+b64("n8")),
}

func TestParseShareLinkSubscription(t *testing.T) {
	content := strings.Join(sampleLinks, "\n") + "\nnot-a-link garbage line\nss://bad\n"
	nodes, err := parse([]byte(content), 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(nodes) != len(sampleLinks) {
		t.Fatalf("应解析出 %d 个节点，实际 %d", len(sampleLinks), len(nodes))
	}
	for i, n := range nodes {
		if n.Index != i+1 {
			t.Fatalf("Index 编号错误: 第 %d 个节点 Index=%d", i, n.Index)
		}
		if n.Name == "" || n.Server == "" || n.Raw["type"] == "" || n.Raw["port"] == nil {
			t.Fatalf("节点 %d 字段不全: %+v", i, n)
		}
	}
}

func TestParseBase64WrappedSubscription(t *testing.T) {
	content := b64std(strings.Join(sampleLinks, "\n"))
	nodes, err := parse([]byte(content), 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(nodes) != len(sampleLinks) {
		t.Fatalf("base64 订阅应解析出 %d 个节点，实际 %d", len(sampleLinks), len(nodes))
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()

	// 明文分享链接文件
	linkFile := filepath.Join(dir, "sub.txt")
	if err := os.WriteFile(linkFile, []byte(strings.Join(sampleLinks, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	nodes, name, err := Load(linkFile)
	if err != nil {
		t.Fatalf("Load 分享链接文件: %v", err)
	}
	if len(nodes) != len(sampleLinks) {
		t.Fatalf("Load 应返回 %d 个节点，实际 %d", len(sampleLinks), len(nodes))
	}
	if name != "sub" {
		t.Fatalf("分享链接文件名应提取为 %q, 实际 %q", "sub", name)
	}

	// Clash YAML 文件
	yamlFile := filepath.Join(dir, "clash.yaml")
	if err := os.WriteFile(yamlFile, []byte(clashYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	nodes, name, err = Load(yamlFile)
	if err != nil {
		t.Fatalf("Load clash 文件: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("clash 文件应返回 2 个节点，实际 %d", len(nodes))
	}
	if name != "clash" {
		t.Fatalf("clash 文件名应提取为 %q, 实际 %q", "clash", name)
	}
}

func TestLoadBadSource(t *testing.T) {
	if _, _, err := Load(filepath.Join(t.TempDir(), "nonexistent.yaml")); err == nil {
		t.Fatal("读取不存在的文件应报错")
	}
	if _, err := parse([]byte("hello world, not a subscription"), 0); err == nil {
		t.Fatal("无法识别的内容应报错")
	}
}

func TestParseShareLinkErrors(t *testing.T) {
	bad := []string{
		"ss://",                      // 空
		"ss://!!!notbase64!!!",       // 解码失败
		"ssr://aGVsbG8",              // 解码后字段不足
		"vmess://aGVsbG8",            // 非 JSON
		"vless://@:0",                // 无效端口
		"hysteria2://p@h:notaport#x", // 无效端口
		"tuic://u:p@h",               // 缺端口
		"snell://p@h:abc",            // 无效端口
		"unknown://whatever",         // 不支持的协议
	}
	for _, link := range bad {
		if _, err := ParseShareLink(link); err == nil {
			t.Errorf("链接 %q 应解析失败", link)
		}
	}
}

func ExampleParseShareLink() {
	n, _ := ParseShareLink("trojan://pw@example.com:443?sni=example.com#demo")
	fmt.Println(n.Type, n.Server, n.Raw["port"], n.Raw["password"])
	// Output: Trojan example.com 443 pw
}

func TestSanitizeName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"一分机场", "一分机场"},
		{"  机场名称  ", "机场名称"},
		{"sub.txt", "sub"},
		{"clash.yaml", "clash"},
		{"name/with/slash", "name_with_slash"},
		{`name:with|illegal*chars?`, "name_with_illegal_chars"},
		{"a_b__c", "a_b_c"},
	}
	for _, c := range cases {
		got := sanitizeName(c.in)
		if got != c.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractNameFromContent(t *testing.T) {
	cases := []struct{ in, want string }{
		{"# 一分机场\nvmess://...", "一分机场"},
		{"# name: 测试订阅\nss://...", "测试订阅"},
		{"# 机场：GongHaiLin\n...", "GongHaiLin"},
		{"ss://...\n", ""},
		{"\n\n# 单纯注释\n...", "单纯注释"},
	}
	for _, c := range cases {
		got := extractNameFromContent([]byte(c.in))
		if got != c.want {
			t.Errorf("extractNameFromContent(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractName(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Disposition", `attachment; filename="abc.yaml"`)
	cases := []struct {
		source string
		data   []byte
		header http.Header
		want   string
	}{
		{"/path/to/一分机场.yaml", nil, nil, "一分机场"},
		{"https://example.com/sub/xyz?title=MySub", nil, nil, "MySub"},
		{"https://example.com/sub/xyz", []byte("# 机场名\nss://..."), nil, "机场名"},
		{"https://example.com/sub/xyz", nil, h, "abc"},
		{"https://example.com/sub/xyz", nil, nil, "xyz"},
	}
	for _, c := range cases {
		got := extractName(c.source, c.data, c.header)
		if got != c.want {
			t.Errorf("extractName(%q, %q, %v) = %q, want %q", c.source, c.data, c.header, got, c.want)
		}
	}
}

// ---- 拉取环节：压缩响应与代理 ----

const testClashYAML = `proxies:
  - { name: 'test-ss', type: ss, server: 1.2.3.4, port: 8388, cipher: aes-256-gcm, password: pass123 }
`

func TestLoadGzipResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write([]byte(testClashYAML)); err != nil {
			t.Errorf("gzip write: %v", err)
		}
		if err := gw.Close(); err != nil {
			t.Errorf("gzip close: %v", err)
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	nodes, _, err := Load(srv.URL + "/sub")
	if err != nil {
		t.Fatalf("Load gzip 订阅失败: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Name != "test-ss" {
		t.Fatalf("gzip 订阅解析节点错误: %+v", nodes)
	}
}

func TestLoadDeflateResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		if _, err := zw.Write([]byte(testClashYAML)); err != nil {
			t.Errorf("zlib write: %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Errorf("zlib close: %v", err)
		}
		w.Header().Set("Content-Encoding", "deflate")
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	nodes, _, err := Load(srv.URL + "/sub")
	if err != nil {
		t.Fatalf("Load deflate 订阅失败: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Name != "test-ss" {
		t.Fatalf("deflate 订阅解析节点错误: %+v", nodes)
	}
}

func TestLoadViaProxy(t *testing.T) {
	var sawProxyRequest bool
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawProxyRequest = true
		if !r.URL.IsAbs() {
			t.Errorf("代理应收到绝对 URL 请求，实际: %s", r.URL)
		}
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte(testClashYAML))
	}))
	defer proxySrv.Close()

	if err := SetProxy(proxySrv.URL); err != nil {
		t.Fatalf("SetProxy: %v", err)
	}
	defer func() {
		if err := SetProxy(""); err != nil {
			t.Fatalf("还原代理设置: %v", err)
		}
	}()

	// 目标域名不存在，只有经过代理才能"拉取成功"
	nodes, _, err := Load("http://sub.invalid/t")
	if err != nil {
		t.Fatalf("经代理 Load 失败: %v", err)
	}
	if !sawProxyRequest {
		t.Fatal("请求未经过代理服务器")
	}
	if len(nodes) != 1 || nodes[0].Name != "test-ss" {
		t.Fatalf("代理订阅解析节点错误: %+v", nodes)
	}
}

func TestSetProxyInvalid(t *testing.T) {
	if err := SetProxy("ftp://127.0.0.1:21"); err == nil {
		t.Fatal("不支持的代理协议应报错")
	}
	if err := SetProxy("http://127.0.0.1:7890"); err != nil {
		t.Fatalf("合法代理地址不应报错: %v", err)
	}
	if err := SetProxy(""); err != nil {
		t.Fatalf("恢复默认代理不应报错: %v", err)
	}
}
