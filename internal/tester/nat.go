package tester

import (
	"context"
	"net"
	"time"

	"github.com/pion/stun"

	"nekospeed/internal/model"
	"nekospeed/internal/nodepool"
)

// NAT 类型检测结果（写入 NodeResult.NAT）。
const (
	NATFullCone           = "FullCone"
	NATRestrictedCone     = "RestrictedCone"
	NATPortRestrictedCone = "PortRestrictedCone"
	NATSymmetric          = "Symmetric"
	NATUnknown            = "Unknown"
)

// stunServers 默认 STUN 服务器（依次 fallback）。
var stunServers = []string{
	"stun.l.google.com:19302",
	"stunserver2025.stunprotocol.org:3478",
}

// CHANGE-REQUEST 标志位（RFC 5780 Section 6）。
const (
	changeIP   = 0x04
	changePort = 0x02
)

// natTimeout NAT 检测整体超时。
const natTimeout = 15 * time.Second

// stunAttemptTimeout 单次 STUN 请求等待响应超时。
const stunAttemptTimeout = 3 * time.Second

// NAT 基于 RFC 5780 流程经代理 UDP 检测 NAT 类型。
// UDP 必须走代理：d 需实现 nodepool.UDPProxyDialer，否则返回 Unknown。
func NAT(ctx context.Context, d model.Dialer) string {
	udp, ok := d.(nodepool.UDPProxyDialer)
	if !ok || !udp.SupportsUDP() {
		return NATUnknown
	}

	ctx, cancel := context.WithTimeout(ctx, natTimeout)
	defer cancel()

	// 解析 STUN 服务器地址（直连 DNS 即可，只有 UDP 报文走代理）
	serverAddrs := make([]*net.UDPAddr, 0, len(stunServers))
	for _, srv := range stunServers {
		if ctx.Err() != nil {
			return NATUnknown
		}
		if addr, err := net.ResolveUDPAddr("udp", srv); err == nil {
			serverAddrs = append(serverAddrs, addr)
		}
	}
	if len(serverAddrs) == 0 {
		return NATUnknown
	}

	// test1（无 change-request）：先在首个可用服务器上拿到映射地址
	var (
		pc      net.PacketConn
		primary *net.UDPAddr
		mapped1 *stun.XORMappedAddress
	)
	for _, addr := range serverAddrs {
		conn, err := udp.ListenPacket(ctx, addr.String())
		if err != nil {
			continue
		}
		if m, err := stunTest(ctx, conn, addr, 0); err == nil {
			pc, primary, mapped1 = conn, addr, m
			break
		}
		conn.Close()
	}
	if mapped1 == nil {
		return NATUnknown // test1 无响应：UDP 被阻断
	}
	defer pc.Close()

	// 经代理时无法获知真实本地地址（出口在代理服务器），跳过“映射==本地”比较，按有 NAT 处理。

	// test2（change ip+port）：能收到其他地址:端口发来的响应即 FullCone
	if _, err := stunTest(ctx, pc, primary, changeIP|changePort); err == nil {
		return NATFullCone
	}

	// 向另一台 STUN 服务器（同一 socket）再做一次 mapping，判断 Symmetric
	for _, addr := range serverAddrs {
		if addr.String() == primary.String() {
			continue
		}
		if m, err := stunTest(ctx, pc, addr, 0); err == nil {
			if m.IP.String() != mapped1.IP.String() || m.Port != mapped1.Port {
				return NATSymmetric
			}
			break
		}
	}

	// test3（change port only）：区分 RestrictedCone / PortRestrictedCone
	if _, err := stunTest(ctx, pc, primary, changePort); err == nil {
		return NATRestrictedCone
	}
	return NATPortRestrictedCone
}

// stunTest 发送一次 Binding Request（changeFlags 为 CHANGE-REQUEST 标志位，0 表示不带该属性），
// 最多重试 2 次，返回 XOR-MAPPED-ADDRESS。
func stunTest(ctx context.Context, pc net.PacketConn, server *net.UDPAddr, changeFlags byte) (*stun.XORMappedAddress, error) {
	req := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	if changeFlags != 0 {
		req.Add(stun.AttrChangeRequest, []byte{0, 0, 0, changeFlags})
	}
	req.Encode()

	buf := make([]byte, 1500)
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, err := pc.WriteTo(req.Raw, server); err != nil {
			return nil, err
		}
		mapped, err := readStunResponse(pc, req, buf)
		if err == nil {
			return mapped, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// readStunResponse 在等待窗口内读取匹配事务 ID 的 Binding Success 响应。
func readStunResponse(pc net.PacketConn, req *stun.Message, buf []byte) (*stun.XORMappedAddress, error) {
	deadline := time.Now().Add(stunAttemptTimeout)
	for {
		if err := pc.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			return nil, err
		}
		msg := &stun.Message{}
		if _, err := msg.Write(buf[:n]); err != nil {
			continue // 非 STUN 报文，丢弃
		}
		if msg.Type != stun.BindingSuccess || msg.TransactionID != req.TransactionID {
			continue
		}
		mapped := &stun.XORMappedAddress{}
		if err := mapped.GetFrom(msg); err != nil {
			// 部分服务器只回 MAPPED-ADDRESS
			var plain stun.MappedAddress
			if err2 := plain.GetFrom(msg); err2 != nil {
				return nil, err
			}
			mapped.IP, mapped.Port = plain.IP, plain.Port
		}
		return mapped, nil
	}
}
