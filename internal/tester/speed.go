package tester

import (
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"nekospeed/internal/model"
)

// defaultSpeedURLs 未指定测速地址时的默认大文件列表（依次 fallback）。
var defaultSpeedURLs = []string{
	"https://dl.google.com/dl/android/studio/install/3.4.1.0/android-studio-ide-183.5522156-windows.exe",
	"https://speed.cloudflare.com/__down?bytes=104857600",
}

// Speed 多线程 HTTP GET 下载测速：threads 个 goroutine 循环下载同一 URL，
// 共享原子字节计数器；主 goroutine 每秒采样一次写入 PerSecond（共 durationSec 个）。
// 单线程连接失败自动换 URL/重连，不拖垮整个测试。
func Speed(ctx context.Context, d model.Dialer, url string, threads, durationSec int) *model.SpeedResult {
	urls := defaultSpeedURLs
	if url != "" {
		urls = []string{url}
	}
	if threads <= 0 {
		threads = 1
	}
	if durationSec <= 0 {
		durationSec = 1
	}

	// 兜底超时：测速时长 + 15s
	ctx, cancel := context.WithTimeout(ctx, time.Duration(durationSec)*time.Second+15*time.Second)
	defer cancel()
	// dlCtx 在采样结束时取消，让工作线程尽快收工
	dlCtx, stopDownload := context.WithCancel(ctx)
	defer stopDownload()

	var total atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			downloadLoop(dlCtx, d, urls, workerID, &total)
		}(i)
	}

	res := &model.SpeedResult{
		PerSecond: make([]float64, 0, durationSec),
		Threads:   threads,
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var last int64
loop:
	for i := 0; i < durationSec; i++ {
		select {
		case <-ctx.Done():
			break loop // 外部取消/兜底超时，提前结束
		case <-ticker.C:
			cur := total.Load()
			delta := float64(cur - last)
			last = cur
			res.PerSecond = append(res.PerSecond, delta)
			if delta > res.Max {
				res.Max = delta
			}
		}
	}
	stopDownload()
	wg.Wait()

	if n := len(res.PerSecond); n > 0 {
		sum := 0.0
		for _, v := range res.PerSecond {
			sum += v
		}
		res.Avg = sum / float64(n)
	}
	return res
}

// downloadLoop 单个工作线程：循环下载直到 ctx 取消；失败换下一个 URL 重连。
func downloadLoop(ctx context.Context, d model.Dialer, urls []string, workerID int, total *atomic.Int64) {
	transport := &http.Transport{
		DialContext:           d.DialContext,
		Proxy:                 nil,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			return
		}
		target := urls[(workerID+attempt)%len(urls)]
		downloadOnce(ctx, client, target, total)
		if ctx.Err() != nil {
			return
		}
		// 失败后稍作退避再重连，避免打爆代理
		select {
		case <-ctx.Done():
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// downloadOnce 下载一次 target，把读到的字节计入 total。
func downloadOnce(ctx context.Context, client *http.Client, target string, total *atomic.Int64) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "nekospeed")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return
	}
	_, _ = io.Copy(countingWriter{total: total}, resp.Body)
}

// countingWriter 把写入字节数累加到共享原子计数器，数据本体丢弃。
type countingWriter struct {
	total *atomic.Int64
}

func (w countingWriter) Write(p []byte) (int, error) {
	w.total.Add(int64(len(p)))
	return len(p), nil
}
