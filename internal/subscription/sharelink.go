package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"nekospeed/internal/model"
)

// typeDisplayNames clash 协议 type → 展示名。
var typeDisplayNames = map[string]string{
	"ss":        "Shadowsocks",
	"ssr":       "ShadowsocksR",
	"vmess":     "Vmess",
	"vless":     "Vless",
	"trojan":    "Trojan",
	"hysteria2": "Hysteria2",
	"tuic":      "Tuic",
	"snell":     "Snell",
}

// shareSchemes 支持的分享链接 scheme（hy2 是 hysteria2 的别名）。
var shareSchemes = map[string]string{
	"ss":        "ss",
	"ssr":       "ssr",
	"vmess":     "vmess",
	"vless":     "vless",
	"trojan":    "trojan",
	"hysteria2": "hysteria2",
	"hy2":       "hysteria2",
	"tuic":      "tuic",
	"snell":     "snell",
}

// isShareLink 判断一行是否以支持的协议 scheme 开头。
func isShareLink(line string) bool {
	scheme, _, ok := strings.Cut(line, "://")
	if !ok {
		return false
	}
	_, supported := shareSchemes[strings.ToLower(scheme)]
	return supported
}

// ParseShareLink 把一条分享链接解析为 model.Node（Index 由调用方填充）。
// Raw 是 clash 单节点 map，字段名与 mihomo adapter.ParseProxy 对齐。
func ParseShareLink(line string) (model.Node, error) {
	scheme, body, _ := strings.Cut(strings.TrimSpace(line), "://")
	typ, ok := shareSchemes[strings.ToLower(scheme)]
	if !ok {
		return model.Node{}, fmt.Errorf("不支持的协议 %q", scheme)
	}

	var (
		node model.Node
		err  error
	)
	switch typ {
	case "ss":
		node, err = parseSS(body)
	case "ssr":
		node, err = parseSSR(body)
	case "vmess":
		node, err = parseVmess(body)
	case "vless", "trojan", "hysteria2", "tuic", "snell":
		node, err = parseURLLink(typ, line)
	}
	if err != nil {
		return model.Node{}, err
	}
	if node.Name == "" {
		node.Name = node.Server
		node.Raw["name"] = node.Name
	}
	return node, nil
}

// base64DecodeAny 尝试用常见 base64 编码解码 s（标准 / RawURL / URL / RawStd）。
func base64DecodeAny(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.StdEncoding,
		base64.RawStdEncoding,
	}
	var lastErr error
	for _, enc := range encodings {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		} else {
			lastErr = err
		}
	}
	return nil, lastErr
}

// splitFragment 拆出 #fragment 作为节点名（URL 解码），返回主体与名称。
func splitFragment(body string) (rest, name string) {
	if i := strings.Index(body, "#"); i >= 0 {
		if n, err := url.PathUnescape(body[i+1:]); err == nil {
			name = n
		} else {
			name = body[i+1:]
		}
		body = body[:i]
	}
	return body, name
}

// splitHostPort 拆 host:port，port 必须是数字。
func splitHostPort(hostport string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", 0, fmt.Errorf("无效的地址 %q: %w", hostport, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("无效的端口 %q", portStr)
	}
	return strings.Trim(host, "[]"), port, nil
}

// newNode 组装 Node，raw 里已含 name/type/server/port 及协议字段。
func newNode(name, typ, server string, raw map[string]any) model.Node {
	return model.Node{
		Name:   name,
		Type:   displayType(typ),
		Server: server,
		Raw:    raw,
	}
}

// parseSS 解析 ss:// 链接。
// SIP002: ss://base64(method:password)@host:port#name 或 ss://method:password@host:port#name
// 旧式:   ss://base64(method:password@host:port)#name
func parseSS(body string) (model.Node, error) {
	body, name := splitFragment(body)
	// 去掉 ?plugin=... 等查询参数（插件参数不映射到 clash 字段，忽略）
	if i := strings.Index(body, "?"); i >= 0 {
		body = body[:i]
	}

	var userinfo, hostport string
	if i := strings.LastIndex(body, "@"); i >= 0 {
		userinfo, hostport = body[:i], body[i+1:]
		// userinfo 可能是 base64(method:password)，也可能是明文（可能带 URL 转义）
		if dec, err := base64DecodeAny(userinfo); err == nil && strings.Contains(string(dec), ":") {
			userinfo = string(dec)
		} else if u, err := url.PathUnescape(userinfo); err == nil {
			userinfo = u
		}
	} else {
		// 旧式：整体 base64
		dec, err := base64DecodeAny(body)
		if err != nil {
			return model.Node{}, fmt.Errorf("ss 链接 base64 解码失败: %w", err)
		}
		s := string(dec)
		i := strings.LastIndex(s, "@")
		if i < 0 {
			return model.Node{}, fmt.Errorf("ss 链接缺少 @ 分隔: %q", s)
		}
		userinfo, hostport = s[:i], s[i+1:]
	}

	method, password, ok := strings.Cut(userinfo, ":")
	if !ok || method == "" {
		return model.Node{}, fmt.Errorf("ss 链接缺少 method:password: %q", userinfo)
	}
	host, port, err := splitHostPort(hostport)
	if err != nil {
		return model.Node{}, err
	}
	raw := map[string]any{
		"name": name, "type": "ss", "server": host, "port": port,
		"cipher": method, "password": password, "udp": true,
	}
	return newNode(name, "ss", host, raw), nil
}

// parseSSR 解析 ssr:// 链接：整体 base64 编码的
// host:port:protocol:method:obfs:base64pass/?obfsparam=b64&protoparam=b64&remarks=b64
func parseSSR(body string) (model.Node, error) {
	dec, err := base64DecodeAny(body)
	if err != nil {
		return model.Node{}, fmt.Errorf("ssr 链接 base64 解码失败: %w", err)
	}
	s := string(dec)

	mainPart, paramStr := s, ""
	if i := strings.Index(s, "/?"); i >= 0 {
		mainPart, paramStr = s[:i], s[i+2:]
	}

	parts := strings.Split(mainPart, ":")
	if len(parts) < 6 {
		return model.Node{}, fmt.Errorf("ssr 链接字段不足: %q", mainPart)
	}
	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return model.Node{}, fmt.Errorf("ssr 链接端口无效: %q", parts[1])
	}
	protocol, method, obfs := parts[2], parts[3], parts[4]
	// 密码段本身是 base64（理论上不含冒号，稳妥起见把剩余段拼回去）
	password, _ := base64DecodeAny(strings.Join(parts[5:], ":"))

	params := map[string]string{}
	for _, kv := range strings.Split(paramStr, "&") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			continue
		}
		if decV, err := base64DecodeAny(v); err == nil {
			params[k] = string(decV)
		} else {
			params[k] = v
		}
	}

	name := params["remarks"]
	raw := map[string]any{
		"name": name, "type": "ssr", "server": host, "port": port,
		"cipher": method, "password": string(password),
		"protocol": protocol, "obfs": obfs, "udp": true,
	}
	if v := params["obfsparam"]; v != "" {
		raw["obfs-param"] = v
	}
	if v := params["protoparam"]; v != "" {
		raw["protocol-param"] = v
	}
	return newNode(name, "ssr", host, raw), nil
}

// parseVmess 解析 vmess:// 链接：base64 编码的 JSON。
func parseVmess(body string) (model.Node, error) {
	dec, err := base64DecodeAny(strings.TrimSpace(body))
	if err != nil {
		return model.Node{}, fmt.Errorf("vmess 链接 base64 解码失败: %w", err)
	}
	var v map[string]any
	if err := json.Unmarshal(dec, &v); err != nil {
		return model.Node{}, fmt.Errorf("vmess 链接 JSON 解析失败: %w", err)
	}

	name := jsonStr(v, "ps")
	host := jsonStr(v, "add")
	if host == "" {
		return model.Node{}, fmt.Errorf("vmess 链接缺少 add 字段")
	}
	port := jsonInt(v, "port")
	if port == 0 {
		return model.Node{}, fmt.Errorf("vmess 链接端口无效: %v", v["port"])
	}
	cipher := jsonStr(v, "scy")
	if cipher == "" {
		cipher = "auto"
	}

	raw := map[string]any{
		"name": name, "type": "vmess", "server": host, "port": port,
		"uuid": jsonStr(v, "id"), "alterId": jsonInt(v, "aid"),
		"cipher": cipher, "udp": true,
	}
	if tlsv := jsonStr(v, "tls"); tlsv == "tls" || tlsv == "reality" {
		raw["tls"] = true
	}
	if sni := jsonStr(v, "sni"); sni != "" {
		raw["servername"] = sni
	}
	if alpn := jsonStr(v, "alpn"); alpn != "" {
		raw["alpn"] = splitComma(alpn)
	}

	network := jsonStr(v, "net")
	switch network {
	case "ws":
		raw["network"] = "ws"
		wsOpts := map[string]any{}
		if p := jsonStr(v, "path"); p != "" {
			wsOpts["path"] = p
		}
		if h := jsonStr(v, "host"); h != "" {
			wsOpts["headers"] = map[string]any{"Host": h}
		}
		if len(wsOpts) > 0 {
			raw["ws-opts"] = wsOpts
		}
	case "grpc":
		raw["network"] = "grpc"
		if p := jsonStr(v, "path"); p != "" {
			raw["grpc-opts"] = map[string]any{"grpc-service-name": p}
		}
	case "h2", "http":
		raw["network"] = "h2"
		h2Opts := map[string]any{}
		if h := jsonStr(v, "host"); h != "" {
			h2Opts["host"] = splitComma(h)
		}
		if p := jsonStr(v, "path"); p != "" {
			h2Opts["path"] = p
		}
		if len(h2Opts) > 0 {
			raw["h2-opts"] = h2Opts
		}
	case "", "tcp":
		// 默认 tcp，无需设置 network
	default:
		raw["network"] = network
	}
	return newNode(name, "vmess", host, raw), nil
}

// parseURLLink 解析 URL 形式的分享链接（vless/trojan/hysteria2/tuic/snell）。
func parseURLLink(typ, line string) (model.Node, error) {
	u, err := url.Parse(line)
	if err != nil {
		return model.Node{}, fmt.Errorf("%s 链接 URL 解析失败: %w", typ, err)
	}
	host := u.Hostname()
	if host == "" {
		return model.Node{}, fmt.Errorf("%s 链接缺少服务器地址", typ)
	}
	portStr := u.Port()
	q := u.Query()

	raw := map[string]any{
		"name": u.Fragment, "type": typ, "server": host,
	}
	// hysteria2 支持端口段（如 443 或留空走 ports），其余协议端口必须是数字
	if port, err := strconv.Atoi(portStr); err == nil {
		raw["port"] = port
	} else if typ == "hysteria2" && portStr != "" {
		raw["ports"] = portStr
	} else {
		return model.Node{}, fmt.Errorf("%s 链接端口无效: %q", typ, portStr)
	}

	switch typ {
	case "vless":
		fillVless(raw, u, q)
	case "trojan":
		fillTrojan(raw, u, q)
	case "hysteria2":
		fillHysteria2(raw, u, q)
	case "tuic":
		fillTuic(raw, u, q)
	case "snell":
		fillSnell(raw, u, q)
	}
	return newNode(u.Fragment, typ, host, raw), nil
}

// fillVless 填充 vless 字段：uuid/flow/tls/reality-opts/network 等。
func fillVless(raw map[string]any, u *url.URL, q url.Values) {
	if u.User != nil {
		raw["uuid"] = u.User.Username()
	}
	if flow := q.Get("flow"); flow != "" {
		raw["flow"] = flow
	}
	security := q.Get("security")
	if security == "tls" || security == "reality" {
		raw["tls"] = true
	}
	if security == "reality" {
		reality := map[string]any{}
		if pbk := q.Get("pbk"); pbk != "" {
			reality["public-key"] = pbk
		}
		if sid := q.Get("sid"); sid != "" {
			reality["short-id"] = sid
		}
		raw["reality-opts"] = reality
	}
	if sni := q.Get("sni"); sni != "" {
		raw["servername"] = sni
	}
	if fp := q.Get("fp"); fp != "" {
		raw["client-fingerprint"] = fp
	}
	if boolish(q.Get("allowInsecure")) || boolish(q.Get("allow_insecure")) {
		raw["skip-cert-verify"] = true
	}
	if alpn := q.Get("alpn"); alpn != "" {
		raw["alpn"] = splitComma(alpn)
	}
	raw["udp"] = true
	fillNetwork(raw, q)
}

// fillTrojan 填充 trojan 字段：password/sni/alpn/network 等。
func fillTrojan(raw map[string]any, u *url.URL, q url.Values) {
	if u.User != nil {
		raw["password"] = u.User.Username()
	}
	if sni := q.Get("sni"); sni != "" {
		raw["sni"] = sni
	} else if peer := q.Get("peer"); peer != "" {
		raw["sni"] = peer
	}
	if boolish(q.Get("allowInsecure")) || boolish(q.Get("allow_insecure")) {
		raw["skip-cert-verify"] = true
	}
	if alpn := q.Get("alpn"); alpn != "" {
		raw["alpn"] = splitComma(alpn)
	}
	raw["udp"] = true
	fillNetwork(raw, q)
}

// fillHysteria2 填充 hysteria2 字段：password/sni/obfs/obfs-password 等。
func fillHysteria2(raw map[string]any, u *url.URL, q url.Values) {
	if u.User != nil {
		// userinfo 形如 password 或 auth:password，mihomo 只需要密码本体
		user := u.User.Username()
		if pwd, ok := u.User.Password(); ok {
			user = user + ":" + pwd
		}
		raw["password"] = user
	}
	if sni := q.Get("sni"); sni != "" {
		raw["sni"] = sni
	}
	if boolish(q.Get("insecure")) {
		raw["skip-cert-verify"] = true
	}
	if obfs := q.Get("obfs"); obfs != "" {
		raw["obfs"] = obfs
		if obfsPass := q.Get("obfs-password"); obfsPass != "" {
			raw["obfs-password"] = obfsPass
		}
	}
	if alpn := q.Get("alpn"); alpn != "" {
		raw["alpn"] = splitComma(alpn)
	}
}

// fillTuic 填充 tuic 字段：uuid/password/congestion-controller/udp-relay-mode 等。
func fillTuic(raw map[string]any, u *url.URL, q url.Values) {
	if u.User != nil {
		raw["uuid"] = u.User.Username()
		if pwd, ok := u.User.Password(); ok {
			raw["password"] = pwd
		}
	}
	if sni := q.Get("sni"); sni != "" {
		raw["sni"] = sni
	}
	if cc := q.Get("congestion_control"); cc != "" {
		raw["congestion-controller"] = cc
	}
	if mode := q.Get("udp_relay_mode"); mode != "" {
		raw["udp-relay-mode"] = mode
	}
	if alpn := q.Get("alpn"); alpn != "" {
		raw["alpn"] = splitComma(alpn)
	}
	if boolish(q.Get("allow_insecure")) || boolish(q.Get("allowInsecure")) {
		raw["skip-cert-verify"] = true
	}
	if boolish(q.Get("disable_sni")) {
		raw["disable-sni"] = true
	}
}

// fillSnell 填充 snell 字段：psk/version。
func fillSnell(raw map[string]any, u *url.URL, q url.Values) {
	if u.User != nil {
		raw["psk"] = u.User.Username()
	}
	if v := q.Get("version"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			raw["version"] = n
		}
	}
}

// fillNetwork 把 URL query 的 type=ws/grpc 等传输层参数映射为 clash 字段。
func fillNetwork(raw map[string]any, q url.Values) {
	switch q.Get("type") {
	case "ws":
		raw["network"] = "ws"
		wsOpts := map[string]any{}
		if p := q.Get("path"); p != "" {
			wsOpts["path"] = p
		}
		if h := q.Get("host"); h != "" {
			wsOpts["headers"] = map[string]any{"Host": h}
		}
		if len(wsOpts) > 0 {
			raw["ws-opts"] = wsOpts
		}
	case "grpc":
		raw["network"] = "grpc"
		grpcOpts := map[string]any{}
		if sn := q.Get("serviceName"); sn != "" {
			grpcOpts["grpc-service-name"] = sn
		}
		if len(grpcOpts) > 0 {
			raw["grpc-opts"] = grpcOpts
		}
	case "http", "h2":
		raw["network"] = "h2"
	case "tcp", "":
		// 默认 tcp，无需设置 network
	default:
		raw["network"] = q.Get("type")
	}
}

// jsonStr 取 JSON map 的字符串字段。
func jsonStr(m map[string]any, key string) string {
	switch v := m[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

// jsonInt 取 JSON map 的整数字段（兼容数字与字符串）。
func jsonInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

// splitComma 按逗号拆分并去掉空项。
func splitComma(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// boolish 判断 query 参数是否为真值。
func boolish(s string) bool {
	return s == "1" || s == "true" || s == "yes"
}
