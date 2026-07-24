package render

import (
	"fmt"

	"nekospeed/internal/model"
)

// 平台列定义（表头即小写平台名）。
var (
	mediaPlatforms = []string{"youtube", "netflix", "disney_plus", "bilibili", "dazn", "abema", "bahamut", "steam", "spotify"}
	aiPlatforms    = []string{"openai", "copilot", "gemini", "llama", "claude", "perplexity", "sora", "grok"}
)

const footerLatencyNote = "已核实 TLS 证书。TLS RTT 为单次数据交换延迟， HTTPS Ping 为单次请求体感延迟。"

// baseCols 返回 序号|节点名称|类型 三列。
func baseCols() []Col {
	return []Col{
		{Header: "序号"},
		{Header: "节点名称", Kind: kindName},
		{Header: "类型"},
	}
}

// baseCells 返回一行的 序号|节点名称|类型 单元格。
func baseCells(r *model.NodeResult, theme Theme) []Cell {
	return []Cell{
		{Text: fmt.Sprintf("%d", r.Node.Index), FG: theme.IndexFG, Align: AlignCenter},
		{Text: r.Node.Name, Align: AlignLeft},
		{Text: r.Node.Type, Align: AlignCenter},
	}
}

// latencyCells 返回 TLS RTT 与 HTTPS延迟 单元格（失败红底 -）。
func latencyCells(r *model.NodeResult, theme Theme) []Cell {
	fail := Cell{Text: "-", BG: theme.FailBG, Align: AlignCenter}
	tls, https := fail, fail
	if r.Err == "" && r.Latency != nil {
		if r.Latency.AvgTLSRTT > 0 {
			tls = Cell{Text: fmt.Sprintf("%dms", r.Latency.AvgTLSRTT), BG: theme.latencyColor(r.Latency.AvgTLSRTT), Align: AlignCenter}
		}
		if r.Latency.AvgHTTPS > 0 {
			https = Cell{Text: fmt.Sprintf("%dms", r.Latency.AvgHTTPS), BG: theme.latencyColor(r.Latency.AvgHTTPS), Align: AlignCenter}
		}
	}
	return []Cell{tls, https}
}

// speedCells 返回 平均速度|最高速度|每秒速度 单元格。
func speedCells(r *model.NodeResult, theme Theme) []Cell {
	zero := Cell{Text: "0.00B", BG: theme.ZeroBG, Align: AlignCenter}
	avg, max := zero, zero
	spark := Cell{Spark: []float64{}, Align: AlignCenter}
	if r.Err == "" && r.Speed != nil && r.Speed.Avg > 0 {
		avg = Cell{Text: formatSpeed(r.Speed.Avg), BG: theme.speedColor(r.Speed.Avg), Align: AlignCenter}
		max = Cell{Text: formatSpeed(r.Speed.Max), BG: theme.speedColor(r.Speed.Max), Align: AlignCenter}
		spark = Cell{Spark: r.Speed.PerSecond}
	}
	return []Cell{avg, max, spark}
}

// findUnlock 按平台标识查找解锁结果。
func findUnlock(list []model.UnlockResult, platform string) *model.UnlockResult {
	for i := range list {
		if list[i].Platform == platform {
			return &list[i]
		}
	}
	return nil
}

// unlockCell 解锁状态单元格：找不到结果显示灰底 N/A。
func unlockCell(list []model.UnlockResult, platform string, theme Theme) Cell {
	u := findUnlock(list, platform)
	if u == nil {
		return Cell{Text: "N/A", BG: theme.statusColor(model.StatusNA), Align: AlignCenter}
	}
	return Cell{Text: u.Label(), BG: theme.statusColor(u.Status), Align: AlignCenter}
}

// formatSpeed 1024 进制，保留两位小数，单位 B/KB/MB/GB。
func formatSpeed(v float64) string {
	if v < 1 {
		return "0.00B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.2f%s", v, units[i])
}

// SaveSpeedImage 渲染测速图（*_speed.png）。
func SaveSpeedImage(results []*model.NodeResult, meta Meta, path string) error {
	// NAT 未测（全部为空）则去掉 UDP类型 列
	hasNAT := false
	for _, r := range results {
		if r.NAT != "" {
			hasNAT = true
			break
		}
	}
	cols := append(baseCols(),
		Col{Header: "TLS RTT"},
		Col{Header: "HTTPS延迟"},
		Col{Header: "平均速度"},
		Col{Header: "最高速度"},
		Col{Header: "每秒速度", Kind: kindSpark},
	)
	if hasNAT {
		cols = append(cols, Col{Header: "UDP类型"})
	}
	t := &Table{Cols: cols}
	for _, r := range results {
		row := append(baseCells(r, meta.Theme), latencyCells(r, meta.Theme)...)
		row = append(row, speedCells(r, meta.Theme)...)
		if hasNAT {
			nat := r.NAT
			if nat == "" {
				nat = "-"
			}
			row = append(row, Cell{Text: nat, Align: AlignCenter})
		}
		t.Rows = append(t.Rows, row)
	}
	img, err := renderTables(meta, LocalISPTitle("NekoSpeed | speed - ", "订阅速度", meta.LocalISP, meta.Bandwidth, meta.Single), footerLatencyNote, []*Table{t})
	if err != nil {
		return err
	}
	return savePNG(img, path)
}
