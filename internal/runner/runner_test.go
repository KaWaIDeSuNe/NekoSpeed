package runner

import (
	"testing"

	"nekospeed/internal/model"
)

func TestFormatLocalISP(t *testing.T) {
	cases := []struct {
		g    model.GeoInfo
		want string
	}{
		{model.GeoInfo{City: "上海", ISP: "China Telecom"}, "上海电信"},
		{model.GeoInfo{City: "深圳", ISP: "China Unicom"}, "深圳联通"},
		{model.GeoInfo{Region: "陕西", ISP: "China Mobile"}, "陕西移动"},
		{model.GeoInfo{City: "北京", ISP: "中国电信"}, "北京电信"},
		{model.GeoInfo{City: "广州", ISP: "China Broadcasting Network"}, "广州广电"},
		{model.GeoInfo{City: "上海", ISP: ""}, "上海"},
		{model.GeoInfo{}, ""},
	}
	for _, c := range cases {
		got := formatLocalISP(c.g)
		if got != c.want {
			t.Errorf("formatLocalISP(%+v) = %q, want %q", c.g, got, c.want)
		}
	}
}

func TestMaxBandwidth(t *testing.T) {
	cases := []struct {
		results  []*model.NodeResult
		override string
		want     string
	}{
		{nil, "1Gbps", "1Gbps"},
		{nil, "", ""},
		{[]*model.NodeResult{{Speed: &model.SpeedResult{Max: 125_000_000}}}, "", "1Gbps"},
		{[]*model.NodeResult{{Speed: &model.SpeedResult{Max: 62_500_000}}}, "", "500Mbps"},
		{[]*model.NodeResult{{Speed: &model.SpeedResult{Max: 250_000_000}}}, "", "2Gbps"},
		{[]*model.NodeResult{{Speed: &model.SpeedResult{Max: 1}}}, "", "<1Mbps"},
	}
	for _, c := range cases {
		got := maxBandwidth(c.results, c.override)
		if got != c.want {
			t.Errorf("maxBandwidth(%v, %q) = %q, want %q", c.results, c.override, got, c.want)
		}
	}
}
