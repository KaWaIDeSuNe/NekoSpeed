package render

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// fontCandidates 常见平台的中文字体探测路径。
var fontCandidates = []string{
	"/System/Library/Fonts/PingFang.ttc",
	"/System/Library/Fonts/STHeiti Light.ttc",
	"/System/Library/Fonts/Hiragino Sans GB.ttc",
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/opentype/noto/NotoSansCJKsc-Regular.otf",
	"/system/fonts/NotoSansCJK-Regular.ttc",
	"/system/fonts/NotoSansSC-Regular.otf",
	"/system/fonts/NotoSansCJKsc-Regular.otf",
	"/system/fonts/SourceHanSansSC-Regular.otf",
	"/system/fonts/DroidSansFallbackFull.ttf",
	"/system/fonts/DroidSansFallback.ttf",
	"/system/fonts/MiSans-Regular.ttf",
	"/system/fonts/MiSans-Regular.ttc",
	"/system/fonts/MiSansVF.ttf",
	"/system/fonts/HarmonyOS_Sans_SC_Regular.ttf",
	"/system/fonts/HarmonyOS_Sans_SC.ttf",
	"/system/fonts/OnePlusSans-Regular.ttf",
	"/system/fonts/oplus-Regular.ttf",
	"/system/fonts/OPPOSans-Regular.ttf",
	"/system/fonts/Sans-Regular.ttf",
	"/system/fonts/vivo-Sans-Regular.ttf",
	`C:\Windows\Fonts\msyh.ttc`,
	`C:\Windows\Fonts\simhei.ttf`,
}

// fontBook 缓存已加载的字体与按字号派生的 font.Face。
type fontBook struct {
	mu    sync.Mutex
	font  *opentype.Font
	faces map[float64]font.Face
}

var book = &fontBook{faces: map[float64]font.Face{}}

// loadFont 按 meta.FontPath → 系统候选路径的顺序加载字体。
func loadFont(custom string) (*opentype.Font, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	if book.font != nil {
		return book.font, nil
	}
	paths := fontCandidates
	if custom != "" {
		paths = append([]string{custom}, fontCandidates...)
	}
	var errs []string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		f, err := parseFont(b)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		book.font = f
		return f, nil
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("字体加载失败（%s）；请通过 -font 指定一个中文字体文件", strings.Join(errs, "; "))
	}
	return nil, fmt.Errorf("未找到可用中文字体（已尝试 %s 等路径）；请通过 -font 指定字体文件", strings.Join(paths, ", "))
}

// parseFont 解析字体文件内容，优先按 TTC 合集解析，失败再按单字体解析。
func parseFont(b []byte) (*opentype.Font, error) {
	if coll, err := opentype.ParseCollection(b); err == nil && coll.NumFonts() > 0 {
		return coll.Font(0)
	}
	return opentype.Parse(b)
}

// face 返回指定字号的 font.Face（带缓存）。
func face(f *opentype.Font, size float64) (font.Face, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	if fc, ok := book.faces[size]; ok {
		return fc, nil
	}
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	book.faces[size] = fc
	return fc, nil
}

// hasGlyph 判断字体是否包含某 rune 的字形。
func hasGlyph(fc font.Face, r rune) bool {
	_, ok := fc.GlyphAdvance(r)
	return ok
}

// sanitize 去掉字体中没有字形的 rune（如国旗 emoji），避免渲染 panic 或豆腐块。
func sanitize(fc font.Face, s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || hasGlyph(fc, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
