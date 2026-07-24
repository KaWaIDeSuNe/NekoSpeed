package unlock

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"nekospeed/internal/model"
)

// 各平台检测端点（包内变量，测试时可替换为 httptest 服务器地址）。
var (
	youtubeURL           = "https://www.youtube.com/premium"
	netflixNonOriginalID = "https://www.netflix.com/title/81280792" // 非自制剧（乐高幻影忍者），各区库不同步
	netflixOriginalID    = "https://www.netflix.com/title/80057281" // 自制剧（怪奇物语），有网飞即可看
	disneyPlusURL        = "https://www.disneyplus.com/"
	bilibiliPlayURL      = "https://api.bilibili.com/pgc/player/web/playurl?avid=18281381&cid=29892777&qn=0&type=&otype=json&ep_id=183799&fourk=1&fnver=0&fnval=16&module=bangumi" // 港澳台限定剧集（2026-07 验证：CN/US 返回 -10403）；旧剧集 ep_id=307247 已被纳入大陆片库会误判
	tiktokURL            = "https://www.tiktok.com/"
	tiktokAPIURL         = "https://www.tiktok.com/node/share/video/7049860927877885189"
	daznStartupURL       = "https://startup.core.indazn.com/misl/v5/Startup"
	abemaCheckURL        = "https://abema.tv/"
	abemaAPIURL          = "https://api.abema.io/v1/ip/check?device=android"
	bahamutDeviceURL     = "https://ani.gamer.com.tw/ajax/getdeviceid.php"
	bahamutTokenURL      = "https://ani.gamer.com.tw/ajax/token.php?adID=89422&sn=37783"
	bahamutHomeURL       = "https://ani.gamer.com.tw/"
	steamAppURL          = "https://store.steampowered.com/app/761830"
	spotifySignupURL     = "https://www.spotify.com/signup/"
)

var (
	youtubeGLRe  = regexp.MustCompile(`"gl"\s*:\s*"([A-Z]{2})"`)
	youtubeURLRe = regexp.MustCompile(`[?&]gl=([A-Z]{2})`)
	netflixLocRe = regexp.MustCompile(`"requestCountry"\s*:\s*\{[^}]*"id"\s*:\s*"([A-Z]{2})"`)
	tiktokRegRe  = regexp.MustCompile(`"region"\s*:\s*"([a-zA-Z]{2})"`)
	daznRegionRe = regexp.MustCompile(`"(?i:geoLocatedCountry|country|region)[A-Za-z]*"\s*:\s*"([A-Z]{2})"`)
	steamCurrRe  = regexp.MustCompile(`itemprop="priceCurrency"\s+content="([A-Z]{3})"`)
	bahamutGeoRe = regexp.MustCompile(`data-geo="([^"]+)"`)
)

// randHex16 生成 16 字节随机 hex（bilibili session 参数用）。
func randHex16() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "0123456789abcdef0123456789abcdef"
	}
	return hex.EncodeToString(b)
}

// checkYouTube 通过 premium 页面提取 gl=XX 地区码；CN 或无地区码视为送中。
func checkYouTube(ctx context.Context, d model.Dialer) model.UnlockResult {
	r, err := doGet(ctx, newClient(d, true), youtubeURL)
	if err != nil {
		return reqFailed("youtube")
	}
	if r.StatusCode != 200 {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	region := ""
	if m := youtubeGLRe.FindSubmatch(r.Body); m != nil {
		region = string(m[1])
	} else if m := youtubeURLRe.FindStringSubmatch(r.FinalURL); m != nil {
		region = m[1]
	}
	if region == "" || region == "CN" {
		return model.UnlockResult{Status: model.StatusSentToCN, Region: "CN"}
	}
	return model.UnlockResult{Status: model.StatusUnlock, Region: region}
}

// checkNetflix 分别请求非自制剧与自制剧页面（跟随重定向），页面含 "Oh no!" 即本区不可看：
// 非自制可看 → 全解锁；仅自制可看 → 待解锁；均不可看 → 失败。
// 注意不能靠状态码判断：Netflix 对地区限制页也返回 200，特征在正文。
func checkNetflix(ctx context.Context, d model.Dialer) model.UnlockResult {
	c := newClient(d, true)
	nonOriginal, err := doGet(ctx, c, netflixNonOriginalID)
	if err != nil {
		return reqFailed("netflix")
	}
	original, err := doGet(ctx, c, netflixOriginalID)
	if err != nil {
		return reqFailed("netflix")
	}
	nonAvail := bytes.Contains(nonOriginal.Body, []byte("Oh no!"))
	origAvail := bytes.Contains(original.Body, []byte("Oh no!"))
	if !nonAvail {
		region := ""
		if m := netflixLocRe.FindSubmatch(nonOriginal.Body); m != nil {
			region = string(m[1])
		}
		return model.UnlockResult{Status: model.StatusUnlock, Region: region}
	}
	if !origAvail {
		return model.UnlockResult{Status: model.StatusPending}
	}
	return model.UnlockResult{Status: model.StatusFailed, Info: fmt.Sprintf("非自制/自制均不可看 (%d/%d)", nonOriginal.StatusCode, original.StatusCode)}
}

// checkDisneyPlus 首页跳转 blocked/unavailable 即未解锁，否则视为解锁。
func checkDisneyPlus(ctx context.Context, d model.Dialer) model.UnlockResult {
	r, err := doGet(ctx, newClient(d, false), disneyPlusURL)
	if err != nil {
		return reqFailed("disney_plus")
	}
	loc := strings.ToLower(r.Header.Get("Location"))
	if strings.Contains(loc, "blocked") || strings.Contains(loc, "unavailable") {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	body := strings.ToLower(string(r.Body))
	if strings.Contains(body, "not available in your region") {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	if r.StatusCode >= 400 {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	return model.UnlockResult{Status: model.StatusUnlock}
}

// checkBilibili 请求港澳台限定番剧 playurl（带随机 session 防缓存），code==0 即解锁港澳台地区。
func checkBilibili(ctx context.Context, d model.Dialer) model.UnlockResult {
	u, err := url.Parse(bilibiliPlayURL)
	if err != nil {
		return reqFailed("bilibili")
	}
	q := u.Query()
	q.Set("session", randHex16())
	u.RawQuery = q.Encode()
	r, err := doGet(ctx, newClient(d, true), u.String())
	if err != nil {
		return reqFailed("bilibili")
	}
	if r.StatusCode != 200 {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(r.Body, &resp); err != nil {
		return model.UnlockResult{Status: model.StatusFailed, Info: "响应非 JSON"}
	}
	if resp.Code == 0 {
		return model.UnlockResult{Status: model.StatusUnlock, Region: "港澳台地区"}
	}
	return model.UnlockResult{Status: model.StatusFailed, Info: fmt.Sprintf("code:%d %s", resp.Code, resp.Message)}
}

// checkTikTok 尝试获取 TikTok 视频分享页以提取 region 字段。
// 首页对 Go HTTPS 客户端返回极小 JS 引导页（cookiejar 也无效），
// 因此转用公开视频分享页，其 HTML 内嵌 JSON 含 region 信息。
func checkTikTok(ctx context.Context, d model.Dialer) model.UnlockResult {
	r, err := doGet(ctx, newClient(d, true), tiktokAPIURL)
	if err != nil {
		return reqFailed("tiktok")
	}
	if r.StatusCode != 200 {
		return model.UnlockResult{Status: model.StatusFailed, Info: fmt.Sprintf("HTTP %d(%db)", r.StatusCode, len(r.Body))}
	}
	body := string(r.Body)
	if m := tiktokRegRe.FindStringSubmatch(body); m != nil {
		return model.UnlockResult{Status: model.StatusUnlock, Region: strings.ToUpper(m[1])}
	}
	return model.UnlockResult{Status: model.StatusFailed, Info: fmt.Sprintf("视频页无 region(%db)", len(r.Body))}
}

// checkDazn 通过启动接口返回的地区字段判断可访问地区。
func checkDazn(ctx context.Context, d model.Dialer) model.UnlockResult {
	payload := strings.NewReader(`{"LandingPageKey":"generic","Languages":"en","Platform":"web","Version":"2"}`)
	r, err := doRequest(ctx, newClient(d, true), "POST", daznStartupURL, payload,
		map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return reqFailed("dazn")
	}
	if r.StatusCode != 200 {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	body := string(r.Body)
	if strings.Contains(body, `"isAllowed":false`) || strings.Contains(body, `"IsAllowed":false`) {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	if m := daznRegionRe.FindStringSubmatch(body); m != nil {
		return model.UnlockResult{Status: model.StatusUnlock, Region: m[1]}
	}
	return model.UnlockResult{Status: model.StatusUnlock}
}

// checkAbema 通过主站检测；仅日本解锁。
// 主站返回 200 表示可访问，403/503 表示地区限制。
func checkAbema(ctx context.Context, d model.Dialer) model.UnlockResult {
	r, err := doGet(ctx, newClient(d, true), abemaCheckURL)
	if err != nil {
		return reqFailed("abema")
	}
	if r.StatusCode == 200 {
		return model.UnlockResult{Status: model.StatusUnlock, Region: "JP"}
	}
	body := strings.ToLower(string(r.Body))
	if strings.Contains(body, "blocked") || strings.Contains(body, "not available") || r.StatusCode == 403 || r.StatusCode == 503 {
		return model.UnlockResult{Status: model.StatusFailed, Info: fmt.Sprintf("HTTP %d", r.StatusCode)}
	}
	return model.UnlockResult{Status: model.StatusFailed, Info: fmt.Sprintf("HTTP %d", r.StatusCode)}
}

// checkBahamut 动画疯需先取 deviceid（带 cookie）再请求 token 接口，返回 animeSn 即解锁。
// 地区取自首页 data-geo 属性（TW/HK/MO）。
func checkBahamut(ctx context.Context, d model.Dialer) model.UnlockResult {
	c := newClient(d, true)
	c.Jar, _ = cookiejar.New(nil)

	dev, err := doGet(ctx, c, bahamutDeviceURL)
	if err != nil {
		return reqFailed("bahamut")
	}
	var devResp struct {
		DeviceID string `json:"deviceid"`
	}
	if err := json.Unmarshal(dev.Body, &devResp); err != nil || devResp.DeviceID == "" {
		return model.UnlockResult{Status: model.StatusFailed, Info: "无法获取 deviceid"}
	}

	token, err := doGet(ctx, c, bahamutTokenURL+"&device="+devResp.DeviceID)
	if err != nil {
		return reqFailed("bahamut")
	}
	if !bytes.Contains(token.Body, []byte("animeSn")) {
		return model.UnlockResult{Status: model.StatusFailed, Info: fmt.Sprintf("token 无 animeSn (%d)", token.StatusCode)}
	}

	region := "TW"
	if home, err := doGet(ctx, c, bahamutHomeURL); err == nil {
		if m := bahamutGeoRe.FindSubmatch(home.Body); m != nil {
			region = string(m[1])
		}
	}
	return model.UnlockResult{Status: model.StatusUnlock, Region: region}
}

// checkSteam 商店页面 HTML 的 priceCurrency 即账户钱包货币（随 IP 地区）。
func checkSteam(ctx context.Context, d model.Dialer) model.UnlockResult {
	r, err := doGet(ctx, newClient(d, true), steamAppURL)
	if err != nil {
		return reqFailed("steam")
	}
	if r.StatusCode != 200 {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	if m := steamCurrRe.FindSubmatch(r.Body); m != nil {
		return model.UnlockResult{Status: model.StatusUnlock, Info: "货币：" + string(m[1])}
	}
	return model.UnlockResult{Status: model.StatusUnlock}
}

// checkSpotify 注册页含地区限制提示即不可注册。
func checkSpotify(ctx context.Context, d model.Dialer) model.UnlockResult {
	r, err := doGet(ctx, newClient(d, true), spotifySignupURL)
	if err != nil {
		return reqFailed("spotify")
	}
	if r.StatusCode != 200 {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	body := strings.ToLower(string(r.Body))
	if strings.Contains(body, "not available in your country") || strings.Contains(body, "spotify is currently not available") {
		return model.UnlockResult{Status: model.StatusFailed}
	}
	return model.UnlockResult{Status: model.StatusUnlock}
}
