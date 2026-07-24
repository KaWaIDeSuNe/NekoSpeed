package tester

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"nekospeed/internal/model"
)

// defaultPingURL 未指定 pingURL 时的默认延迟测试地址。
const defaultPingURL = "http://www.gstatic.com/generate_204"

// pingTimeout 单次延迟测试超时。
const pingTimeout = 10 * time.Second

// Latency 通过代理向 pingURL 发起 count 次 HTTP 请求，用 httptrace 精确计时。
// HTTPS：强制 TLS1.3，TLS RTT = TLSHandshakeDone→GotFirstResponseByte，
// HTTPSPing = 请求开始→GotFirstResponseByte；
// HTTP：TLSRTT 退化为 TCP 连接时间，HTTPSPing = 写请求→首字节。
// 失败的次数不计入切片与平均，只计入丢包率；全部失败时 Avg 为 0。
func Latency(ctx context.Context, d model.Dialer, pingURL string, count int) *model.LatencyResult {
	if pingURL == "" {
		pingURL = defaultPingURL
	}
	if count <= 0 {
		count = 1
	}

	res := &model.LatencyResult{
		TLSRTT:    make([]int, 0, count),
		HTTPSPing: make([]int, 0, count),
	}
	failed := 0
	for i := 0; i < count; i++ {
		tlsRTT, httpsPing, code, err := pingOnce(ctx, d, pingURL)
		if err != nil {
			failed++
			continue
		}
		res.TLSRTT = append(res.TLSRTT, tlsRTT)
		res.HTTPSPing = append(res.HTTPSPing, httpsPing)
		res.HTTPCode = code
	}
	res.PacketLoss = float64(failed) / float64(count)
	res.AvgTLSRTT = avgInts(res.TLSRTT)
	res.AvgHTTPS = avgInts(res.HTTPSPing)
	return res
}

// pingOnce 发起一次请求并返回 (tlsRTTms, httpsPingMs, httpCode, error)。
func pingOnce(ctx context.Context, d model.Dialer, pingURL string) (int, int, int, error) {
	isHTTPS := strings.HasPrefix(strings.ToLower(pingURL), "https://")

	transport := &http.Transport{
		DialContext:         d.DialContext,
		Proxy:               nil,
		DisableKeepAlives:   true, // 每次测量都重新建连，保证计时完整
		TLSHandshakeTimeout: pingTimeout,
	}
	if isHTTPS {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS13}
	}
	defer transport.CloseIdleConnections()

	var (
		start, wroteReq, tlsDone, firstByte time.Time
		connectDone, gotConn                time.Time
	)
	trace := &httptrace.ClientTrace{
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			tlsDone = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			connectDone = time.Now()
		},
		GotConn: func(_ httptrace.GotConnInfo) {
			gotConn = time.Now()
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			wroteReq = time.Now()
		},
		GotFirstResponseByte: func() {
			firstByte = time.Now()
		},
	}

	reqCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(reqCtx, trace), http.MethodGet, pingURL, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	req.Header.Set("User-Agent", "nekospeed")

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向，首字节即结果
		},
	}

	start = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	// 读完响应体，避免过早关闭连接影响计时语义
	_, _ = io.Copy(io.Discard, resp.Body)

	if firstByte.IsZero() {
		firstByte = time.Now() // 防御：空响应等极端情况
	}

	var tlsRTT, httpsPing time.Duration
	if isHTTPS {
		base := tlsDone
		if base.IsZero() {
			base = start
		}
		tlsRTT = firstByte.Sub(base)
		httpsPing = firstByte.Sub(start)
	} else {
		base := connectDone
		if base.IsZero() {
			base = gotConn
		}
		if base.IsZero() {
			base = start
		}
		tlsRTT = base.Sub(start)
		writeBase := wroteReq
		if writeBase.IsZero() {
			writeBase = base
		}
		httpsPing = firstByte.Sub(writeBase)
	}
	if tlsRTT < 0 {
		tlsRTT = 0
	}
	if httpsPing < 0 {
		httpsPing = 0
	}
	return int(tlsRTT.Milliseconds()), int(httpsPing.Milliseconds()), resp.StatusCode, nil
}

// avgInts 求平均（空切片返回 0）。
func avgInts(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	sum := 0
	for _, x := range xs {
		sum += x
	}
	return sum / len(xs)
}
