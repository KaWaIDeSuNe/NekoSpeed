package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"nekospeed/internal/assets"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// 版式常量（像素）。
const (
	fontSize    float64 = 26  // 正文字号
	titleSize   float64 = 34  // 图标题字号
	footerSize  float64 = 22  // 页脚字号
	wmSize      float64 = 40  // 水印字号
	cellPadX            = 14  // 单元格水平内边距
	rowHeight           = 50  // 行高（约 字号*1.9）
	margin              = 26  // 整图边距
	tableGap            = 34  // 多表之间的间隔
	maxNameW            = 460 // 节点名称列最大宽（超出截断加 …）
	sparkColW           = 150 // 每秒速度柱状图列宽
	footerLineH         = 36  // 页脚行高
	titleH              = 61  // 图标题块高（titleSize*1.8）
	sectionH            = 46  // 表小标题块高（fontSize*1.8）
	badgeSize           = 31  // 栈徽章边长（fontSize*1.2）
)

// Align 单元格文字对齐方式。
type Align int

const (
	AlignCenter Align = iota
	AlignLeft
)

// colKind 列类型：普通文字 / 节点名称（左对齐+限宽截断）/ 柱状图（定宽）。
type colKind int

const (
	kindText colKind = iota
	kindName
	kindSpark
)

// Col 表格列定义。
type Col struct {
	Header string
	Kind   colKind
	width  int // 计算出的列宽
}

// Cell 单元格。
type Cell struct {
	Text  string
	BG    color.Color // 整格底色，nil 为白底
	FG    color.Color // 文字色，nil 为自动（随底色反白或黑）
	Align Align
	Spark []float64 // 非空则画柱状图而不是文字
	Badge bool      // Text 为 "4"/"6"/"46"，画蓝色小方块徽章
	Span  int       // 纵向合并：>1 表示向下合并的行数
	Skip  bool      // 被上方合并单元格覆盖
}

// Table 一张表。
type Table struct {
	Section string // 表上方居中的小标题（如 入口/出口），可空
	Cols    []Col
	Rows    [][]Cell
}

// painter 持有整张图共享的绘制状态。
type painter struct {
	img       *image.RGBA
	face      font.Face // 正文
	titleFace font.Face // 图标题
	footFace  font.Face // 页脚
	wmFace    font.Face // 水印
	theme     Theme     // 当前配色主题
}


// renderTables 渲染一张完整图片：图标题 + 若干张表 + 页脚 + 水印。
// footer1 为页脚第一行（各图不同），第二、三行固定。
func renderTables(meta Meta, title, footer1 string, tables []*Table) (*image.RGBA, error) {
	f, err := loadFont(meta.FontPath)
	if err != nil {
		return nil, err
	}
	theme := meta.Theme
	if theme.Name == "" {
		theme = RandomTheme()
	}
	p := &painter{theme: theme}
	if p.face, err = face(f, fontSize); err != nil {
		return nil, err
	}
	if p.titleFace, err = face(f, titleSize); err != nil {
		return nil, err
	}
	if p.footFace, err = face(f, footerSize); err != nil {
		return nil, err
	}
	if p.wmFace, err = face(f, wmSize); err != nil {
		return nil, err
	}

	// 计算各表列宽与总宽
	maxTW := 0
	for _, t := range tables {
		p.measureTable(t)
		if tw := t.totalWidth(); tw > maxTW {
			maxTW = tw
		}
	}
	imgW := maxTW + 2*margin

	// 计算总高
	imgH := margin
	if title != "" {
		imgH += titleH
	}
	for _, t := range tables {
		if t.Section != "" {
			imgH += sectionH
		}
		imgH += rowHeight * (1 + len(t.Rows))
		imgH += tableGap
	}
	imgH += 3*footerLineH + margin

	p.img = image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	// 优先使用内嵌背景图，并叠一层主题底色半透明覆盖以保证可读性。
	if bgImg, _, err := assets.RandomBackgroundImage(); err == nil {
		xdraw.BiLinear.Scale(p.img, p.img.Bounds(), bgImg, bgImg.Bounds(), draw.Src, nil)
		base := color.NRGBAModel.Convert(theme.ImageBG).(color.NRGBA)
		base.A = 205
		draw.Draw(p.img, p.img.Bounds(), image.NewUniform(base), image.Point{}, draw.Over)
	} else {
		draw.Draw(p.img, p.img.Bounds(), image.NewUniform(theme.ImageBG), image.Point{}, draw.Src)
	}
	drawPattern(p.img, theme.Pattern, theme.PatternColor)

	// 图标题
	y := margin
	if title != "" {
		h := titleH
		p.drawText(image.Rect(0, y, imgW, y+h), p.titleFace, title, AlignCenter, theme.Text)
		y += h
	}
	// 各表（水平居中）
	for _, t := range tables {
		x := (imgW - t.totalWidth()) / 2
		y += p.drawTable(t, x, y)
		y += tableGap
	}
	// 页脚三行
	mark := "√"
	if hasGlyph(p.footFace, '✅') {
		mark = "✅"
	}
	foot := []string{
		mark + " " + footer1,
		fmt.Sprintf("NekoSpeed=%s, 线程=%d 概要=%d/%d 排序=订阅原序", meta.Version, meta.Threads, meta.Succeeded, meta.Total),
		fmt.Sprintf("测试时间： %s (CST), 本测试为试验性结果，仅供参考。", meta.StartTime.Local().Format("2006-01-02 15:04:05")),
	}
	for _, line := range foot {
		p.drawText(image.Rect(0, y, imgW, y+footerLineH), p.footFace, line, AlignCenter, theme.FooterText)
		y += footerLineH
	}

	// 水印（最上层，斜向平铺）
	drawWatermark(p.img, p.wmFace, sanitize(p.wmFace, meta.Watermark), theme.Watermark)
	return p.img, nil
}

func (t *Table) totalWidth() int {
	w := 0
	for _, c := range t.Cols {
		w += c.width
	}
	return w
}

// measureString 量取（sanitize 后的）文字宽度。
func (p *painter) measureString(fc font.Face, s string) int {
	return font.MeasureString(fc, sanitize(fc, s)).Ceil()
}

// measureTable 自适应计算每列宽度：表头与所有单元格文字宽度 + padding，
// 名称列限宽，柱状图列定宽。
func (p *painter) measureTable(t *Table) {
	for j := range t.Cols {
		c := &t.Cols[j]
		switch c.Kind {
		case kindSpark:
			c.width = sparkColW
			continue
		}
		w := p.measureString(p.face, c.Header) + 2*cellPadX
		for _, row := range t.Rows {
			if j >= len(row) {
				continue
			}
			cell := row[j]
			if cell.Skip || cell.Spark != nil {
				continue
			}
			cw := p.measureString(p.face, cell.Text) + 2*cellPadX
			if cell.Badge && cell.Text == "46" {
				cw += badgeSize + 6 // 两个并排徽章的额外宽度
			}
			if cw > w {
				w = cw
			}
		}
		if c.Kind == kindName && w > maxNameW {
			w = maxNameW
		}
		c.width = w
	}
}

// drawTable 在 (x,y) 处绘制一张表，返回消耗的高度。
func (p *painter) drawTable(t *Table, x, y int) int {
	y0 := y
	tw := t.totalWidth()
	if t.Section != "" {
		h := sectionH
		p.drawText(image.Rect(x, y, x+tw, y+h), p.face, t.Section, AlignCenter, p.theme.Text)
		y += h
	}
	// 表头：主题色底黑字居中
	hdr := image.Rect(x, y, x+tw, y+rowHeight)
	draw.Draw(p.img, hdr, image.NewUniform(p.theme.HeaderBG), image.Point{}, draw.Src)
	cx := x
	for j := range t.Cols {
		r := image.Rect(cx, y, cx+t.Cols[j].width, y+rowHeight)
		p.drawText(r, p.face, t.Cols[j].Header, AlignCenter, p.theme.Text)
		cx += t.Cols[j].width
	}
	y += rowHeight
	// 表头下分隔线
	p.hline(x, x+tw, y-1)

	// 数据行
	for _, row := range t.Rows {
		cx := x
		for j := range t.Cols {
			if j >= len(row) {
				cx += t.Cols[j].width
				continue
			}
			cell := row[j]
			w := t.Cols[j].width
			cx += w
			if cell.Skip {
				continue
			}
			h := rowHeight
			if cell.Span > 1 {
				h = rowHeight * cell.Span
			}
			r := image.Rect(cx-w, y, cx, y+h)
			p.drawCell(r, cell)
		}
		y += rowHeight
	}
	return y - y0
}

// drawCell 绘制单个单元格：整格底色 + 内容 + 底部分隔线。
func (p *painter) drawCell(r image.Rectangle, c Cell) {
	if c.BG != nil {
		draw.Draw(p.img, r, image.NewUniform(c.BG), image.Point{}, draw.Src)
	}
	var fg color.Color = p.theme.Text
	if c.FG != nil {
		fg = c.FG
	} else if c.BG != nil {
		fg = p.theme.textOn(toRGBA(c.BG))
	}
	switch {
	case c.Spark != nil:
		drawSparkline(p.img, r, c.Spark, p.theme.Spark)
	case c.Badge:
		p.drawBadges(r, c.Text)
	case c.Text != "":
		p.drawText(r, p.face, c.Text, c.Align, fg)
	}
	// 底部细灰线（合并单元格只在自己的底边画）
	p.hline(r.Min.X, r.Max.X, r.Max.Y-1)
}

// drawBadges 绘制栈徽章：蓝底白字 "4"/"6"，"46" 画两个并排。
func (p *painter) drawBadges(r image.Rectangle, s string) {
	size := badgeSize
	gap := 6
	n := len(s)
	total := n*size + (n-1)*gap
	x := r.Min.X + (r.Dx()-total)/2
	y := r.Min.Y + (r.Dy()-size)/2
	for i := 0; i < n; i++ {
		br := image.Rect(x, y, x+size, y+size)
		draw.Draw(p.img, br, image.NewUniform(p.theme.BadgeBG), image.Point{}, draw.Src)
		p.drawText(br, p.face, string(s[i]), AlignCenter, color.White)
		x += size + gap
	}
}

// drawText 在矩形内按对齐方式绘制单行文字（垂直居中），名称列左对齐并限宽截断。
func (p *painter) drawText(r image.Rectangle, fc font.Face, s string, align Align, fg color.Color) {
	s = sanitize(fc, s)
	if s == "" {
		return
	}
	maxW := r.Dx() - 2*cellPadX
	if align == AlignLeft {
		s = truncate(fc, s, maxW)
	}
	w := font.MeasureString(fc, s).Ceil()
	var x int
	switch align {
	case AlignLeft:
		x = r.Min.X + cellPadX
	default:
		x = r.Min.X + (r.Dx()-w)/2
	}
	m := fc.Metrics()
	height := (m.Ascent + m.Descent).Ceil()
	baseY := r.Min.Y + (r.Dy()-height)/2 + m.Ascent.Ceil()
	d := &font.Drawer{
		Dst:  p.img,
		Src:  image.NewUniform(fg),
		Face: fc,
		Dot:  fixed.P(x, baseY),
	}
	d.DrawString(s)
}

// truncate 超出 maxW 时截断并加 …。
func truncate(fc font.Face, s string, maxW int) string {
	if font.MeasureString(fc, s).Ceil() <= maxW {
		return s
	}
	rs := []rune(s)
	for len(rs) > 1 {
		rs = rs[:len(rs)-1]
		if font.MeasureString(fc, string(rs)+"…").Ceil() <= maxW {
			return string(rs) + "…"
		}
	}
	return "…"
}

// hline 画一条 1px 水平分隔线。
func (p *painter) hline(x0, x1, y int) {
	r := image.Rect(x0, y, x1, y+1).Intersect(p.img.Bounds())
	if !r.Empty() {
		draw.Draw(p.img, r, image.NewUniform(p.theme.RowLine), image.Point{}, draw.Src)
	}
}

// drawWatermark 整图叠斜向（约 -30°）半透明重复平铺水印。
func drawWatermark(img *image.RGBA, fc font.Face, text string, c color.RGBA) {
	if text == "" {
		return
	}
	m := fc.Metrics()
	w := font.MeasureString(fc, text).Ceil()
	h := (m.Ascent + m.Descent).Ceil()
	pad := 24
	// 把水印文字渲染到小图上
	src := image.NewRGBA(image.Rect(0, 0, w+2*pad, h+2*pad))
	d := &font.Drawer{
		Dst:  src,
		Src:  image.NewUniform(c),
		Face: fc,
		Dot:  fixed.P(pad, pad+m.Ascent.Ceil()),
	}
	d.DrawString(text)

	// 旋转 -30° 进瓦片图（瓦片尺寸按旋转后的包围盒 + 间距）
	ang := -30 * math.Pi / 180
	cos, sin := math.Cos(ang), math.Sin(ang)
	sw, sh := float64(src.Bounds().Dx()), float64(src.Bounds().Dy())
	rotW := int(math.Ceil(sw*math.Abs(cos) + sh*math.Abs(sin)))
	rotH := int(math.Ceil(sw*math.Abs(sin) + sh*math.Abs(cos)))
	tile := image.NewRGBA(image.Rect(0, 0, rotW+160, rotH+140))
	// dst = R·src + t 的逆映射：src = R⁻¹·(dst - t)，t 为旋转后包围盒的平移
	tX := (float64(rotW) - (sw*cos - sh*sin)) / 2
	tY := (float64(rotH) - (sw*sin + sh*cos)) / 2
	m6 := [6]float64{
		cos, sin, -(cos*tX + sin*tY),
		-sin, cos, -(-sin*tX + cos*tY),
	}
	xdraw.ApproxBiLinear.Transform(tile, m6, src, src.Bounds(), xdraw.Over, nil)

	// 平铺
	b := img.Bounds()
	for ty := b.Min.Y - tile.Bounds().Dy(); ty < b.Max.Y; ty += tile.Bounds().Dy() {
		for tx := b.Min.X - tile.Bounds().Dx(); tx < b.Max.X; tx += tile.Bounds().Dx() {
			r := image.Rect(tx, ty, tx+tile.Bounds().Dx(), ty+tile.Bounds().Dy()).Intersect(b)
			draw.Draw(img, r, tile, r.Min.Sub(image.Pt(tx, ty)), draw.Over)
		}
	}
}

func toRGBA(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xFF}
}
