// Package webui 提供内嵌 Web UI：浏览器里填订阅 → 启动测速 →
// SSE 实时日志 → 查看结果图。仅使用标准库（net/http + embed）。
package webui

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"nekospeed/internal/assets"
	"nekospeed/internal/runner"
)

//go:embed static/index.html
var indexHTML []byte

// Serve 在 addr 上启动 Web UI HTTP 服务并阻塞，直到服务出错。
// 启动后自动尝试打开浏览器，失败仅打印 URL。
func Serve(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听 %s 失败: %w", addr, err)
	}

	s := &server{hub: newHub()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.Handle("GET /bg/", assets.BGHandler())
	mux.HandleFunc("GET /api/random-bg", assets.RandomBGHandler())
	mux.HandleFunc("POST /api/run", s.handleRun)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("GET /files/{name}", s.handleFiles)

	url := "http://localhost" + ln.Addr().String()
	if strings.HasSuffix(url, "]") { // 监听在 ":port" 上时 Addr 是 "[::]:port"
		url = "http://localhost:" + strings.TrimPrefix(addr, ":")
	}
	fmt.Printf("Web UI 已启动：%s\n", url)
	openBrowser(url)

	return http.Serve(ln, mux)
}

type server struct {
	hub *hub

	mu      sync.Mutex
	running bool
	outDir  string // 最近一次任务的输出目录
}

// runRequest 是 POST /api/run 的请求体（runner.Options 的子集）。
type runRequest struct {
	Source      string `json:"source"`
	Tests       string `json:"tests"`
	Threads     int    `json:"threads"`
	Duration    int    `json:"duration"`
	Concurrency int    `json:"concurrency"`
	PingCount   int    `json:"pingCount"`
	Single      bool   `json:"single"`
	Filter      string `json:"filter"`
	// Proxy 拉取订阅使用的代理（如 http://127.0.0.1:7890），为空时读取 HTTPS_PROXY 等环境变量。
	Proxy string `json:"proxy"`
}

// doneEvent 是 SSE done 事件的数据。
type doneEvent struct {
	OK        bool     `json:"ok"`
	Error     string   `json:"error,omitempty"`
	Total     int      `json:"total"`
	Succeeded int      `json:"succeeded"`
	Elapsed   string   `json:"elapsed"`
	Files     []string `json:"files"` // 文件名（相对 /files/）
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体不是合法 JSON", http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		http.Error(w, "订阅链接/本地路径不能为空", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		http.Error(w, "已有任务在运行中", http.StatusConflict)
		return
	}
	s.running = true
	s.mu.Unlock()

	// 应用默认值（与 CLI flag 默认值一致）
	if req.Tests == "" {
		req.Tests = "all"
	}
	if req.Threads <= 0 {
		req.Threads = 8
	}
	if req.Duration <= 0 {
		req.Duration = 10
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 1
	}
	if req.PingCount <= 0 {
		req.PingCount = 4
	}

	outDir := filepath.Join("out", "web", time.Now().Format("20060102_150405"))
	s.mu.Lock()
	s.outDir = outDir
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"status": "started", "outDir": outDir})

	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()

		s.hub.reset()
		s.hub.logf("输出目录：%s", outDir)
		out, err := runner.Run(context.Background(), runner.Options{
			Source:      req.Source,
			Tests:       req.Tests,
			Threads:     req.Threads,
			Duration:    req.Duration,
			Concurrency: req.Concurrency,
			PingCount:   req.PingCount,
			Single:      req.Single,
			Filter:      req.Filter,
			OutDir:      outDir,
			PingURL:     "http://www.gstatic.com/generate_204",
			Preflight:   true,
			Proxy:       req.Proxy,
		}, s.hub.logf)

		evt := doneEvent{OK: err == nil}
		if err != nil {
			evt.Error = err.Error()
			s.hub.logf("任务失败：%v", err)
		} else {
			evt.Total = out.Total
			evt.Succeeded = out.Succeeded
			evt.Elapsed = out.Elapsed.String()
			for _, f := range out.Files {
				evt.Files = append(evt.Files, filepath.Base(f))
			}
		}
		s.hub.done(evt)
	}()
}

func (s *server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持 SSE", http.StatusInternalServerError)
		return
	}

	ch, replay := s.hub.subscribe()
	defer s.hub.unsubscribe(ch)
	for _, frame := range replay {
		fmt.Fprint(w, frame)
	}
	flusher.Flush()

	for {
		select {
		case frame := <-ch:
			fmt.Fprint(w, frame)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *server) handleFiles(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		http.Error(w, "非法文件名", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	dir := s.outDir
	s.mu.Unlock()
	if dir == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(dir, name))
}

// hub 管理 SSE 订阅者并缓存历史日志帧，后连接的客户端能补齐之前的日志。
type hub struct {
	mu      sync.Mutex
	history []string
	subs    map[chan string]struct{}
}

func newHub() *hub {
	return &hub{subs: make(map[chan string]struct{})}
}

func (h *hub) reset() {
	h.mu.Lock()
	h.history = nil
	h.mu.Unlock()
}

func (h *hub) logf(format string, args ...any) {
	h.broadcast("data: " + fmt.Sprintf(format, args...) + "\n\n")
}

func (h *hub) done(evt doneEvent) {
	data, _ := json.Marshal(evt)
	h.broadcast("event: done\ndata: " + string(data) + "\n\n")
}

func (h *hub) broadcast(frame string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history = append(h.history, frame)
	for ch := range h.subs {
		select {
		case ch <- frame:
		default: // 客户端读取慢时丢弃，避免阻塞任务
		}
	}
}

func (h *hub) subscribe() (chan string, []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 256)
	h.subs[ch] = struct{}{}
	return ch, append([]string(nil), h.history...)
}

func (h *hub) unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

// openBrowser 按平台打开默认浏览器，失败仅提示手动访问。
func openBrowser(url string) {
	go func() {
		time.Sleep(300 * time.Millisecond)
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		if err := cmd.Run(); err != nil {
			fmt.Printf("请手动在浏览器中打开：%s\n", url)
		}
	}()
}
