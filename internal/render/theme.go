package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"time"

	"nekospeed/internal/model"
)

// Pattern 背景花纹类型。
type Pattern string

const (
	PatternNone     Pattern = "none"
	PatternDiagonal Pattern = "diagonal"
	PatternDots     Pattern = "dots"
	PatternCross    Pattern = "cross"
)

// Theme 定义一套图片配色方案。
type Theme struct {
	Name string

	ImageBG    color.RGBA // 整图背景
	HeaderBG   color.RGBA // 表头背景
	RowLine    color.RGBA // 行分隔线
	Text       color.RGBA // 正文与标题文字
	FooterText color.RGBA // 页脚文字
	IndexFG    color.RGBA // 序号列文字

	FailBG  color.RGBA // 失败/超时单元格
	ZeroBG  color.RGBA // 速度为 0 单元格
	BadgeBG color.RGBA // 栈徽章（4/6/46）
	Spark   color.RGBA // 每秒速度柱状图

	Watermark color.RGBA // 水印文字颜色（带 Alpha）

	Pattern      Pattern    // 背景花纹类型
	PatternColor color.RGBA // 花纹颜色（建议低透明度）

	LatencyStops []gradStop // 延迟底色渐变
	SpeedStops   []gradStop // 速度底色渐变
	UnlockColors map[model.UnlockStatus]color.RGBA
}

// DefaultTheme 返回经典浅色主题。
func DefaultTheme() Theme {
	return Theme{
		Name:       "default",
		ImageBG:    rgb(0xFF, 0xFF, 0xFF),
		HeaderBG:   rgb(0xE8, 0xE8, 0xE8),
		RowLine:    rgb(0xD9, 0xD9, 0xD9),
		Text:       rgb(0x1A, 0x1A, 0x1A),
		FooterText: rgb(0x1A, 0x1A, 0x1A),
		IndexFG:    rgb(0xD6, 0x30, 0x31),
		FailBG:     rgb(0xE0, 0x60, 0x60),
		ZeroBG:     rgb(0x9E, 0x9E, 0x9E),
		BadgeBG:    rgb(0x3B, 0x8B, 0xD4),
		Spark:      rgb(0x2F, 0xB8, 0xD6),
		Watermark:  color.RGBA{0x9A, 0x9A, 0x9A, 0x30},
		Pattern:    PatternNone,
		LatencyStops: []gradStop{
			{0.0, rgb(0x7B, 0xD8, 0x8F)},
			{0.3, rgb(0xC2, 0xCF, 0x5C)},
			{0.6, rgb(0xF0, 0xA0, 0x4B)},
			{1.0, rgb(0xE0, 0x60, 0x60)},
		},
		SpeedStops: []gradStop{
			{0.0, rgb(0x7B, 0xD8, 0x8F)},
			{0.5, rgb(0x46, 0xC7, 0xC7)},
			{1.0, rgb(0x5B, 0x8D, 0xEF)},
		},
		UnlockColors: defaultUnlockColors(),
	}
}

func defaultUnlockColors() map[model.UnlockStatus]color.RGBA {
	return map[model.UnlockStatus]color.RGBA{
		model.StatusUnlock:    rgb(0x7B, 0xD8, 0x8F),
		model.StatusPending:   rgb(0xF5, 0xD7, 0x6E),
		model.StatusSentToCN:  rgb(0xC9, 0xB9, 0x5C),
		model.StatusFailed:    rgb(0xE0, 0x60, 0x60),
		model.StatusReqFailed: rgb(0xB0, 0x30, 0x30),
		model.StatusNA:        rgb(0xA0, 0xA0, 0xA0),
	}
}

// hsl 把 HSL(0~360, 0~100, 0~100) 转为 RGB。
func hsl(h, s, l float64) color.RGBA {
	h = math.Mod(h, 360.0)
	if h < 0 {
		h += 360.0
	}
	s /= 100.0
	l /= 100.0
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2.0)-1))
	m := l - c/2.0
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return rgb(uint8((r+m)*255), uint8((g+m)*255), uint8((b+m)*255))
}

// hsla 同 hsl，并设置 Alpha 通道。
func hsla(h, s, l, a float64) color.RGBA {
	c := hsl(h, s, l)
	c.A = uint8(a * 255)
	return c
}

// fixedLatencyStops 与主题无关的语义延迟渐变。
var fixedLatencyStops = []gradStop{
	{0.0, rgb(0x7B, 0xD8, 0x8F)},
	{0.3, rgb(0xC2, 0xCF, 0x5C)},
	{0.6, rgb(0xF0, 0xA0, 0x4B)},
	{1.0, rgb(0xE0, 0x60, 0x60)},
}

// fixedSpeedStops 与主题无关的悦目速度渐变：绿 -> 青 -> 蓝。
var fixedSpeedStops = []gradStop{
	{0.0, rgb(0x7B, 0xD8, 0x8F)},
	{0.5, rgb(0x46, 0xC7, 0xC7)},
	{1.0, rgb(0x5B, 0x8D, 0xEF)},
}

// makeLightTheme 生成柔和浅色主题，带指定花纹。
func makeLightTheme(name string, hue float64, pattern Pattern) Theme {
	t := Theme{
		Name:       name,
		ImageBG:    hsl(hue, 12, 97),
		HeaderBG:   hsl(hue, 32, 90),
		RowLine:    hsl(hue, 15, 90),
		Text:       hsl(hue, 15, 22),
		FooterText: hsl(hue, 10, 45),
		IndexFG:    hsl(0, 78, 50),
		FailBG:     hsl(0, 68, 72),
		ZeroBG:     hsl(hue, 10, 72),
		BadgeBG:    hsl(hue, 62, 52),
		Spark:      hsl(hue, 72, 52),
		Watermark:  hsla(hue, 10, 55, 0.15),
		Pattern:    pattern,
		PatternColor: hsla(hue, 25, 88, 0.12),
		LatencyStops: fixedLatencyStops,
		SpeedStops:   fixedSpeedStops,
		UnlockColors: defaultUnlockColors(),
	}
	return t
}

// makeDarkTheme 生成深色主题，带指定花纹。
func makeDarkTheme(name string, hue float64, pattern Pattern) Theme {
	t := Theme{
		Name:       name,
		ImageBG:    hsl(hue, 10, 14),
		HeaderBG:   hsl(hue, 20, 22),
		RowLine:    hsl(hue, 12, 26),
		Text:       hsl(hue, 10, 92),
		FooterText: hsl(hue, 8, 70),
		IndexFG:    hsl(0, 78, 68),
		FailBG:     hsl(0, 60, 55),
		ZeroBG:     hsl(hue, 10, 36),
		BadgeBG:    hsl(hue, 65, 58),
		Spark:      hsl(hue, 70, 60),
		Watermark:  hsla(hue, 10, 45, 0.14),
		Pattern:    pattern,
		PatternColor: hsla(hue, 15, 32, 0.15),
		LatencyStops: fixedLatencyStops,
		SpeedStops:   fixedSpeedStops,
		UnlockColors: defaultUnlockColors(),
	}
	return t
}

// curatedTheme 用精确 RGB 组装一套主题。
func curatedTheme(name string, imageBG, headerBG, rowLine, text, footerText, indexFG, failBG, zeroBG, badgeBG, spark, watermark color.RGBA, pattern Pattern, patternColor color.RGBA) Theme {
	t := Theme{
		Name:       name,
		ImageBG:    imageBG,
		HeaderBG:   headerBG,
		RowLine:    rowLine,
		Text:       text,
		FooterText: footerText,
		IndexFG:    indexFG,
		FailBG:     failBG,
		ZeroBG:     zeroBG,
		BadgeBG:    badgeBG,
		Spark:      spark,
		Watermark:  watermark,
		Pattern:    pattern,
		PatternColor: patternColor,
		LatencyStops: fixedLatencyStops,
		SpeedStops:   fixedSpeedStops,
		UnlockColors: defaultUnlockColors(),
	}
	return t
}

// themeRegistry 预定义主题列表：16 浅色色相主题 + 4 深色精选 + default。
var themeRegistry = func() []Theme {
	var list []Theme
	lightNames := []string{
		"rose_light", "peach_light", "amber_light", "yellow_light",
		"lime_light", "green_light", "mint_light", "teal_light",
		"cyan_light", "sky_light", "azure_light", "blue_light",
		"indigo_light", "violet_light", "purple_light", "pink_light",
	}
	lightHues := []float64{350, 25, 40, 55, 85, 135, 155, 165, 185, 200, 210, 220, 245, 270, 290, 335}
	lightPatterns := []Pattern{
		PatternDiagonal, PatternDots, PatternDiagonal, PatternDots,
		PatternDiagonal, PatternCross, PatternDots, PatternDiagonal,
		PatternCross, PatternDots, PatternDiagonal, PatternCross,
		PatternDots, PatternDiagonal, PatternCross, PatternDots,
	}
	for i, hue := range lightHues {
		list = append(list, makeLightTheme(lightNames[i], hue, lightPatterns[i]))
	}

	// 深色精选（数量少，作为可选）
	list = append(list, makeDarkTheme("nord_dark", 210, PatternDots))
	list = append(list, makeDarkTheme("tokyo_dark", 220, PatternDiagonal))
	list = append(list, makeDarkTheme("forest_dark", 120, PatternCross))
	list = append(list, makeDarkTheme("sunset_dark", 25, PatternDiagonal))

	list = append(list, DefaultTheme())
	return list
}()

// RandomTheme 从主题库随机返回一个主题。
func RandomTheme() Theme {
	return themeRegistry[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(themeRegistry))]
}

// ThemeByName 按名称查找主题，找不到返回 DefaultTheme。
func ThemeByName(name string) Theme {
	for _, t := range themeRegistry {
		if t.Name == name {
			return t
		}
	}
	return DefaultTheme()
}

// ThemeNames 返回所有主题名称，供 CLI/Web UI 展示。
func ThemeNames() []string {
	names := make([]string, len(themeRegistry))
	for i, t := range themeRegistry {
		names[i] = t.Name
	}
	return names
}

// latencyColor 返回延迟值对应的单元格底色。
func (t Theme) latencyColor(ms int) color.RGBA {
	g := float64(ms) / 2000.0
	if g < 0 {
		g = 0
	}
	if g > 1 {
		g = 1
	}
	return gradient(t.LatencyStops, g)
}

// speedColor 返回速度（字节/秒）对应的单元格底色。
func (t Theme) speedColor(v float64) color.RGBA {
	if v <= 0 {
		return t.SpeedStops[0].c
	}
	g := math.Log1p(v) / math.Log1p(1 << 30)
	if g > 1 {
		g = 1
	}
	return gradient(t.SpeedStops, g)
}

// statusColor 返回解锁状态对应的单元格底色。
func (t Theme) statusColor(s model.UnlockStatus) color.RGBA {
	if c, ok := t.UnlockColors[s]; ok {
		return c
	}
	return t.UnlockColors[model.StatusNA]
}

// textOn 按底色亮度选择黑/白文字色。
func (t Theme) textOn(bg color.RGBA) color.RGBA {
	lum := 0.299*float64(bg.R) + 0.587*float64(bg.G) + 0.114*float64(bg.B)
	if lum > 130 {
		return t.Text
	}
	return color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
}

// drawPattern 在图片上绘制背景花纹。花纹会先画在整图上，之后再画表格。
func drawPattern(img *image.RGBA, pattern Pattern, c color.RGBA) {
	if pattern == PatternNone || c.A == 0 {
		return
	}
	tile := createPatternTile(pattern, c)
	if tile == nil {
		return
	}
	b := img.Bounds()
	tw, th := tile.Bounds().Dx(), tile.Bounds().Dy()
	for y := b.Min.Y; y < b.Max.Y; y += th {
		for x := b.Min.X; x < b.Max.X; x += tw {
			r := image.Rect(x, y, x+tw, y+th).Intersect(b)
			draw.Draw(img, r, tile, r.Min.Sub(image.Pt(x, y)), draw.Over)
		}
	}
}

// createPatternTile 创建花纹瓦片。
func createPatternTile(pattern Pattern, c color.RGBA) *image.RGBA {
	const size = 32
	tile := image.NewRGBA(image.Rect(0, 0, size, size))
	switch pattern {
	case PatternDiagonal:
		// 对角线，间隔 16px
		for x := -size; x < size*2; x += 16 {
			for i := 0; i < size; i++ {
				xx, yy := x+i, i
				if xx >= 0 && xx < size && yy >= 0 && yy < size {
					tile.SetRGBA(xx, yy, c)
					// 加粗 1px 线
					if xx+1 < size {
						tile.SetRGBA(xx+1, yy, c)
					}
				}
			}
		}
	case PatternDots:
		// 圆点，中心 3x3
		center := size / 2
		r := image.Rect(center-1, center-1, center+2, center+2)
		draw.Draw(tile, r, image.NewUniform(c), image.Point{}, draw.Src)
	case PatternCross:
		// 十字线，间隔 16px
		for x := -size; x < size*2; x += 16 {
			for i := 0; i < size; i++ {
				xx, yy := x+i, i
				if xx >= 0 && xx < size && yy >= 0 && yy < size {
					tile.SetRGBA(xx, yy, c)
				}
				xx2, yy2 := x+i, size-1-i
				if xx2 >= 0 && xx2 < size && yy2 >= 0 && yy2 < size {
					tile.SetRGBA(xx2, yy2, c)
				}
			}
		}
	default:
		return nil
	}
	return tile
}
