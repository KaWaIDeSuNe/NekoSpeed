package render

import (
	"image/color"
)

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 0xFF} }

// gradStop 渐变锚点，pos 取值 0~1。
type gradStop struct {
	pos float64
	c   color.RGBA
}

// gradient 按 t(0~1) 在锚点间线性插值。
func gradient(stops []gradStop, t float64) color.RGBA {
	if t <= stops[0].pos {
		return stops[0].c
	}
	if t >= stops[len(stops)-1].pos {
		return stops[len(stops)-1].c
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].pos {
			a, b := stops[i-1], stops[i]
			k := (t - a.pos) / (b.pos - a.pos)
			mix := func(x, y uint8) uint8 {
				return uint8(float64(x) + (float64(y)-float64(x))*k)
			}
			return rgb(mix(a.c.R, b.c.R), mix(a.c.G, b.c.G), mix(a.c.B, b.c.B))
		}
	}
	return stops[len(stops)-1].c
}
