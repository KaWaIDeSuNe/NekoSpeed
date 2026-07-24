// Package unlock 实现流媒体与 AI 平台解锁检测。
// 每个平台一个独立 checker 函数，通过 model.Dialer 经代理节点发请求，
// 根据各平台公开特征（重定向、JSON 字段、HTTP 状态码）判断解锁状态与地区。
package unlock

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"nekospeed/internal/model"
)

// requestTimeout 单个 HTTP 请求超时。
const requestTimeout = 15 * time.Second

// desktopUA 伪装桌面版 Chrome。
const desktopUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// maxBodyBytes 限制响应体读取大小，避免大页面占内存（只需前部特征）。
const maxBodyBytes = 1 << 20

// Checker 单个平台的解锁检测器。
type Checker interface {
	Platform() string
	Check(ctx context.Context, d model.Dialer) model.UnlockResult
}

// checkerFunc 适配普通函数为 Checker。
type checkerFunc struct {
	platform string
	fn       func(ctx context.Context, d model.Dialer) model.UnlockResult
}

func (c checkerFunc) Platform() string { return c.platform }

func (c checkerFunc) Check(ctx context.Context, d model.Dialer) (res model.UnlockResult) {
	// 单个 checker 不得 panic 拖垮整体。
	defer func() {
		if r := recover(); r != nil {
			res = model.UnlockResult{Platform: c.platform, Status: model.StatusReqFailed}
		}
	}()
	res = c.fn(ctx, d)
	res.Platform = c.platform
	if res.Status == "" {
		res.Status = model.StatusNA
	}
	return res
}

// mediaCheckers 媒体平台固定顺序（与图上列表头一致）。
var mediaCheckers = []Checker{
	checkerFunc{"youtube", checkYouTube},
	checkerFunc{"netflix", checkNetflix},
	checkerFunc{"disney_plus", checkDisneyPlus},
	checkerFunc{"bilibili", checkBilibili},
	checkerFunc{"dazn", checkDazn},
	checkerFunc{"abema", checkAbema},
	checkerFunc{"bahamut", checkBahamut},
	checkerFunc{"steam", checkSteam},
	checkerFunc{"spotify", checkSpotify},
}

// aiCheckers AI 平台固定顺序。
var aiCheckers = []Checker{
	checkerFunc{"openai", checkOpenAI},
	checkerFunc{"copilot", checkCopilot},
	checkerFunc{"gemini", checkGemini},
	checkerFunc{"llama", checkLlama},
	checkerFunc{"claude", checkClaude},
	checkerFunc{"perplexity", checkPerplexity},
	checkerFunc{"sora", checkSora},
	checkerFunc{"grok", checkGrok},
}

// RunMedia 并发执行全部媒体 checker，结果按 mediaCheckers 顺序返回。
func RunMedia(ctx context.Context, d model.Dialer) []model.UnlockResult {
	return runAll(ctx, d, mediaCheckers)
}

// RunAI 并发执行全部 AI checker，结果按 aiCheckers 顺序返回。
func RunAI(ctx context.Context, d model.Dialer) []model.UnlockResult {
	return runAll(ctx, d, aiCheckers)
}

func runAll(ctx context.Context, d model.Dialer, checkers []Checker) []model.UnlockResult {
	results := make([]model.UnlockResult, len(checkers))
	var wg sync.WaitGroup
	for i, c := range checkers {
		wg.Add(1)
		go func(i int, c Checker) {
			defer wg.Done()
			results[i] = c.Check(ctx, d)
		}(i, c)
	}
	wg.Wait()
	return results
}

// httpResult 一次 HTTP 请求的摘要结果。
type httpResult struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	FinalURL   string
}

// httpError 网络层错误，映射为 StatusReqFailed。
type httpError struct{ err error }

func (e *httpError) Error() string { return e.err.Error() }

// newClient 构建经 d 拨号的 http.Client；followRedirect=false 时不跟随重定向。
func newClient(d model.Dialer, followRedirect bool) *http.Client {
	c := model.HTTPClient(d, requestTimeout)
	if !followRedirect {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return c
}

// doGet 发送 GET 请求并读取响应体（截断至 maxBodyBytes）。
// 网络错误返回 *httpError；4xx/5xx 不算错误，由调用方判断状态码。
func doGet(ctx context.Context, c *http.Client, url string) (*httpResult, error) {
	return doRequest(ctx, c, http.MethodGet, url, nil, nil)
}

// doRequest 发送任意方法的请求。
func doRequest(ctx context.Context, c *http.Client, method, url string, body io.Reader, headers map[string]string) (*httpResult, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, &httpError{err: err}
	}
	req.Header.Set("User-Agent", desktopUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json;q=0.8,*/*;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, &httpError{err: err}
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, &httpError{err: err}
	}
	return &httpResult{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       b,
		FinalURL:   resp.Request.URL.String(),
	}, nil
}

// reqFailed 构造网络请求失败结果。
func reqFailed(platform string) model.UnlockResult {
	return model.UnlockResult{Platform: platform, Status: model.StatusReqFailed}
}
