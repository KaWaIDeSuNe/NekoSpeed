// Package geo 提供节点入口/出口 IP 归属分析（GeoIP）。
// 参考 miaospeed 的 GEOIP_INBOUND/GEOIP_OUTBOUND 思路：
// 入口直连解析节点服务器 IP 再查归属；出口经代理访问在线 API 拿落地 IP 再补全归属。
// 查询后端二选一：本地 MaxMind mmdb（GeoLite2-City 或 -ASN）或在线 API（ip-api.com）。
package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"nekospeed/internal/model"
)

// 在线 GeoIP API 端点。定义为包级变量以便测试替换为 httptest 服务器。
var (
	// ipAPIBaseURL ip-api.com 基础地址。走代理时强制 http，避免额外证书问题。
	ipAPIBaseURL = "http://ip-api.com"
	// ipinfoURL ipinfo.io 兜底端点（出口查询失败时使用）。
	ipinfoURL = "https://ipinfo.io/json"
)

const ipAPIFields = "status,countryCode,regionName,city,query,as,org,isp"

// Resolver 封装 GeoIP 查询后端，带 IP 级缓存与并发去重。
type Resolver struct {
	mmdb *maxminddb.Reader // nil 表示使用在线 API

	mu       sync.Mutex
	cache    map[string]model.GeoInfo // key = IP
	inflight map[string]*lookupCall   // 进行中的查询，相同 IP 只发一次请求
}

type lookupCall struct {
	done chan struct{}
	info model.GeoInfo
}

// NewResolver 创建 Resolver。mmdbPath 非空时打开 MaxMind mmdb（GeoLite2-City 或
// GeoLite2-ASN，查不到的字段优雅留空）；为空时使用在线 API。
func NewResolver(mmdbPath string) (*Resolver, error) {
	r := &Resolver{
		cache:    make(map[string]model.GeoInfo),
		inflight: make(map[string]*lookupCall),
	}
	if mmdbPath != "" {
		db, err := maxminddb.Open(mmdbPath)
		if err != nil {
			return nil, fmt.Errorf("打开 mmdb: %w", err)
		}
		r.mmdb = db
	}
	return r, nil
}

// Close 释放 mmdb 句柄（在线模式下为空操作）。
func (r *Resolver) Close() error {
	if r.mmdb != nil {
		return r.mmdb.Close()
	}
	return nil
}

// Lookup 查询 ip 的归属信息。Stack 按 IP 是 v4/v6 填 "4"/"6"。
// d 非 nil 时经代理发起 API 请求（仅在线模式；mmdb 模式查本地库，忽略 d）。
// 查询失败返回仅含 IP/Stack 的降级结果，同一 IP 的结果会被缓存去重。
func (r *Resolver) Lookup(ctx context.Context, ip string, d model.Dialer) model.GeoInfo {
	if ip == "" {
		return model.GeoInfo{}
	}
	r.mu.Lock()
	if g, ok := r.cache[ip]; ok {
		r.mu.Unlock()
		return g
	}
	if c, ok := r.inflight[ip]; ok {
		r.mu.Unlock()
		select {
		case <-c.done:
			return c.info
		case <-ctx.Done():
			return model.GeoInfo{IP: ip, Stack: stackOf(ip)}
		}
	}
	c := &lookupCall{done: make(chan struct{})}
	r.inflight[ip] = c
	r.mu.Unlock()

	g := r.doLookup(ctx, ip, d)

	r.mu.Lock()
	r.cache[ip] = g
	delete(r.inflight, ip)
	c.info = g
	r.mu.Unlock()
	close(c.done)
	return g
}

func (r *Resolver) doLookup(ctx context.Context, ip string, d model.Dialer) model.GeoInfo {
	if r.mmdb != nil {
		return r.lookupMMDB(ip)
	}
	if g, err := r.lookupIPAPI(ctx, ip, d); err == nil {
		return g
	}
	return model.GeoInfo{IP: ip, Stack: stackOf(ip)}
}

// lookupMMDB 查本地库。City 库通常没有 ASN/Org 字段，查不到时留空。
func (r *Resolver) lookupMMDB(ip string) model.GeoInfo {
	g := model.GeoInfo{IP: ip, Stack: stackOf(ip)}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return g
	}
	var rec struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
		Subdivisions []struct {
			Names struct {
				En string `maxminddb:"en"`
			} `maxminddb:"names"`
		} `maxminddb:"subdivisions"`
		City struct {
			Names struct {
				En string `maxminddb:"en"`
			} `maxminddb:"names"`
		} `maxminddb:"city"`
		AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
		ISP                          string `maxminddb:"isp"`
		Organization                 string `maxminddb:"organization"`
	}
	if err := r.mmdb.Lookup(parsed, &rec); err != nil {
		return g
	}
	g.Country = rec.Country.ISOCode
	if len(rec.Subdivisions) > 0 {
		g.Region = rec.Subdivisions[0].Names.En
	}
	g.City = rec.City.Names.En
	if rec.AutonomousSystemNumber != 0 {
		g.ASN = fmt.Sprintf("AS%d", rec.AutonomousSystemNumber)
	}
	if rec.ISP != "" {
		g.ISP = rec.ISP
	} else if rec.Organization != "" {
		g.ISP = rec.Organization
	} else {
		g.ISP = rec.AutonomousSystemOrganization
	}
	g.Org = rec.AutonomousSystemOrganization
	if g.Org == "" {
		g.Org = rec.Organization
	}
	return g
}

// LookupLocalNetwork 查询本机公网 IP 的归属（不经过代理）。
// 用于在测速图标题等位置标注本地运营商，如 "Shanghai China Telecom"。
// 查询失败时返回空的 GeoInfo，调用方应降级处理。
func (r *Resolver) LookupLocalNetwork(ctx context.Context) model.GeoInfo {
	if g, err := r.lookupIPAPI(ctx, "", nil); err == nil && g.IP != "" {
		return r.putCache(g)
	}
	// 回退 ipinfo.io（无代理直连）。
	if g, err := r.lookupIPInfo(ctx, nil); err == nil && g.IP != "" {
		return r.putCache(g)
	}
	return model.GeoInfo{}
}

// ResolveNode 解析节点的入口（服务器 IP，直连解析+直连查询）与
// 出口（经代理拿落地 IP 再补全归属）。任一侧失败时对应返回 nil。
func (r *Resolver) ResolveNode(ctx context.Context, node model.Node, d model.Dialer) (inbound, outbound *model.GeoInfo) {
	// 入口：直连解析 node.Server，优先 IPv4，再直连 Lookup。
	if ip := resolveServerIP(ctx, node.Server); ip != "" {
		g := r.Lookup(ctx, ip, nil)
		inbound = &g
	}
	if d == nil {
		return inbound, nil
	}
	// 出口：经代理访问 ip-api 拿落地信息；失败回退 ipinfo.io。
	if g, err := r.lookupIPAPI(ctx, "", d); err == nil {
		return inbound, r.finishOutbound(g)
	}
	if g, err := r.lookupIPInfo(ctx, d); err == nil {
		return inbound, r.finishOutbound(g)
	}
	return inbound, nil
}

// finishOutbound 出口补全：mmdb 模式下只取 API 返回的落地 IP，归属用本地库；
// 在线模式下 API 结果即最终归属，写入缓存去重。
func (r *Resolver) finishOutbound(g model.GeoInfo) *model.GeoInfo {
	if r.mmdb != nil {
		out := r.Lookup(context.Background(), g.IP, nil)
		return &out
	}
	out := r.putCache(g)
	return &out
}

// putCache 写入缓存；已存在时返回缓存中的副本，保证同一 IP 只有一份归属数据。
func (r *Resolver) putCache(g model.GeoInfo) model.GeoInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.cache[g.IP]; ok {
		return c
	}
	r.cache[g.IP] = g
	return g
}

// resolveServerIP 直连解析节点服务器地址（域名或 IP，可带端口），优先返回 IPv4。
func resolveServerIP(ctx context.Context, server string) string {
	server = strings.TrimSpace(server)
	if server == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(server); err == nil {
		server = h
	}
	if net.ParseIP(server) != nil {
		return server
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip", server)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	for _, a := range addrs {
		if a4 := a.To4(); a4 != nil {
			return a4.String()
		}
	}
	return addrs[0].String()
}

// httpClient 直连用默认客户端，走代理用 model.HTTPClient（强制 http 由 URL 决定）。
// 为避免本机 HTTP_PROXY 环境变量干扰，两种客户端都显式禁用 Proxy。
func httpClient(d model.Dialer) *http.Client {
	if d == nil {
		return &http.Client{
			Timeout:   15 * time.Second,
			Transport: &http.Transport{Proxy: nil},
		}
	}
	return model.HTTPClient(d, 15*time.Second)
}

type ipAPIResponse struct {
	Status      string `json:"status"`
	CountryCode string `json:"countryCode"`
	RegionName  string `json:"regionName"`
	City        string `json:"city"`
	Query       string `json:"query"`
	As          string `json:"as"` // 形如 "AS13335 Cloudflare Inc"
	Org         string `json:"org"`
	ISP         string `json:"isp"`
}

// lookupIPAPI 查询 ip-api.com。ip 为空时返回请求端（代理出口）自身的信息。
func (r *Resolver) lookupIPAPI(ctx context.Context, ip string, d model.Dialer) (model.GeoInfo, error) {
	url := ipAPIBaseURL + "/json/" + ip + "?fields=" + ipAPIFields + "&lang=zh-CN"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return model.GeoInfo{}, err
	}
	resp, err := httpClient(d).Do(req)
	if err != nil {
		return model.GeoInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return model.GeoInfo{}, err
	}
	var a ipAPIResponse
	if err := json.Unmarshal(body, &a); err != nil {
		return model.GeoInfo{}, err
	}
	if a.Status != "success" || a.Query == "" {
		return model.GeoInfo{}, fmt.Errorf("ip-api 查询失败: %s", a.Status)
	}
	g := model.GeoInfo{
		IP:      a.Query,
		Stack:   stackOf(a.Query),
		Country: a.CountryCode,
		Region:  a.RegionName,
		City:    a.City,
		Org:     a.Org,
		ISP:     a.ISP,
	}
	// as 字段首段为 ASN（"AS13335"），其余为组织名。
	if f := strings.Fields(a.As); len(f) > 0 {
		g.ASN = f[0]
		if g.Org == "" && len(f) > 1 {
			g.Org = strings.Join(f[1:], " ")
		}
	}
	return g, nil
}

type ipinfoResponse struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	Org     string `json:"org"` // 形如 "AS13335 Cloudflare, Inc."
}

// lookupIPInfo 查询 ipinfo.io（出口查询兜底），只能拿到请求端自身信息。
func (r *Resolver) lookupIPInfo(ctx context.Context, d model.Dialer) (model.GeoInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ipinfoURL, nil)
	if err != nil {
		return model.GeoInfo{}, err
	}
	resp, err := httpClient(d).Do(req)
	if err != nil {
		return model.GeoInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return model.GeoInfo{}, err
	}
	var a ipinfoResponse
	if err := json.Unmarshal(body, &a); err != nil {
		return model.GeoInfo{}, err
	}
	if a.IP == "" {
		return model.GeoInfo{}, fmt.Errorf("ipinfo 未返回 IP")
	}
	g := model.GeoInfo{IP: a.IP, Stack: stackOf(a.IP), Country: a.Country}
	if f := strings.Fields(a.Org); len(f) > 0 && strings.HasPrefix(f[0], "AS") {
		g.ASN = f[0]
		g.Org = strings.Join(f[1:], " ")
	} else {
		g.Org = a.Org
	}
	return g, nil
}

// stackOf 按 IP 字面量判断协议栈："4" 或 "6"。
func stackOf(ip string) string {
	if strings.Contains(ip, ":") {
		return "6"
	}
	return "4"
}
