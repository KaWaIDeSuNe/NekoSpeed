// Package nodepool 的 preflight：测试前探测节点服务器域名解析出的全部 IP，
// 并发 TCP 连接节点端口，挑出当前可达的 IP 注入 mihomo DefaultHosts，
// 避免 DDNS 轮换 IP 池中被封锁的 IP 被随机选中导致整节点测试失败。
//
// 针对"权威 DNS 每次应答只给池中一个 IP、TTL 极短"的轮换型域名，
// 单一系统解析只能看到池的一个样本；这里并行直查多个公共 DNS，
// 利用各递归缓存互相独立，一次收割池中多个 IP 再逐一探测。
package nodepool

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/metacubex/mihomo/component/resolver"
	"github.com/miekg/dns"

	"nekospeed/internal/model"
)

// PreflightTimeout 单个 IP:port 的 TCP 探测超时。
// 需短于 GFW 封 IP 的典型黑洞表现（SYN 无回应），2s 足够区分。
const PreflightTimeout = 2 * time.Second

// preflightDNServers 用于采集轮换 IP 池样本的公共递归 DNS。
// 各递归缓存独立，同一时刻可能持有池中不同 IP。
var preflightDNServers = []string{
	"223.5.5.5:53",    // 阿里
	"119.29.29.29:53", // 腾讯
	"114.114.114.114:53",
	"1.12.12.12:53", // 腾讯 DNSPod 新地址
}

// PreflightResult 汇总预检结果。
type PreflightResult struct {
	PinnedHosts  int      // 成功钉住 IP 的域名数
	FailedHosts  []string // 全部采样 IP 均不可达的域名
	SkippedHosts int      // 服务器本身为 IP 字面量而跳过的数量
}

// PreflightServers 对 nodes 中出现的服务器域名做可达性预检。
// 命中的域名写入 resolver.DefaultHosts（mihomo 解析时优先于 DNS）。
// 服务器为 IP 字面量的节点不参与。上下文取消时提前返回。
func PreflightServers(ctx context.Context, nodes []model.Node) PreflightResult {
	type hostInfo struct {
		ports map[uint16]struct{}
	}
	hosts := map[string]*hostInfo{}
	var order []string
	res := PreflightResult{}

	for _, n := range nodes {
		server := n.Server
		if server == "" {
			continue
		}
		if _, err := netip.ParseAddr(server); err == nil {
			res.SkippedHosts++
			continue // 已是 IP，无需预检
		}
		h, ok := hosts[server]
		if !ok {
			h = &hostInfo{ports: map[uint16]struct{}{}}
			hosts[server] = h
			order = append(order, server)
		}
		if p := rawPort(n.Raw); p != 0 {
			h.ports[p] = struct{}{}
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, host := range order {
		info := hosts[host]
		wg.Add(1)
		go func(host string, ports map[uint16]struct{}) {
			defer wg.Done()
			good := probeHost(ctx, host, ports)
			mu.Lock()
			defer mu.Unlock()
			if len(good) == 0 {
				res.FailedHosts = append(res.FailedHosts, host)
				return
			}
			hv, err := resolver.NewHostValueByIPs(good)
			if err != nil {
				res.FailedHosts = append(res.FailedHosts, host)
				return
			}
			if err := resolver.DefaultHosts.Insert(host, hv); err != nil {
				res.FailedHosts = append(res.FailedHosts, host)
				return
			}
			res.PinnedHosts++
		}(host, info.ports)
	}
	wg.Wait()
	sort.Strings(res.FailedHosts)
	return res
}

// probeHost 采集 host 的多个 IP 池样本并探测可达性，返回活 IP 列表。
// 第一轮：多公共 DNS 并行采样 + 系统解析；全部探测失败时等待缓存过期后第二轮。
func probeHost(ctx context.Context, host string, ports map[uint16]struct{}) []netip.Addr {
	portList := make([]uint16, 0, len(ports))
	for p := range ports {
		portList = append(portList, p)
	}
	if len(portList) == 0 {
		return nil
	}

	for round := 0; round < 2; round++ {
		if round > 0 {
			// 等递归缓存（TTL 通常 5s）过期后再采样一轮
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(6 * time.Second):
			}
		}
		ips := sampleIPs(ctx, host)
		if len(ips) == 0 {
			continue
		}
		if good := probeIPs(ctx, ips, portList); len(good) > 0 {
			return good
		}
	}
	return nil
}

// sampleIPs 通过多个公共 DNS 与系统解析并行采集 host 的 A 记录（去重）。
func sampleIPs(ctx context.Context, host string) []netip.Addr {
	var mu sync.Mutex
	seen := map[netip.Addr]struct{}{}
	var wg sync.WaitGroup

	add := func(a netip.Addr) {
		a = a.Unmap()
		if !a.Is4() {
			return
		}
		mu.Lock()
		seen[a] = struct{}{}
		mu.Unlock()
	}

	for _, srv := range preflightDNServers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()
			for _, ip := range queryA(ctx, server, host) {
				add(ip)
			}
		}(srv)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		lctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if ips, err := net.DefaultResolver.LookupIP(lctx, "ip4", host); err == nil {
			for _, ip := range ips {
				if a, ok := netip.AddrFromSlice(ip); ok {
					add(a)
				}
			}
		}
	}()
	wg.Wait()

	out := make([]netip.Addr, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	return out
}

// queryA 向指定 DNS 服务器查询 host 的 A 记录（含 CNAME 链末端记录）。
func queryA(ctx context.Context, server, host string) []netip.Addr {
	c := &dns.Client{Timeout: 3 * time.Second}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)
	m.RecursionDesired = true

	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	r, _, err := c.ExchangeContext(rctx, m, server)
	if err != nil || r == nil {
		return nil
	}
	var out []netip.Addr
	for _, rr := range r.Answer {
		if a, ok := rr.(*dns.A); ok {
			if addr, ok2 := netip.AddrFromSlice(a.A); ok2 {
				out = append(out, addr)
			}
		}
	}
	return out
}

// probeIPs 并发 TCP 探测 ips 的全部端口，返回通过所有端口探测的 IP（按输入顺序）。
func probeIPs(ctx context.Context, ips []netip.Addr, portList []uint16) []netip.Addr {
	type result struct {
		ip netip.Addr
		ok bool
	}
	results := make([]result, len(ips))
	var wg sync.WaitGroup
	for i, ip := range ips {
		wg.Add(1)
		go func(i int, ip netip.Addr) {
			defer wg.Done()
			ok := true
			for _, p := range portList {
				if !tcpProbe(ctx, ip, p) {
					ok = false
					break
				}
			}
			results[i] = result{ip: ip, ok: ok}
		}(i, ip)
	}
	wg.Wait()

	var good []netip.Addr
	for _, r := range results {
		if r.ok {
			good = append(good, r.ip)
		}
	}
	return good
}

// tcpProbe 以 PreflightTimeout 尝试 TCP 连接 ip:port，成功即视为该 IP 存活。
func tcpProbe(ctx context.Context, ip netip.Addr, port uint16) bool {
	d := net.Dialer{Timeout: PreflightTimeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), strconv.Itoa(int(port))))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// rawPort 从 clash 节点配置中提取端口（int / float64 / string 均可）。
func rawPort(raw map[string]any) uint16 {
	if raw == nil {
		return 0
	}
	switch v := raw["port"].(type) {
	case int:
		return uint16(v)
	case int64:
		return uint16(v)
	case float64:
		return uint16(v)
	case string:
		if p, err := strconv.Atoi(v); err == nil {
			return uint16(p)
		}
	}
	return 0
}

// FormatPreflightLog 输出一行预检摘要日志。
func FormatPreflightLog(r PreflightResult) string {
	if r.PinnedHosts == 0 && len(r.FailedHosts) == 0 {
		return ""
	}
	s := fmt.Sprintf("预检：%d 个服务器域名已锁定可达 IP", r.PinnedHosts)
	if len(r.FailedHosts) > 0 {
		s += fmt.Sprintf("，%d 个全部采样 IP 不可达（%v）", len(r.FailedHosts), r.FailedHosts)
	}
	return s
}
