package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// drawSparkline 在给定矩形内画竖条迷你柱状图：每根竖条对应一个采样，
// 高度按 log1p 归一，使用传入颜色 c。
func drawSparkline(img *image.RGBA, r image.Rectangle, vals []float64, c color.RGBA) {
	if len(vals) == 0 {
		return
	}
	inner := r.Inset(4)
	if inner.Dx() < len(vals) || inner.Dy() < 4 {
		return
	}
	maxV := 0.0
	for _, v := range vals {
		if v > maxV {
			maxV = v
		}
	}
	if maxV <= 0 {
		return
	}
	logMax := math.Log1p(maxV)
	bw := float64(inner.Dx()) / float64(len(vals))
	base := inner.Max.Y
	for i, v := range vals {
		if v <= 0 {
			continue
		}
		h := int(math.Round(math.Log1p(v) / logMax * float64(inner.Dy())))
		if h < 2 {
			h = 2
		}
		x0 := inner.Min.X + int(float64(i)*bw)
		x1 := inner.Min.X + int(float64(i+1)*bw)
		if x1-x0 > 1 {
			x1-- // 条间留 1px 缝
		}
		if x1 <= x0 {
			x1 = x0 + 1
		}
		bar := image.Rect(x0, base-h, x1, base)
		draw.Draw(img, bar, image.NewUniform(c), image.Point{}, draw.Src)
	}
}
