// Package nodepool 将订阅节点（clash 单节点配置）包装成统一拨号器。
// 底层使用 mihomo 内核的 adapter.ParseProxy 解析出 constant.Proxy，
// 再适配为 model.Dialer（network, addr 风格）。
package nodepool

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"

	"github.com/metacubex/mihomo/adapter"
	C "github.com/metacubex/mihomo/constant"

	"nekospeed/internal/model"
)

// UDPProxyDialer 在 model.Dialer 之上暴露 UDP 能力（tester.NAT 等需要）。
// NewDialer 返回的 dialer 必然实现本接口，可安全做类型断言；
// 节点本身是否支持 UDP 由 SupportsUDP() 报告。
type UDPProxyDialer interface {
	model.Dialer
	// ListenPacket 通过代理建立到 addr（"host:port"）的 UDP 会话。
	// 返回的 PacketConn 用 WriteTo(目标地址) 发送，可接收任意来源的报文。
	ListenPacket(ctx context.Context, addr string) (net.PacketConn, error)
	// SupportsUDP 报告节点协议是否支持 UDP 转发。
	SupportsUDP() bool
}

// proxyDialer 把 constant.Proxy 的 Metadata 风格拨号适配为 model.Dialer。
type proxyDialer struct {
	proxy C.Proxy
}

func (d *proxyDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	md, err := addrToMetadata(addr, C.TCP)
	if err != nil {
		return nil, err
	}
	return d.proxy.DialContext(ctx, md)
}

func (d *proxyDialer) ListenPacket(ctx context.Context, addr string) (net.PacketConn, error) {
	md, err := addrToMetadata(addr, C.UDP)
	if err != nil {
		return nil, err
	}
	if !d.proxy.SupportUDP() {
		return nil, fmt.Errorf("节点 %q 不支持 UDP", d.proxy.Name())
	}
	return d.proxy.ListenPacketContext(ctx, md)
}

func (d *proxyDialer) SupportsUDP() bool {
	return d.proxy.SupportUDP()
}

// addrToMetadata 把 "host:port" 拆成 mihomo 的 constant.Metadata。
// host 为 IP 时填 DstIP，否则填 Host（由代理端/系统解析）。
func addrToMetadata(addr string, network C.NetWork) (*C.Metadata, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("解析目标地址 %q 失败: %w", addr, err)
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("解析目标地址 %q 端口失败: %w", addr, err)
	}
	md := &C.Metadata{
		NetWork: network,
		Host:    host,
		DstPort: uint16(port),
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		md.Host = ""
		md.DstIP = ip.Unmap()
	}
	return md, nil
}

// NewDialer 用 mihomo adapter.ParseProxy 解析节点原始配置并包装成 model.Dialer。
// 返回值同时实现 UDPProxyDialer。
func NewDialer(node model.Node) (model.Dialer, error) {
	proxy, err := adapter.ParseProxy(node.Raw)
	if err != nil {
		return nil, fmt.Errorf("解析节点 %q 失败: %w", node.Name, err)
	}
	return &proxyDialer{proxy: proxy}, nil
}
