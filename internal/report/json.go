// Package report 输出结构化 JSON 测试结果。
package report

import (
	"encoding/json"
	"os"
	"time"

	"nekospeed/internal/model"
)

type document struct {
	Version   string          `json:"version"`
	StartTime time.Time       `json:"startTime"`
	Elapsed   string          `json:"elapsed"`
	Total     int             `json:"total"`
	Succeeded int             `json:"succeeded"`
	Nodes     []nodeDoc       `json:"nodes"`
	Topology  *model.Topology `json:"topology,omitempty"`
}

type nodeDoc struct {
	Index    int                  `json:"index"`
	Name     string               `json:"name"`
	Type     string               `json:"type"`
	Server   string               `json:"server"`
	Error    string               `json:"error,omitempty"`
	Latency  *model.LatencyResult `json:"latency,omitempty"`
	Speed    *model.SpeedResult   `json:"speed,omitempty"`
	NAT      string               `json:"nat,omitempty"`
	Media    []model.UnlockResult `json:"media,omitempty"`
	AI       []model.UnlockResult `json:"ai,omitempty"`
	Inbound  *model.GeoInfo       `json:"inbound,omitempty"`
	Outbound *model.GeoInfo       `json:"outbound,omitempty"`
}

// SaveJSON 将结果写入 path（缩进 JSON）。
func SaveJSON(results []*model.NodeResult, topo *model.Topology, version string, start time.Time, elapsed time.Duration, path string) error {
	doc := document{
		Version:   version,
		StartTime: start,
		Elapsed:   elapsed.String(),
		Total:     len(results),
		Topology:  topo,
	}
	for _, r := range results {
		if r.OK() {
			doc.Succeeded++
		}
		doc.Nodes = append(doc.Nodes, nodeDoc{
			Index:    r.Node.Index,
			Name:     r.Node.Name,
			Type:     r.Node.Type,
			Server:   r.Node.Server,
			Error:    r.Err,
			Latency:  r.Latency,
			Speed:    r.Speed,
			NAT:      r.NAT,
			Media:    r.Media,
			AI:       r.AI,
			Inbound:  r.Inbound,
			Outbound: r.Outbound,
		})
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
