// NekoSpeed — 机场测速工具（CLI 一次性命令 + 内嵌 Web UI）。
// 参考 miaospeed 架构：mihomo 内核作拨号层，输出复刻 MiaoKo 风格的 PNG 结果图 + JSON。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"nekospeed/internal/runner"
	"nekospeed/internal/webui"
)

func main() {
	var (
		source      = flag.String("sub", "", "订阅链接或本地文件路径（Clash YAML 或分享链接订阅）")
		config      = flag.String("config", "", "本地 Clash 配置文件路径（等价于 -sub 的本地文件）")
		tests       = flag.String("tests", "all", "测试项，逗号分隔：speed,media,ai,topo,nat 或 all")
		threads     = flag.Int("threads", 8, "测速线程数（1-32）")
		duration    = flag.Int("duration", 10, "测速时长秒（1-30）")
		single      = flag.Bool("single", false, "单线程测速（等价于 -threads 1）")
		filter      = flag.String("filter", "", "节点名过滤正则")
		concurrency = flag.Int("concurrency", 1, "同时测试的节点数（默认 1，避免节点间抢带宽）")
		outDir      = flag.String("o", "out", "输出目录")
		mmdb        = flag.String("mmdb", "", "MaxMind GeoIP2 mmdb 文件路径（不提供则用在线 API）")
		fontPath    = flag.String("font", "", "自定义中文字体路径（默认自动探测系统字体）")
		watermark   = flag.String("watermark", "", "图片水印文字（默认无水印）")
		name        = flag.String("name", "", "订阅名称，用于输出文件名前缀（覆盖自动提取）")
		speedURL    = flag.String("speed-url", "", "自定义测速文件 URL（默认 Google/Cloudflare 大文件）")
		pingURL     = flag.String("ping-url", "http://www.gstatic.com/generate_204", "延迟测试地址")
		pingCount   = flag.Int("ping-count", 4, "每节点延迟测试次数")
		bandwidth   = flag.String("bandwidth", "", "本地带宽标签（如 1Gbps/1000Mbps），未指定时自动取测速峰值")
		preflight   = flag.Bool("preflight", true, "测试前探测服务器 IP 可达性并锁定活 IP（应对 DDNS 轮换 IP 被封锁）")
		theme       = flag.String("theme", "", "渲染主题名称（如 blue_dark），为空时随机选择")
		proxy       = flag.String("proxy", "", "拉取订阅使用的代理（如 http://127.0.0.1:7890 或 socks5://127.0.0.1:1080），默认读取 HTTPS_PROXY 环境变量")
		web         = flag.String("web", "", "启动内嵌 Web UI（如 :18189 或 18189），非空即启用")
	)
	flag.Parse()

	if *web != "" {
		addr := *web
		if !strings.Contains(addr, ":") {
			addr = ":" + addr
		}
		fatalIf(webui.Serve(addr), "Web UI")
		return
	}

	src := *source
	if src == "" {
		src = *config
	}
	if src == "" {
		fmt.Fprintln(os.Stderr, "错误：必须提供 -sub <订阅URL|文件> 或 -config <clash.yaml>")
		flag.Usage()
		os.Exit(2)
	}

	_, err := runner.Run(context.Background(), runner.Options{
		Source:      src,
		Tests:       *tests,
		Threads:     *threads,
		Duration:    *duration,
		Concurrency: *concurrency,
		PingCount:   *pingCount,
		Single:      *single,
		Filter:      *filter,
		OutDir:      *outDir,
		MMDB:        *mmdb,
		FontPath:    *fontPath,
		Watermark:   *watermark,
		Name:        *name,
		SpeedURL:    *speedURL,
		PingURL:     *pingURL,
		Bandwidth:   *bandwidth,
		Preflight:   *preflight,
		Theme:       *theme,
		Proxy:       *proxy,
	}, func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	})
	fatalIf(err, "执行测速")
}

func fatalIf(err error, what string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s失败: %v\n", what, err)
		os.Exit(1)
	}
}
