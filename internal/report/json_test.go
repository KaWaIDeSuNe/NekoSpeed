package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nekospeed/internal/model"
)

// TestSaveJSONKeys 锁定 result.json 的字段名契约（小写 camelCase），
// Android 客户端的 Kotlin DTO 依赖这些键名解析。
func TestSaveJSONKeys(t *testing.T) {
	results := []*model.NodeResult{
		{
			Node: model.Node{Index: 1, Name: "节点A", Type: "Vless", Server: "example.com"},
			Latency: &model.LatencyResult{
				TLSRTT:     []int{100, 110},
				HTTPSPing:  []int{200, 210},
				AvgTLSRTT:  105,
				AvgHTTPS:   205,
				PacketLoss: 0.5,
				HTTPCode:   204,
			},
			Speed: &model.SpeedResult{
				PerSecond: []float64{1.5, 2.5},
				Avg:       2.0,
				Max:       2.5,
				Threads:   8,
			},
			NAT: "FullCone",
			Media: []model.UnlockResult{
				{Platform: "youtube", Status: model.StatusUnlock, Region: "JP", Info: ""},
			},
			AI: []model.UnlockResult{
				{Platform: "openai", Status: model.StatusFailed, Region: "CN"},
			},
			Inbound:  &model.GeoInfo{IP: "1.2.3.4", Country: "JP", Region: "Tokyo", City: "Tokyo", ASN: "AS1", Org: "TestOrg", ISP: "TestISP", Stack: "4"},
			Outbound: &model.GeoInfo{IP: "5.6.7.8", Country: "US"},
		},
	}
	topo := &model.Topology{
		Inbounds:  []model.InboundEntry{{Index: 1, Geo: model.GeoInfo{IP: "1.2.3.4"}, Cluster: 1, OutCount: 2}},
		Outbounds: []model.OutboundEntry{{InboundIndex: 1, Geo: model.GeoInfo{IP: "5.6.7.8"}, Cluster: 1, NodeName: "节点A"}},
	}

	path := filepath.Join(t.TempDir(), "result.json")
	if err := SaveJSON(results, topo, "test", time.Now(), time.Second, path); err != nil {
		t.Fatalf("SaveJSON 失败: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取输出失败: %v", err)
	}
	s := string(data)

	wantKeys := []string{
		`"version"`, `"startTime"`, `"elapsed"`, `"total"`, `"succeeded"`, `"nodes"`, `"topology"`,
		`"index"`, `"name"`, `"type"`, `"server"`,
		`"latency"`, `"tlsRTT"`, `"httpsPing"`, `"avgTlsRTT"`, `"avgHTTPS"`, `"packetLoss"`, `"httpCode"`,
		`"speed"`, `"perSecond"`, `"avg"`, `"max"`, `"threads"`,
		`"nat"`, `"media"`, `"ai"`, `"platform"`, `"status"`, `"region"`, `"info"`,
		`"inbound"`, `"outbound"`, `"ip"`, `"country"`, `"city"`, `"asn"`, `"org"`, `"isp"`, `"stack"`,
		`"inbounds"`, `"outbounds"`, `"geo"`, `"cluster"`, `"outCount"`, `"inboundIndex"`, `"nodeName"`,
	}
	for _, k := range wantKeys {
		if !strings.Contains(s, k) {
			t.Errorf("输出缺少键 %s", k)
		}
	}

	badKeys := []string{
		`"AvgTLSRTT"`, `"AvgHTTPS"`, `"PacketLoss"`, `"HTTPCode"`, `"TLSRTT"`, `"HTTPSPing"`,
		`"PerSecond"`, `"Avg"`, `"Max"`, `"Threads"`,
		`"Platform"`, `"Status"`, `"Region"`, `"Info"`,
		`"IP"`, `"Country"`, `"City"`, `"ASN"`, `"Org"`, `"ISP"`, `"Stack"`,
		`"Inbounds"`, `"Outbounds"`, `"Geo"`, `"Cluster"`, `"OutCount"`, `"InboundIndex"`, `"NodeName"`,
	}
	for _, k := range badKeys {
		if strings.Contains(s, k) {
			t.Errorf("输出不应包含大写键 %s", k)
		}
	}
}
