package render

import (
	"nekospeed/internal/model"
)

// platformCols 按平台名列表生成列定义。
func platformCols(platforms []string) []Col {
	cols := make([]Col, len(platforms))
	for i, p := range platforms {
		cols[i] = Col{Header: p}
	}
	return cols
}

// SaveFullImage 渲染综合图（*_full.png）：测速列 + 流媒体列 + AI 列。
func SaveFullImage(results []*model.NodeResult, meta Meta, path string) error {
	cols := append(baseCols(),
		Col{Header: "TLS RTT"},
		Col{Header: "HTTPS延迟"},
		Col{Header: "平均速度"},
		Col{Header: "最高速度"},
		Col{Header: "每秒速度", Kind: kindSpark},
	)
	cols = append(cols, platformCols(mediaPlatforms)...)
	cols = append(cols, platformCols(aiPlatforms)...)
	t := &Table{Cols: cols}
	for _, r := range results {
		row := append(baseCells(r, meta.Theme), latencyCells(r, meta.Theme)...)
		row = append(row, speedCells(r, meta.Theme)...)
		for _, p := range mediaPlatforms {
			row = append(row, unlockCell(r.Media, p, meta.Theme))
		}
		for _, p := range aiPlatforms {
			row = append(row, unlockCell(r.AI, p, meta.Theme))
		}
		t.Rows = append(t.Rows, row)
	}
	img, err := renderTables(meta, LocalISPTitle("NekoSpeed | fftest - ", "综合测试", meta.LocalISP, meta.Bandwidth, meta.Single), footerLatencyNote, []*Table{t})
	if err != nil {
		return err
	}
	return savePNG(img, path)
}

// SaveMediaImage 渲染流媒体图（*_media.png）：延迟列 + 媒体各列 + openai。
func SaveMediaImage(results []*model.NodeResult, meta Meta, path string) error {
	cols := append(baseCols(),
		Col{Header: "TLS RTT"},
		Col{Header: "HTTPS延迟"},
	)
	cols = append(cols, platformCols(mediaPlatforms)...)
	cols = append(cols, Col{Header: "openai"})
	t := &Table{Cols: cols}
	for _, r := range results {
		row := append(baseCells(r, meta.Theme), latencyCells(r, meta.Theme)...)
		for _, p := range mediaPlatforms {
			row = append(row, unlockCell(r.Media, p, meta.Theme))
		}
		row = append(row, unlockCell(r.AI, "openai", meta.Theme))
		t.Rows = append(t.Rows, row)
	}
	img, err := renderTables(meta, LocalISPTitle("NekoSpeed | test - ", "基础流媒体", meta.LocalISP, meta.Bandwidth, meta.Single), footerLatencyNote, []*Table{t})
	if err != nil {
		return err
	}
	return savePNG(img, path)
}

// SaveAIImage 渲染 AI 图（*_ai.png）：无延迟列，仅 AI 各列。
func SaveAIImage(results []*model.NodeResult, meta Meta, path string) error {
	cols := append(baseCols(), platformCols(aiPlatforms)...)
	t := &Table{Cols: cols}
	for _, r := range results {
		row := baseCells(r, meta.Theme)
		for _, p := range aiPlatforms {
			row = append(row, unlockCell(r.AI, p, meta.Theme))
		}
		t.Rows = append(t.Rows, row)
	}
	img, err := renderTables(meta, LocalISPTitle("NekoSpeed | ai - ", "AI可用性检测", meta.LocalISP, meta.Bandwidth, meta.Single), footerLatencyNote, []*Table{t})
	if err != nil {
		return err
	}
	return savePNG(img, path)
}
