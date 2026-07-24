package geo

import (
	"sort"

	"nekospeed/internal/model"
)

// BuildTopology 聚合各节点的入口/出口归属为线路分析结果。
// results 必须按订阅原序传入（Node.Index 顺序）。
//
// 入口：相同入口 IP 归为一个 InboundEntry，按首次出现的订阅顺序编号（从 1）。
// 簇：按订阅顺序扫描，入口 IP 与上一个节点相同则同簇，否则簇号 +1
// （参考 MiaoKo 线路分析图：同入口按订阅顺序相邻的归一簇，断开则开新簇）。
// 出口：无出口信息的节点跳过；OutboundEntry 按出口 IP 首次出现顺序稳定排列，
// 同一出口 IP 的节点相邻。
func BuildTopology(results []*model.NodeResult) *model.Topology {
	topo := &model.Topology{}
	indexOf := map[string]int{} // 入口 IP -> 入口序号（从 1）

	cluster := 0
	prevIP := ""
	prevOK := false

	for _, r := range results {
		if r == nil || r.Inbound == nil {
			// 无入口信息：不参与聚合，并打断簇的连续性
			prevOK = false
			continue
		}
		in := r.Inbound
		if !prevOK || in.IP != prevIP {
			cluster++
		}
		prevIP, prevOK = in.IP, true

		idx, ok := indexOf[in.IP]
		if !ok {
			topo.Inbounds = append(topo.Inbounds, model.InboundEntry{
				Index:   len(topo.Inbounds) + 1,
				Geo:     *in,
				Cluster: cluster,
			})
			idx = len(topo.Inbounds)
			indexOf[in.IP] = idx
		} else {
			// 同一入口同时服务 v4/v6 节点时 Stack 合并为 "46"
			e := &topo.Inbounds[idx-1]
			e.Geo.Stack = mergeStack(e.Geo.Stack, in.Stack)
		}

		if r.Outbound == nil {
			continue
		}
		topo.Inbounds[idx-1].OutCount++
		topo.Outbounds = append(topo.Outbounds, model.OutboundEntry{
			InboundIndex: idx,
			Geo:          *r.Outbound,
			Cluster:      cluster,
			NodeName:     r.Node.Name,
		})
	}

	// 出口按出口 IP 首次出现顺序稳定排列
	rank := map[string]int{}
	for _, o := range topo.Outbounds {
		if _, ok := rank[o.Geo.IP]; !ok {
			rank[o.Geo.IP] = len(rank)
		}
	}
	sort.SliceStable(topo.Outbounds, func(i, j int) bool {
		return rank[topo.Outbounds[i].Geo.IP] < rank[topo.Outbounds[j].Geo.IP]
	})

	return topo
}

// mergeStack 合并协议栈标记，"4"+"6" 归一为 "46"。
func mergeStack(a, b string) string {
	if a == b {
		return a
	}
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return "46"
}
