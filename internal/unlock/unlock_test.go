package unlock

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nekospeed/internal/model"
)

// directDialer 直连拨号器（不经过代理），供测试使用。
func directDialer() model.Dialer {
	nd := &net.Dialer{Timeout: 5 * time.Second}
	return model.DialFunc(nd.DialContext)
}

// swapURL 临时替换包内 URL 变量，测试结束后恢复。
func swapURL(t *testing.T, p *string, v string) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

func TestYouTubeRegionUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`<html><script>var ytcfg={"gl":"JP","hl":"en"};</script></html>`))
	}))
	defer srv.Close()
	swapURL(t, &youtubeURL, srv.URL)

	res := checkYouTube(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "JP" {
		t.Fatalf("期望解锁(JP)，得到 %+v", res)
	}
}

func TestYouTubeSentToCN(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"地区为CN", `{"gl":"CN"}`},
		{"无地区码", `<html>no region info</html>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			swapURL(t, &youtubeURL, srv.URL)

			res := checkYouTube(context.Background(), directDialer())
			if res.Status != model.StatusSentToCN || res.Region != "CN" {
				t.Fatalf("期望送中(CN)，得到 %+v", res)
			}
		})
	}
}

func TestNetflixFullUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`<html>{"requestCountry":{"id":"US","countryName":"United States"}}</html>`))
	}))
	defer srv.Close()
	swapURL(t, &netflixNonOriginalID, srv.URL+"/title/81280792")
	swapURL(t, &netflixOriginalID, srv.URL+"/title/80057281")

	res := checkNetflix(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "US" {
		t.Fatalf("期望解锁(US)，得到 %+v", res)
	}
}

func TestNetflixOriginalOnly(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/title/81280792", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><h1>Oh no!</h1>This title currently isn't available to watch in your country.</html>`))
	})
	mux.HandleFunc("/title/80057281", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>Stranger Things</html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	swapURL(t, &netflixNonOriginalID, srv.URL+"/title/81280792")
	swapURL(t, &netflixOriginalID, srv.URL+"/title/80057281")

	res := checkNetflix(context.Background(), directDialer())
	if res.Status != model.StatusPending {
		t.Fatalf("期望待解锁，得到 %+v", res)
	}
}

func TestNetflixBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><h1>Oh no!</h1></html>`))
	}))
	defer srv.Close()
	swapURL(t, &netflixNonOriginalID, srv.URL+"/title/81280792")
	swapURL(t, &netflixOriginalID, srv.URL+"/title/80057281")

	res := checkNetflix(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

func TestBahamutUnlock(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ajax/getdeviceid.php", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "BAHARUNE=abc123")
		w.Write([]byte(`{"deviceid":"dev-xyz","msg":"success"}`))
	})
	mux.HandleFunc("/ajax/token.php", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("device") != "dev-xyz" {
			w.Write([]byte(`{"error":"device required"}`))
			return
		}
		w.Write([]byte(`{"animeSn":37783,"token":"x"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body data-geo="TW"></body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	swapURL(t, &bahamutDeviceURL, srv.URL+"/ajax/getdeviceid.php")
	swapURL(t, &bahamutTokenURL, srv.URL+"/ajax/token.php?adID=89422&sn=37783")
	swapURL(t, &bahamutHomeURL, srv.URL+"/")

	res := checkBahamut(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "TW" {
		t.Fatalf("期望解锁(TW)，得到 %+v", res)
	}
}

func TestBahamutRegionBlocked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ajax/getdeviceid.php", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"deviceid":"dev-xyz"}`))
	})
	mux.HandleFunc("/ajax/token.php", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"error":"region blocked"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	swapURL(t, &bahamutDeviceURL, srv.URL+"/ajax/getdeviceid.php")
	swapURL(t, &bahamutTokenURL, srv.URL+"/ajax/token.php?adID=89422&sn=37783")
	swapURL(t, &bahamutHomeURL, srv.URL+"/")

	res := checkBahamut(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

func TestTikTokUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><script>{"appContext":{"region":"jp"}}</script></html>`))
	}))
	defer srv.Close()
	swapURL(t, &tiktokAPIURL, srv.URL)

	res := checkTikTok(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "JP" {
		t.Fatalf("期望解锁(JP)，得到 %+v", res)
	}
}

func TestTikTokVerifyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><div id="tiktok-verify-page"></div></html>`))
	}))
	defer srv.Close()
	swapURL(t, &tiktokAPIURL, srv.URL)

	res := checkTikTok(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

// openAIMock 同时 mock trace/compliance/ios 三个端点。
func openAIMock(t *testing.T, traceBody, compBody string, compCode int, iosBody string, iosCode int) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(traceBody))
	})
	mux.HandleFunc("/compliance", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(compCode)
		w.Write([]byte(compBody))
	})
	mux.HandleFunc("/ios", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(iosCode)
		w.Write([]byte(iosBody))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	swapURL(t, &openAITraceURL, srv.URL+"/trace")
	swapURL(t, &openAIComplianceURL, srv.URL+"/compliance")
	swapURL(t, &openAIIOSURL, srv.URL+"/ios")
}

func TestOpenAIUnlockDespiteDCBlock(t *testing.T) {
	// 回归：ios 站点对数据中心/非浏览器客户端固定 403（cf_details dc 拦截），
	// 不含地区/VPN 关键词，不得误判为地区封锁。
	openAIMock(t,
		"fl=1f0\nh=chat.openai.com\nloc=US\nts=1.0\n",
		`{"cookie_requirements":{"required":false}}`, 200,
		`{"cf_details":"Request is not allowed. Please try again later.", "type":"dc"}`, 403)

	res := checkOpenAI(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "US" {
		t.Fatalf("DC 403 不应误判，期望解锁(US)，得到 %+v", res)
	}
}

func TestOpenAIUnsupportedCountry(t *testing.T) {
	openAIMock(t,
		"loc=HK\n",
		`{"unsupported_country_name_list":["Hong Kong"]}`, 200,
		"", 403)

	res := checkOpenAI(context.Background(), directDialer())
	if res.Status != model.StatusFailed || res.Region != "HK" {
		t.Fatalf("期望地区封锁(HK)，得到 %+v", res)
	}
}

func TestOpenAIVPNDetected(t *testing.T) {
	openAIMock(t,
		"loc=JP\n",
		`{"cookie_requirements":{"required":false}}`, 200,
		`<html>You seem to be using a VPN or proxy.</html>`, 403)

	res := checkOpenAI(context.Background(), directDialer())
	if res.Status != model.StatusFailed || res.Region != "JP" {
		t.Fatalf("期望代理识别(JP)，得到 %+v", res)
	}
}

func TestOpenAIComplianceErrorMapsToReqFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("loc=US\n"))
	}))
	swapURL(t, &openAITraceURL, srv.URL)
	swapURL(t, &openAIComplianceURL, srv.URL) // trace 请求后关闭，compliance 将连接失败
	swapURL(t, &openAIIOSURL, srv.URL)

	done := make(chan model.UnlockResult, 1)
	go func() { done <- checkOpenAI(context.Background(), directDialer()) }()
	srv.Close()
	res := <-done
	if res.Status != model.StatusReqFailed {
		t.Fatalf("期望请求失败，得到 %+v", res)
	}
}

func TestBilibiliUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("session") == "" {
			t.Error("缺少随机 session 参数")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"message":"success","result":{}}`))
	}))
	defer srv.Close()
	swapURL(t, &bilibiliPlayURL, srv.URL)

	res := checkBilibili(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "港澳台地区" {
		t.Fatalf("期望解锁(港澳台地区)，得到 %+v", res)
	}
}

func TestBilibiliFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":-10403,"message":"抱歉您所在地区不可观看"}`))
	}))
	defer srv.Close()
	swapURL(t, &bilibiliPlayURL, srv.URL)

	res := checkBilibili(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

func TestRequestErrorMapsToReqFailed(t *testing.T) {
	// 指向一个不可达的本地端口。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // 立即关闭，连接将被拒绝
	swapURL(t, &bilibiliPlayURL, srv.URL)

	res := checkBilibili(context.Background(), directDialer())
	if res.Status != model.StatusReqFailed {
		t.Fatalf("期望请求失败，得到 %+v", res)
	}
}

func TestAbemaUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`<html>ABEMA</html>`))
	}))
	defer srv.Close()
	swapURL(t, &abemaCheckURL, srv.URL)
	res := checkAbema(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "JP" {
		t.Fatalf("期望解锁(JP)，得到 %+v", res)
	}
}

func TestAbemaRegionBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`<html>blocked</html>`))
	}))
	defer srv.Close()
	swapURL(t, &abemaCheckURL, srv.URL)
	res := checkAbema(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

func TestAbemaHTTP503(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte(`<html>Service Unavailable</html>`))
	}))
	defer srv.Close()
	swapURL(t, &abemaCheckURL, srv.URL)
	res := checkAbema(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

func TestCheckerPanicRecovered(t *testing.T) {
	c := checkerFunc{"panic_test", func(ctx context.Context, d model.Dialer) model.UnlockResult {
		panic("boom")
	}}
	res := c.Check(context.Background(), directDialer())
	if res.Status != model.StatusReqFailed || res.Platform != "panic_test" {
		t.Fatalf("panic 应被 recover 为 ReqFailed，得到 %+v", res)
	}
}

func TestRunMediaAndAIOrder(t *testing.T) {
	// 不依赖外网：全部请求失败也应按固定顺序返回完整结果。
	media := RunMedia(context.Background(), directDialer())
	if len(media) != len(mediaCheckers) {
		t.Fatalf("媒体结果数量不符：%d != %d", len(media), len(mediaCheckers))
	}
	for i, c := range mediaCheckers {
		if media[i].Platform != c.Platform() {
			t.Fatalf("媒体结果顺序不符：第 %d 项 %s != %s", i, media[i].Platform, c.Platform())
		}
	}

	ai := RunAI(context.Background(), directDialer())
	if len(ai) != len(aiCheckers) {
		t.Fatalf("AI 结果数量不符：%d != %d", len(ai), len(aiCheckers))
	}
	for i, c := range aiCheckers {
		if ai[i].Platform != c.Platform() {
			t.Fatalf("AI 结果顺序不符：第 %d 项 %s != %s", i, ai[i].Platform, c.Platform())
		}
	}
}

// aiCloudflareMock 同时 mock trace（cdn-cgi/trace）和首页。
func aiCloudflareMock(t *testing.T, homeBody string, homeCode int, traceBody string) (traceURL, homeURL string) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(traceBody))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(homeCode)
		w.Write([]byte(homeBody))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + "/trace", srv.URL + "/"
}

func TestClaudeUnlock(t *testing.T) {
	trace, home := aiCloudflareMock(t, `<html>Claude</html>`, 200, "loc=US\n")
	swapURL(t, &claudeTraceURL, trace)
	swapURL(t, &claudeURL, home)
	res := checkClaude(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "US" {
		t.Fatalf("期望解锁(US)，得到 %+v", res)
	}
}

func TestClaudeRegionBlocked(t *testing.T) {
	trace, home := aiCloudflareMock(t, `<html>not available in your country</html>`, 200, "loc=CN\n")
	swapURL(t, &claudeTraceURL, trace)
	swapURL(t, &claudeURL, home)
	res := checkClaude(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

func TestClaudeCF403NotRegionBlock(t *testing.T) {
	// Cloudflare 403（cf_details dc 拦截）不含"not available"，应返回 failed 而非 unlock
	trace, home := aiCloudflareMock(t, `{"cf_details":"Request is not allowed."}`, 403, "loc=US\n")
	swapURL(t, &claudeTraceURL, trace)
	swapURL(t, &claudeURL, home)
	res := checkClaude(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

// apiTraceMock 同时 mock API 端点（任意路径）与 /trace 端点。
func apiTraceMock(t *testing.T, apiPath string, apiCode int, apiBody, traceBody string) (apiURL, traceURL string) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(apiPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(apiCode)
		w.Write([]byte(apiBody))
	})
	mux.HandleFunc("/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(traceBody))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + apiPath, srv.URL + "/trace"
}

func TestLlamaUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"title":"Authentication Error","status":401}`))
	}))
	defer srv.Close()
	swapURL(t, &llamaAPIURL, srv.URL)
	res := checkLlama(context.Background(), directDialer())
	if res.Status != model.StatusUnlock {
		t.Fatalf("期望解锁，得到 %+v", res)
	}
}

func TestLlamaRegionBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":"not supported in your region"}`))
	}))
	defer srv.Close()
	swapURL(t, &llamaAPIURL, srv.URL)
	res := checkLlama(context.Background(), directDialer())
	if res.Status != model.StatusFailed || res.Info != "地区限制" {
		t.Fatalf("期望失败(地区限制)，得到 %+v", res)
	}
}

func TestLlamaOtherError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	swapURL(t, &llamaAPIURL, srv.URL)
	res := checkLlama(context.Background(), directDialer())
	if res.Status != model.StatusFailed || res.Info == "地区限制" {
		t.Fatalf("期望失败(非地区限制)，得到 %+v", res)
	}
}

func TestPerplexityAPIUnlock(t *testing.T) {
	api, trace := apiTraceMock(t, "/chat/completions", 401, `{"error":{"code":401,"type":"invalid_api_key"}}`, "loc=US\n")
	swapURL(t, &perplexityAPIURL, api)
	swapURL(t, &perplexityTraceURL, trace)
	res := checkPerplexity(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "US" {
		t.Fatalf("期望解锁(US)，得到 %+v", res)
	}
}

func TestPerplexityAPIRegionBlocked(t *testing.T) {
	api, trace := apiTraceMock(t, "/chat/completions", 403, `{"error":"not available in your country"}`, "loc=CN\n")
	swapURL(t, &perplexityAPIURL, api)
	swapURL(t, &perplexityTraceURL, trace)
	res := checkPerplexity(context.Background(), directDialer())
	if res.Status != model.StatusFailed || res.Region != "CN" {
		t.Fatalf("期望失败(CN)，得到 %+v", res)
	}
}

func TestPerplexityAPIOtherError(t *testing.T) {
	api, trace := apiTraceMock(t, "/chat/completions", 500, ``, "loc=US\n")
	swapURL(t, &perplexityAPIURL, api)
	swapURL(t, &perplexityTraceURL, trace)
	res := checkPerplexity(context.Background(), directDialer())
	if res.Status != model.StatusFailed || res.Info == "地区限制" {
		t.Fatalf("期望失败(非地区限制)，得到 %+v", res)
	}
}

func TestSoraAPIUnlock(t *testing.T) {
	api, trace := apiTraceMock(t, "/v1/videos", 401, `{"error":{"code":"invalid_api_key"}}`, "loc=US\n")
	swapURL(t, &soraAPIURL, api)
	swapURL(t, &openAITraceURL, trace)
	res := checkSora(context.Background(), directDialer())
	if res.Status != model.StatusUnlock || res.Region != "US" {
		t.Fatalf("期望解锁(US)，得到 %+v", res)
	}
}

func TestSoraAPIRegionBlocked(t *testing.T) {
	api, trace := apiTraceMock(t, "/v1/videos", 403, `{"error":{"code":"unsupported_country_region_territory","message":"Country, region, or territory not supported"}}`, "loc=CN\n")
	swapURL(t, &soraAPIURL, api)
	swapURL(t, &openAITraceURL, trace)
	res := checkSora(context.Background(), directDialer())
	if res.Status != model.StatusFailed || res.Region != "CN" {
		t.Fatalf("期望失败(CN)，得到 %+v", res)
	}
}

func TestGrokUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`<html>Grok</html>`))
	}))
	defer srv.Close()
	swapURL(t, &grokURL, srv.URL)
	res := checkGrok(context.Background(), directDialer())
	if res.Status != model.StatusUnlock {
		t.Fatalf("期望解锁，得到 %+v", res)
	}
}

func TestGrokRegionBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`<html>not available in your country</html>`))
	}))
	defer srv.Close()
	swapURL(t, &grokURL, srv.URL)
	res := checkGrok(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}

func TestGrokHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer srv.Close()
	swapURL(t, &grokURL, srv.URL)
	res := checkGrok(context.Background(), directDialer())
	if res.Status != model.StatusFailed {
		t.Fatalf("期望失败，得到 %+v", res)
	}
}
