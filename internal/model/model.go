// Package model 定义 NekoSpeed 各模块共享的核心数据结构。
// 所有测试模块（tester/unlock/geo）与渲染/报告模块只依赖本包，避免循环依赖。
package model

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Node 是一个待测试的代理节点。
type Node struct {
	Index  int            // 订阅原序（从 1 开始）
	Name   string         // 节点名称（订阅原始名称，可能含国旗 emoji）
	Type   string         // 协议类型展示名，如 "Hysteria2" "Vless"
	Server string         // 节点服务器地址（域名或 IP），用于入口分析
	Raw    map[string]any // clash 单节点配置（供 mihomo adapter.ParseProxy 解析）
}

// Dialer 抽象代理拨号能力，底层由 mihomo constant.Proxy 实现。
type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// DialFunc 适配函数为 Dialer。
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

func (f DialFunc) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return f(ctx, network, addr)
}

// HTTPClient 返回一个通过 d 拨号的 http.Client（不自动跟随重定向到代理外，
// 并且忽略 HTTP_PROXY/HTTPS_PROXY 环境变量，避免本机代理干扰测速）。
// 超时由调用方控制 context。
func HTTPClient(d Dialer, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           d.DialContext,
			Proxy:                 nil,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          16,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: timeout,
		},
	}
}

// LatencyResult 延迟测试结果（毫秒）。
type LatencyResult struct {
	TLSRTT     []int   `json:"tlsRTT"`     // 每次 TLS RTT
	HTTPSPing  []int   `json:"httpsPing"`  // 每次 HTTPS 完整请求延迟
	AvgTLSRTT  int     `json:"avgTlsRTT"`  // 平均 TLS RTT（0 表示全部失败）
	AvgHTTPS   int     `json:"avgHTTPS"`   // 平均 HTTPS 延迟
	PacketLoss float64 `json:"packetLoss"` // 丢包率 0~1
	HTTPCode   int     `json:"httpCode"`   // 最后一次 HTTP 状态码
}

// SpeedResult 下行测速结果（字节/秒）。
type SpeedResult struct {
	PerSecond []float64 `json:"perSecond"` // 每秒吞吐
	Avg       float64   `json:"avg"`       // 平均速度
	Max       float64   `json:"max"`       // 最高（单秒峰值）速度
	Threads   int       `json:"threads"`
}

// UnlockStatus 解锁检测状态。
type UnlockStatus string

const (
	StatusUnlock    UnlockStatus = "unlock"    // 解锁
	StatusPending   UnlockStatus = "pending"   // 待解锁（如 Netflix 仅自制剧）
	StatusSentToCN  UnlockStatus = "senttocn"  // 送中（YouTube）
	StatusFailed    UnlockStatus = "failed"    // 失败（平台拒绝）
	StatusReqFailed UnlockStatus = "reqfailed" // 请求失败（网络错误）
	StatusNA        UnlockStatus = "na"        // 未测试/不支持
)

// UnlockResult 单个平台的解锁检测结果。
type UnlockResult struct {
	Platform string       `json:"platform"` // 平台标识，如 "youtube" "netflix" "openai"
	Status   UnlockStatus `json:"status"`   // 状态
	Region   string       `json:"region"`   // 解锁地区码，如 "JP"；失败时也可能有地区信息
	Info     string       `json:"info"`     // 附加信息，如 Steam 货币 "JPY"
}

// Label 返回图上展示的文字，如 "解锁(JP)"、"送中(CN)"、"货币：JPY"。
func (u UnlockResult) Label() string {
	switch u.Status {
	case StatusUnlock:
		if u.Info != "" {
			return u.Info
		}
		if u.Region != "" {
			return "解锁(" + u.Region + ")"
		}
		return "解锁"
	case StatusPending:
		return "待解锁"
	case StatusSentToCN:
		if u.Region != "" {
			return "送中(" + u.Region + ")"
		}
		return "送中"
	case StatusFailed:
		if u.Region != "" {
			return "失败(" + u.Region + ")"
		}
		return "失败"
	case StatusReqFailed:
		return "请求失败"
	default:
		return "N/A"
	}
}

// GeoInfo 一个 IP 的归属信息。
type GeoInfo struct {
	IP      string `json:"ip"`
	Country string `json:"country"` // 国家/地区码，如 "HK"
	Region  string `json:"region"`  // 省/地区，如 "Shanghai"
	City    string `json:"city"`    // 城市，如 "Shanghai"
	ASN     string `json:"asn"`     // 自治编号，如 "AS13335"
	Org     string `json:"org"`     // 组织名
	ISP     string `json:"isp"`     // ISP 名称，如 "China Telecom"
	Stack   string `json:"stack"`   // "4" / "6" / "46"
}

// Topology 线路分析聚合结果。
type Topology struct {
	Inbounds  []InboundEntry  `json:"inbounds"`
	Outbounds []OutboundEntry `json:"outbounds"`
}

type InboundEntry struct {
	Index    int     `json:"index"` // 入口序号（从 1 开始）
	Geo      GeoInfo `json:"geo"`
	Cluster  int     `json:"cluster"`  // 簇编号
	OutCount int     `json:"outCount"` // 对应出口数量
}

type OutboundEntry struct {
	InboundIndex int     `json:"inboundIndex"` // 所属入口序号
	Geo          GeoInfo `json:"geo"`
	Cluster      int     `json:"cluster"`
	NodeName     string  `json:"nodeName"`
}

// NodeResult 一个节点的全部测试结果（nil/空表示该项未测试）。
type NodeResult struct {
	Node     Node
	Latency  *LatencyResult
	Speed    *SpeedResult
	NAT      string // FullCone / RestrictedCone / PortRestrictedCone / Symmetric / Unknown
	Media    []UnlockResult
	AI       []UnlockResult
	Inbound  *GeoInfo
	Outbound *GeoInfo
	Err      string // 致命错误（节点无法拨号等），非空则其余结果可能缺失
}

// OK 返回节点是否可用（延迟、测速、拓扑、解锁任一成功）。
func (r *NodeResult) OK() bool {
	return r.Err == "" && ((r.Latency != nil && r.Latency.AvgHTTPS > 0) || (r.Speed != nil && r.Speed.Avg > 0) || (r.Inbound != nil && r.Inbound.IP != "") || (r.Outbound != nil && r.Outbound.IP != "") || len(r.Media) > 0 || len(r.AI) > 0)
}
