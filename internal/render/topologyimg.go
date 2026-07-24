package render

import (
	"fmt"

	"nekospeed/internal/model"
)

// SaveTopologyImage 渲染线路分析图（*_topology.png）：上表入口、下表出口，
// 出口表的"入口"列做纵向合并单元格。
func SaveTopologyImage(results []*model.NodeResult, topo *model.Topology, meta Meta, path string) error {
	inTable := &Table{
		Section: "入口",
		Cols: []Col{
			{Header: "序号"},
			{Header: "地区"},
			{Header: "自治编号"},
			{Header: "组织"},
			{Header: "栈"},
			{Header: "出口数量"},
		},
	}
	for _, e := range topo.Inbounds {
		inTable.Rows = append(inTable.Rows, []Cell{
			{Text: fmt.Sprintf("%d", e.Index), FG: meta.Theme.IndexFG, Align: AlignCenter},
			{Text: e.Geo.Country, Align: AlignCenter},
			{Text: e.Geo.ASN, Align: AlignCenter},
			{Text: e.Geo.Org, Align: AlignCenter},
			{Text: stackText(e.Geo.Stack), Badge: e.Geo.Stack != ""},
			{Text: fmt.Sprintf("%d", e.OutCount), Align: AlignCenter},
		})
	}

	outTable := &Table{
		Section: "出口",
		Cols: []Col{
			{Header: "序号"},
			{Header: "入口"},
			{Header: "地区"},
			{Header: "自治编号"},
			{Header: "组织"},
			{Header: "栈"},
			{Header: "节点名称", Kind: kindName},
		},
	}
	// 同 InboundIndex 的连续行合并"入口"列
	for i, e := range topo.Outbounds {
		inCell := Cell{Align: AlignCenter}
		if i == 0 || topo.Outbounds[i-1].InboundIndex != e.InboundIndex {
			span := 1
			for j := i + 1; j < len(topo.Outbounds) && topo.Outbounds[j].InboundIndex == e.InboundIndex; j++ {
				span++
			}
			inCell = Cell{Text: fmt.Sprintf("%d", e.InboundIndex), Align: AlignCenter, Span: span}
		} else {
			inCell = Cell{Skip: true}
		}
		outTable.Rows = append(outTable.Rows, []Cell{
			{Text: fmt.Sprintf("%d", i+1), FG: meta.Theme.IndexFG, Align: AlignCenter},
			inCell,
			{Text: e.Geo.Country, Align: AlignCenter},
			{Text: e.Geo.ASN, Align: AlignCenter},
			{Text: e.Geo.Org, Align: AlignCenter},
			{Text: stackText(e.Geo.Stack), Badge: e.Geo.Stack != ""},
			{Text: e.NodeName, Align: AlignLeft},
		})
	}

	failed := 0
	for _, r := range results {
		if r.Err != "" {
			failed++
		}
	}
	footer1 := fmt.Sprintf("概要=%d->%d, 失败=%d", len(results), len(topo.Outbounds), failed)
	img, err := renderTables(meta, LocalISPTitle("NekoSpeed | topo - ", "线路分析", meta.LocalISP, meta.Bandwidth, meta.Single), footer1, []*Table{inTable, outTable})
	if err != nil {
		return err
	}
	return savePNG(img, path)
}

// stackText 规范化栈标识（"4"/"6"/"46"）。
func stackText(s string) string {
	switch s {
	case "4", "6", "46":
		return s
	case "64":
		return "46"
	default:
		return s
	}
}
