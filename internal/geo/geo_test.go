package geo

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"nekospeed/internal/model"
)

func gi(ip, country, asn, org string) *model.GeoInfo {
	return &model.GeoInfo{IP: ip, Country: country, ASN: asn, Org: org, Stack: stackOf(ip)}
}

func TestBuildTopology(t *testing.T) {
	node := func(name string) model.Node { return model.Node{Name: name, Server: "srv-" + name} }
	results := []*model.NodeResult{
		{Node: node("A"), Inbound: gi("1.1.1.1", "HK", "AS1", "isp1"), Outbound: gi("2.2.2.2", "JP", "AS2", "isp2")},
		{Node: node("B"), Inbound: gi("1.1.1.1", "HK", "AS1", "isp1"), Outbound: gi("2.2.2.3", "JP", "AS2", "isp2")},
		{Node: node("C"), Inbound: gi("3.3.3.3", "SG", "AS3", "isp3"), Outbound: gi("2.2.2.2", "JP", "AS2", "isp2")},
		{Node: node("D"), Inbound: gi("1.1.1.1", "HK", "AS1", "isp1"), Outbound: gi("9.9.9.9", "US", "AS9", "isp9")},
		{Node: node("E"), Inbound: gi("3.3.3.3", "SG", "AS3", "isp3")}, // 无出口
		{Node: node("F")}, // 无入口，打断簇
		{Node: node("G"), Inbound: gi("3.3.3.3", "SG", "AS3", "isp3"), Outbound: gi("2.2.2.2", "JP", "AS2", "isp2")},
	}

	topo := BuildTopology(results)
	if topo == nil {
		t.Fatal("BuildTopology 返回 nil")
	}

	// 入口：按首次出现顺序编号，Cluster 取首次出现的簇号
	if len(topo.Inbounds) != 2 {
		t.Fatalf("入口数 = %d, 期望 2", len(topo.Inbounds))
	}
	in1, in2 := topo.Inbounds[0], topo.Inbounds[1]
	if in1.Index != 1 || in1.Geo.IP != "1.1.1.1" || in1.Cluster != 1 || in1.OutCount != 3 {
		t.Errorf("入口1 = %+v, 期望 {Index:1 IP:1.1.1.1 Cluster:1 OutCount:3}", in1)
	}
	if in2.Index != 2 || in2.Geo.IP != "3.3.3.3" || in2.Cluster != 2 || in2.OutCount != 2 {
		t.Errorf("入口2 = %+v, 期望 {Index:2 IP:3.3.3.3 Cluster:2 OutCount:2}", in2)
	}

	// 出口：按出口 IP 首次出现顺序（2.2.2.2 < 2.2.2.3 < 9.9.9.9）稳定排列
	type want struct {
		name    string
		inIdx   int
		cluster int
		outIP   string
	}
	wants := []want{
		{"A", 1, 1, "2.2.2.2"},
		{"C", 2, 2, "2.2.2.2"},
		{"G", 2, 5, "2.2.2.2"}, // E(3.3.3.3) 开簇 4，F 打断后 G 再开簇 5
		{"B", 1, 1, "2.2.2.3"},
		{"D", 1, 3, "9.9.9.9"}, // 断开后再用 1.1.1.1，新簇 3
	}
	if len(topo.Outbounds) != len(wants) {
		t.Fatalf("出口数 = %d, 期望 %d", len(topo.Outbounds), len(wants))
	}
	for i, w := range wants {
		o := topo.Outbounds[i]
		if o.NodeName != w.name || o.InboundIndex != w.inIdx || o.Cluster != w.cluster || o.Geo.IP != w.outIP {
			t.Errorf("出口[%d] = %+v, 期望 {Name:%s InboundIndex:%d Cluster:%d IP:%s}",
				i, o, w.name, w.inIdx, w.cluster, w.outIP)
		}
	}
}

func TestBuildTopologyStackMerge(t *testing.T) {
	results := []*model.NodeResult{
		{Node: model.Node{Name: "v4"}, Inbound: &model.GeoInfo{IP: "1.1.1.1", Stack: "4"}},
		{Node: model.Node{Name: "v6"}, Inbound: &model.GeoInfo{IP: "1.1.1.1", Stack: "6"}},
	}
	topo := BuildTopology(results)
	if len(topo.Inbounds) != 1 {
		t.Fatalf("入口数 = %d, 期望 1", len(topo.Inbounds))
	}
	if topo.Inbounds[0].Geo.Stack != "46" {
		t.Errorf("Stack = %q, 期望 %q", topo.Inbounds[0].Geo.Stack, "46")
	}
	// 同栈不合并
	results[1].Inbound.Stack = "4"
	if got := BuildTopology(results).Inbounds[0].Geo.Stack; got != "4" {
		t.Errorf("Stack = %q, 期望 %q", got, "4")
	}
}

func TestBuildTopologyEmpty(t *testing.T) {
	topo := BuildTopology(nil)
	if topo == nil || len(topo.Inbounds) != 0 || len(topo.Outbounds) != 0 {
		t.Errorf("空输入应返回空 Topology, 得到 %+v", topo)
	}
}

// ipAPIServer 返回一个仿 ip-api.com 的 httptest 服务器与请求计数器。
// 路径 /json/{ip} 返回该 IP 的归属；/json/ 返回代理出口自身信息。
func ipAPIServer(t *testing.T, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Query().Get("fields") != ipAPIFields {
			http.Error(w, "bad fields", http.StatusBadRequest)
			return
		}
		ip := r.URL.Path[len("/json/"):]
		if ip == "" {
			ip = "203.0.113.7" // 模拟代理落地 IP
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"success","countryCode":"HK","query":%q,"as":"AS13335 Cloudflare Inc","org":""}`, ip)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func withBaseURL(t *testing.T, base string) {
	t.Helper()
	old := ipAPIBaseURL
	ipAPIBaseURL = base
	t.Cleanup(func() { ipAPIBaseURL = old })
}

func TestLookupOnlineAndCache(t *testing.T) {
	var hits atomic.Int32
	srv := ipAPIServer(t, &hits)
	withBaseURL(t, srv.URL)

	r, err := NewResolver("")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	ctx := context.Background()
	g := r.Lookup(ctx, "1.2.3.4", nil)
	if g.IP != "1.2.3.4" || g.Country != "HK" || g.ASN != "AS13335" || g.Org != "Cloudflare Inc" || g.Stack != "4" {
		t.Errorf("Lookup = %+v", g)
	}
	// 第二次命中缓存，不再发请求
	g2 := r.Lookup(ctx, "1.2.3.4", nil)
	if g2 != g {
		t.Errorf("缓存结果不一致: %+v vs %+v", g2, g)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("请求次数 = %d, 期望 1（缓存去重）", n)
	}
}

func TestLookupConcurrentDedup(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(50 * time.Millisecond) // 放大并发窗口
		fmt.Fprint(w, `{"status":"success","countryCode":"HK","query":"5.6.7.8","as":"AS13335 Cloudflare Inc","org":""}`)
	}))
	t.Cleanup(srv.Close)
	withBaseURL(t, srv.URL)

	r, _ := NewResolver("")
	defer r.Close()

	var wg sync.WaitGroup
	outs := make([]model.GeoInfo, 32)
	for i := range outs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			outs[i] = r.Lookup(context.Background(), "5.6.7.8", nil)
		}(i)
	}
	wg.Wait()
	if n := hits.Load(); n != 1 {
		t.Errorf("并发相同 IP 请求次数 = %d, 期望 1（inflight 去重）", n)
	}
	for i, g := range outs {
		if g.IP != "5.6.7.8" || g.ASN != "AS13335" {
			t.Errorf("outs[%d] = %+v", i, g)
		}
	}
}

func TestLookupStackV6(t *testing.T) {
	var hits atomic.Int32
	srv := ipAPIServer(t, &hits)
	withBaseURL(t, srv.URL)
	r, _ := NewResolver("")
	defer r.Close()
	g := r.Lookup(context.Background(), "2001:db8::1", nil)
	if g.Stack != "6" {
		t.Errorf("Stack = %q, 期望 %q", g.Stack, "6")
	}
}

func TestLookupLocalNetwork(t *testing.T) {
	var hits atomic.Int32
	srv := ipAPIServer(t, &hits)
	withBaseURL(t, srv.URL)

	r, _ := NewResolver("")
	defer r.Close()

	g := r.LookupLocalNetwork(context.Background())
	if g.IP != "203.0.113.7" || g.Country != "HK" {
		t.Errorf("LookupLocalNetwork = %+v", g)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("请求次数 = %d, 期望 1", n)
	}
}

func TestResolveNodeOnline(t *testing.T) {
	var hits atomic.Int32
	srv := ipAPIServer(t, &hits)
	withBaseURL(t, srv.URL)

	r, _ := NewResolver("")
	defer r.Close()

	// 假代理拨号器：记录调用并直连 httptest 服务器
	var dialed atomic.Int32
	dialer := model.DialFunc(func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialed.Add(1)
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	})

	node := model.Node{Name: "n1", Server: "93.184.216.34"} // 字面量 IP，免 DNS
	in, out := r.ResolveNode(context.Background(), node, dialer)

	if in == nil || in.IP != "93.184.216.34" || in.Country != "HK" {
		t.Errorf("入口 = %+v", in)
	}
	if out == nil || out.IP != "203.0.113.7" || out.ASN != "AS13335" {
		t.Errorf("出口 = %+v", out)
	}
	if dialed.Load() == 0 {
		t.Error("出口查询未经过代理 dialer")
	}
}

func TestResolveNodeIPInfoFallback(t *testing.T) {
	// ip-api 始终失败，应回退 ipinfo
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"fail","message":"reserved range"}`)
	}))
	t.Cleanup(failSrv.Close)
	withBaseURL(t, failSrv.URL)

	infoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ip":"198.51.100.9","country":"JP","org":"AS2516 KDDI CORPORATION"}`)
	}))
	t.Cleanup(infoSrv.Close)
	oldInfo := ipinfoURL
	ipinfoURL = infoSrv.URL
	t.Cleanup(func() { ipinfoURL = oldInfo })

	r, _ := NewResolver("")
	defer r.Close()
	dialer := model.DialFunc(func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	})

	// 入口：ip-api 失败 → 降级结果（仅 IP/Stack）
	in, out := r.ResolveNode(context.Background(), model.Node{Name: "n", Server: "10.0.0.1"}, dialer)
	if in == nil || in.IP != "10.0.0.1" || in.Country != "" {
		t.Errorf("入口降级结果 = %+v", in)
	}
	if out == nil || out.IP != "198.51.100.9" || out.ASN != "AS2516" || out.Org != "KDDI CORPORATION" || out.Country != "JP" {
		t.Errorf("ipinfo 回退出口 = %+v", out)
	}
}

func TestResolveNodeNoDialer(t *testing.T) {
	var hits atomic.Int32
	srv := ipAPIServer(t, &hits)
	withBaseURL(t, srv.URL)
	r, _ := NewResolver("")
	defer r.Close()

	in, out := r.ResolveNode(context.Background(), model.Node{Name: "n", Server: "8.8.8.8"}, nil)
	if in == nil || in.IP != "8.8.8.8" {
		t.Errorf("入口 = %+v", in)
	}
	if out != nil {
		t.Errorf("无 dialer 时出口应为 nil, 得到 %+v", out)
	}
}

func TestNewResolverBadMMDB(t *testing.T) {
	if _, err := NewResolver("/nonexistent/GeoLite2-City.mmdb"); err == nil {
		t.Error("打开不存在的 mmdb 应报错")
	}
}

func TestMergeStack(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"4", "4", "4"},
		{"6", "6", "6"},
		{"4", "6", "46"},
		{"6", "4", "46"},
		{"46", "4", "46"},
		{"", "4", "4"},
		{"4", "", "4"},
	}
	for _, c := range cases {
		if got := mergeStack(c.a, c.b); got != c.want {
			t.Errorf("mergeStack(%q,%q) = %q, 期望 %q", c.a, c.b, got, c.want)
		}
	}
}
