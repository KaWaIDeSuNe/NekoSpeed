package tester

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nekospeed/internal/model"
)

// directDialer 用 net.Dialer 做 DIRECT 拨号，模拟 model.Dialer。
func directDialer() model.Dialer {
	nd := &net.Dialer{Timeout: 5 * time.Second}
	return model.DialFunc(nd.DialContext)
}

// newPingServer 返回模拟 generate_204 的测试服务器。
func newPingServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Millisecond) // 人为制造可测量的延迟
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newSpeedServer 返回流式输出大文件的测试服务器。
func newSpeedServer(t *testing.T) *httptest.Server {
	t.Helper()
	chunk := make([]byte, 64*1024)
	for i := range chunk {
		chunk[i] = byte(i)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", 64*1024*1024)) // 64MB，测速窗口内读不完
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			if _, err := w.Write(chunk); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLatencyDirect(t *testing.T) {
	srv := newPingServer(t)
	res := Latency(context.Background(), directDialer(), srv.URL, 3)

	if res.AvgHTTPS <= 0 {
		t.Fatalf("AvgHTTPS 应为非零正值，实际 %d（loss=%v）", res.AvgHTTPS, res.PacketLoss)
	}
	if len(res.HTTPSPing) != 3 || len(res.TLSRTT) != 3 {
		t.Fatalf("成功 3 次应有 3 条记录，实际 HTTPSPing=%d TLSRTT=%d", len(res.HTTPSPing), len(res.TLSRTT))
	}
	if res.PacketLoss != 0 {
		t.Fatalf("本地直连不应丢包，实际 %v", res.PacketLoss)
	}
	if res.HTTPCode != http.StatusNoContent {
		t.Fatalf("HTTPCode 应为 204，实际 %d", res.HTTPCode)
	}
}

func TestLatencyAllFailed(t *testing.T) {
	// 指向一个不可能成功的地址，验证全失败时 Avg 为 0、丢包率为 1
	dialer := model.DialFunc(func(context.Context, string, string) (net.Conn, error) {
		return nil, fmt.Errorf("dial refused")
	})
	res := Latency(context.Background(), dialer, "http://example.invalid/generate_204", 2)
	if res.AvgHTTPS != 0 || res.AvgTLSRTT != 0 {
		t.Fatalf("全部失败时 Avg 应为 0，实际 HTTPS=%d TLS=%d", res.AvgHTTPS, res.AvgTLSRTT)
	}
	if res.PacketLoss != 1 {
		t.Fatalf("丢包率应为 1，实际 %v", res.PacketLoss)
	}
}

func TestSpeedDirect(t *testing.T) {
	srv := newSpeedServer(t)
	const duration = 3
	res := Speed(context.Background(), directDialer(), srv.URL, 4, duration)

	if len(res.PerSecond) != duration {
		t.Fatalf("PerSecond 长度应为 %d，实际 %d", duration, len(res.PerSecond))
	}
	if res.Avg <= 0 {
		t.Fatalf("Avg 应为正值，实际 %v", res.Avg)
	}
	if res.Max <= 0 || res.Max < res.Avg {
		t.Fatalf("Max 应 >= Avg 且为正，实际 Max=%v Avg=%v", res.Max, res.Avg)
	}
	if res.Threads != 4 {
		t.Fatalf("Threads 应为 4，实际 %d", res.Threads)
	}
}

func TestSpeedBadURLFallback(t *testing.T) {
	// 单一坏 URL：不应 panic / 挂死，结果为 0
	dialer := model.DialFunc(func(context.Context, string, string) (net.Conn, error) {
		return nil, fmt.Errorf("dial refused")
	})
	done := make(chan *model.SpeedResult, 1)
	go func() {
		done <- Speed(context.Background(), dialer, "http://example.invalid/file.bin", 2, 2)
	}()
	select {
	case res := <-done:
		if res.Avg != 0 {
			t.Fatalf("全失败时 Avg 应为 0，实际 %v", res.Avg)
		}
		if len(res.PerSecond) != 2 {
			t.Fatalf("PerSecond 长度应为 2，实际 %d", len(res.PerSecond))
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Speed 在坏 URL 下挂死")
	}
}
