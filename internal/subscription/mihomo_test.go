package subscription

import (
	"encoding/base64"
	"testing"

	"github.com/metacubex/mihomo/adapter"
)

// TestMihomoParseProxy 验证解析产物能被 mihomo v1.18 adapter.ParseProxy 直接消费，
// 即字段命名与 clash 配置严格对齐。
func TestMihomoParseProxy(t *testing.T) {
	uuid := "11111111-2222-3333-4444-555555555555"
	// reality public-key 为 32 字节的 RawURL base64（43 字符）
	pbk := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	links := []string{
		"ss://" + b64("aes-256-gcm:p1") + "@1.1.1.1:1001#ss",
		"ssr://" + b64("8.8.8.8:1008:origin:aes-256-cfb:plain:"+b64("p8")+"/?remarks="+b64("ssr")),
		"vmess://" + b64(`{"ps":"vmess","add":"2.2.2.2","port":1002,"id":"`+uuid+`","aid":0,"scy":"auto","net":"ws","host":"cdn.example.com","path":"/ws","tls":"tls","sni":"s.example.com"}`),
		"vless://" + uuid + "@3.3.3.3:443?encryption=none&security=reality&flow=xtls-rprx-vision&pbk=" + pbk + "&sid=ab12&fp=chrome&sni=www.microsoft.com#vless",
		"vless://" + uuid + "@3.3.3.3:443?security=tls&type=ws&path=/ray&host=h.example.com#vless-ws",
		"trojan://p4@4.4.4.4:443?sni=t.example.com#trojan",
		"hysteria2://p5@5.5.5.5:1005?sni=h.example.com&obfs=salamander&obfs-password=o5#hy2",
		"tuic://" + uuid + ":p6@6.6.6.6:1006?congestion_control=bbr&udp_relay_mode=native&alpn=h3#tuic",
		"snell://p7@7.7.7.7:1007?version=3#snell",
	}
	for _, link := range links {
		n, err := ParseShareLink(link)
		if err != nil {
			t.Fatalf("ParseShareLink(%q): %v", link, err)
		}
		proxy, err := adapter.ParseProxy(n.Raw)
		if err != nil {
			t.Errorf("mihomo ParseProxy 拒绝 %q 的解析产物: %v\nraw = %v", link, err, n.Raw)
			continue
		}
		if proxy.Name() != n.Name {
			t.Errorf("mihomo 节点名 %q 与订阅名 %q 不一致", proxy.Name(), n.Name)
		}
	}
}

// TestMihomoParseClash 验证 clash YAML 解析产物同样能被 mihomo 消费。
func TestMihomoParseClash(t *testing.T) {
	yamlText := `proxies:
  - name: hy2
    type: hysteria2
    server: h.example.com
    port: 443
    password: pw
    sni: h.example.com
  - name: vless
    type: vless
    server: v.example.com
    port: 443
    uuid: 11111111-2222-3333-4444-555555555555
    network: ws
    tls: true
    ws-opts:
      path: /ray
`
	nodes, err := parse([]byte(yamlText), 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, n := range nodes {
		if _, err := adapter.ParseProxy(n.Raw); err != nil {
			t.Errorf("mihomo ParseProxy 拒绝 clash 节点 %q: %v", n.Name, err)
		}
	}
}
