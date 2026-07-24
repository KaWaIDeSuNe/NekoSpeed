// Package runner 封装一次完整测速流程的编排逻辑：
// 加载订阅 → 过滤 → 并发测试 → 聚合拓扑 → 渲染 PNG → 写出 JSON。
// CLI（cmd/nekospeed）与 Web UI（internal/webui）共用本包。
package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"nekospeed/internal/geo"
	"nekospeed/internal/model"
	"nekospeed/internal/nodepool"
	"nekospeed/internal/render"
	"nekospeed/internal/report"
	"nekospeed/internal/subscription"
	"nekospeed/internal/tester"
	"nekospeed/internal/unlock"
)

// Version 输出图片与 JSON 中的版本标识。
const Version = "0.1.0"

// Options 一次测速任务的全部参数。
type Options struct {
	Source      string // 订阅链接或本地文件路径
	Tests       string // 测试项，逗号分隔：speed,media,ai,topo,nat 或 all
	Threads     int    // 测速线程数
	Duration    int    // 测速时长秒
	Concurrency int    // 同时测试的节点数
	PingCount   int    // 每节点延迟测试次数
	Single      bool   // 单线程测速（等价于 Threads=1）
	Filter      string // 节点名过滤正则
	OutDir      string // 输出目录
	MMDB        string // MaxMind GeoIP2 mmdb 路径（空则用在线 API）
	FontPath    string // 自定义中文字体路径
	// Name 手动指定订阅名称，用于输出文件名前缀（覆盖自动提取）。
	Name      string
	// Theme 手动指定渲染主题，为空时随机选择。
	Theme     string
	Watermark string // 图片水印文字
	SpeedURL    string // 自定义测速文件 URL（默认 Google / Cloudflare 大文件）
	PingURL     string // 延迟测试地址
	// Preflight 测试前探测服务器域名全部 IP 的可达性并锁定活 IP（默认开）。
	Preflight bool
	// Bandwidth 手动指定的本地带宽标签（如 "1Gbps"），为空时取测速峰值。
	Bandwidth string
	// Proxy 拉取订阅使用的代理（如 http://127.0.0.1:7890），为空时读取 HTTPS_PROXY 等环境变量。
	Proxy string
}

// Output 一次测速任务的结果摘要。
type Output struct {
	Files     []string      // 生成的文件（含路径）
	Total     int           // 节点总数
	Succeeded int           // 成功节点数
	Elapsed   time.Duration // 测试耗时
}

// Run 执行一次完整测速流程，日志通过 logf 输出（logf 可为 nil）。
func Run(ctx context.Context, o Options, logf func(format string, args ...any)) (*Output, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if o.Source == "" {
		return nil, fmt.Errorf("必须提供订阅 URL 或本地文件路径")
	}
	if o.Single {
		o.Threads = 1
	}
	if err := subscription.SetProxy(o.Proxy); err != nil {
		return nil, err
	}

	testSet := parseTests(o.Tests)

	// 1. 加载订阅
	nodes, subName, err := subscription.Load(o.Source)
	if err != nil {
		return nil, fmt.Errorf("加载订阅失败: %w", err)
	}
	if o.Name != "" {
		subName = o.Name
	}
	if o.Filter != "" {
		re, err := regexp.Compile(o.Filter)
		if err != nil {
			return nil, fmt.Errorf("编译过滤正则失败: %w", err)
		}
		var kept []model.Node
		for _, n := range nodes {
			if re.MatchString(n.Name) {
				kept = append(kept, n)
			}
		}
		nodes = kept
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("没有可用节点")
	}
	logf("共 %d 个节点，测试项：%s", len(nodes), strings.Join(testSet, ","))

	// 1.5 服务器 IP 预检：DDNS 轮换 IP 池中可能混有被封锁的死 IP，
	// 探测出当前可达 IP 钉入 mihomo DefaultHosts，避免拨号随机选中死 IP。
	if o.Preflight {
		if s := nodepool.FormatPreflightLog(nodepool.PreflightServers(ctx, nodes)); s != "" {
			logf("%s", s)
		}
	}

	// 2. GeoIP 解析器（topo 需要，speed 图本地运营商标签也需要）
	resolver, err := geo.NewResolver(o.MMDB)
	if err != nil {
		return nil, fmt.Errorf("初始化 GeoIP 失败: %w", err)
	}
	defer resolver.Close()

	// 3. 并发测试
	results := make([]*model.NodeResult, len(nodes))
	sem := make(chan struct{}, o.Concurrency)
	var wg sync.WaitGroup
	start := time.Now()
	for i, node := range nodes {
		results[i] = &model.NodeResult{Node: node}
		wg.Add(1)
		go func(i int, node model.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			testNode(ctx, node, results[i], testSet, testOpts{
				threads:   o.Threads,
				duration:  o.Duration,
				speedURL:  o.SpeedURL,
				pingURL:   o.PingURL,
				pingCount: o.PingCount,
				resolver:  resolver,
			})
			logf("[%d/%d] %s 完成", i+1, len(nodes), node.Name)
		}(i, node)
	}
	wg.Wait()
	elapsed := time.Since(start).Round(time.Second)

	// 4. 线路分析聚合
	var topo *model.Topology
	if hasTest(testSet, "topo") {
		topo = geo.BuildTopology(results)
	}

	// 5. 输出
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	local := resolver.LookupLocalNetwork(ctx)
	stamp := time.Now().Format("20060102_150405")
	prefix := stamp
	if subName != "" {
		prefix = subName + "_" + stamp
	}
	meta := render.Meta{
		Version:   Version,
		Threads:   o.Threads,
		Total:     len(results),
		Succeeded: countOK(results),
		StartTime: start,
		Watermark: o.Watermark,
		FontPath:  o.FontPath,
		Single:    o.Single,
		LocalISP:  formatLocalISP(local),
		Bandwidth: maxBandwidth(results, o.Bandwidth),
		Theme:     render.RandomTheme(),
	}
	if o.Theme != "" {
		meta.Theme = render.ThemeByName(o.Theme)
	}

	out := &Output{
		Total:     len(results),
		Succeeded: countOK(results),
		Elapsed:   elapsed,
	}
	save := func(fn func(string) error, name, what string) error {
		path := filepath.Join(o.OutDir, name)
		if err := fn(path); err != nil {
			return fmt.Errorf("%s失败: %w", what, err)
		}
		out.Files = append(out.Files, path)
		return nil
	}

	if hasTest(testSet, "speed") {
		if err := save(func(p string) error { return render.SaveSpeedImage(results, meta, p) }, prefix+"_speed.png", "渲染测速图"); err != nil {
			return nil, err
		}
	}
	if hasTest(testSet, "media") {
		if err := save(func(p string) error { return render.SaveMediaImage(results, meta, p) }, prefix+"_media.png", "渲染流媒体图"); err != nil {
			return nil, err
		}
	}
	if hasTest(testSet, "ai") {
		if err := save(func(p string) error { return render.SaveAIImage(results, meta, p) }, prefix+"_ai.png", "渲染 AI 图"); err != nil {
			return nil, err
		}
	}
	if hasTest(testSet, "speed") && (hasTest(testSet, "media") || hasTest(testSet, "ai")) {
		if err := save(func(p string) error { return render.SaveFullImage(results, meta, p) }, prefix+"_full.png", "渲染综合图"); err != nil {
			return nil, err
		}
	}
	if topo != nil {
		if err := save(func(p string) error { return render.SaveTopologyImage(results, topo, meta, p) }, prefix+"_topology.png", "渲染线路分析图"); err != nil {
			return nil, err
		}
	}
	if err := save(func(p string) error { return report.SaveJSON(results, topo, meta.Version, start, elapsed, p) }, prefix+"_result.json", "写出 JSON"); err != nil {
		return nil, err
	}

	logf("完成：成功 %d/%d，耗时 %s，输出目录 %s", out.Succeeded, out.Total, elapsed, o.OutDir)
	return out, nil
}

type testOpts struct {
	threads   int
	duration  int
	speedURL  string
	pingURL   string
	pingCount int
	resolver  *geo.Resolver
}

// testNode 对单个节点执行选中的测试项，结果写入 r。
func testNode(ctx context.Context, node model.Node, r *model.NodeResult, testSet []string, o testOpts) {
	dialer, err := nodepool.NewDialer(node)
	if err != nil {
		r.Err = err.Error()
		return
	}

	needLatency := hasTest(testSet, "speed") || hasTest(testSet, "media")
	if needLatency {
		r.Latency = tester.Latency(ctx, dialer, o.pingURL, o.pingCount)
	}
	if hasTest(testSet, "speed") {
		r.Speed = tester.Speed(ctx, dialer, o.speedURL, o.threads, o.duration)
	}
	if hasTest(testSet, "nat") {
		r.NAT = tester.NAT(ctx, dialer)
	}
	if hasTest(testSet, "media") {
		r.Media = unlock.RunMedia(ctx, dialer)
	}
	if hasTest(testSet, "ai") {
		r.AI = unlock.RunAI(ctx, dialer)
	}
	if o.resolver != nil {
		in, out := o.resolver.ResolveNode(ctx, node, dialer)
		r.Inbound, r.Outbound = in, out
	}
}

func parseTests(s string) []string {
	if s == "all" {
		return []string{"speed", "media", "ai", "topo", "nat"}
	}
	var out []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func hasTest(set []string, name string) bool {
	for _, t := range set {
		if t == name {
			return true
		}
	}
	return false
}

func countOK(rs []*model.NodeResult) int {
	n := 0
	for _, r := range rs {
		if r.OK() {
			n++
		}
	}
	return n
}

// formatLocalISP 把 GeoInfo 整理为 "城市运营商" 的展示字符串。
// 优先使用城市 + 地区，ISP 名称会简化为中文简称；信息不足时返回空。
func formatLocalISP(g model.GeoInfo) string {
	loc := g.City
	if loc == "" {
		loc = g.Region
	}
	if loc == "" {
		return ""
	}
	isp := g.ISP
	if isp == "" {
		isp = g.Org
	}
	isp = localizeISP(isp)
	if isp == "" {
		return loc
	}
	return loc + isp
}

// localizeISP 把常见中英运营商名统一简化为中文简称。
func localizeISP(isp string) string {
	if strings.Contains(isp, "电信") {
		return "电信"
	}
	if strings.Contains(isp, "联通") {
		return "联通"
	}
	if strings.Contains(isp, "移动") {
		return "移动"
	}
	if strings.Contains(isp, "广电") {
		return "广电"
	}
	lower := strings.ToLower(isp)
	switch {
	case strings.Contains(lower, "telecom") || strings.Contains(lower, "chinanet"):
		return "电信"
	case strings.Contains(lower, "unicom"):
		return "联通"
	case strings.Contains(lower, "mobile") || strings.Contains(lower, "cmcc"):
		return "移动"
	case strings.Contains(lower, "crtc") || strings.Contains(lower, "broadcasting"):
		return "广电"
	}
	return isp
}

// maxBandwidth 取所有节点测速峰值的最大值，格式化为 "1Gbps" / "500Mbps"。
// 手动指定 override 时优先使用 override；没有有效速度时返回空字符串。
func maxBandwidth(results []*model.NodeResult, override string) string {
	if override != "" {
		return override
	}
	var maxBPS float64
	for _, r := range results {
		if r != nil && r.Speed != nil && r.Speed.Max > maxBPS {
			maxBPS = r.Speed.Max
		}
	}
	if maxBPS <= 0 {
		return ""
	}
	mbps := maxBPS * 8 / 1e6
	if mbps >= 1000 {
		gbps := mbps / 1000
		if gbps == float64(int(gbps)) {
			return fmt.Sprintf("%dGbps", int(gbps))
		}
		return fmt.Sprintf("%.1fGbps", gbps)
	}
	if mbps < 1 {
		return "<1Mbps"
	}
	return fmt.Sprintf("%.0fMbps", mbps)
}

