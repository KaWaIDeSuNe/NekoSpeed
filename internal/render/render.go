// Package render 把测速结果渲染成复刻 MiaoKo（喵速）风格的 PNG 表格图。
//
// 通用风格：浅色主题——白底、浅灰表头黑字居中、数据行白底行间细灰线、
// 序号列红色数字居中、节点名称列左对齐、无圆角无阴影，整图叠斜向平铺水印。
package render

import (
	"image"
	"image/png"
	"os"
	"time"
)

// Meta 渲染元信息，由调用方组装。
type Meta struct {
	Version     string    // 版本号，如 "0.1.0"
	Threads     int       // 测速线程数
	Total       int       // 节点总数
	Succeeded   int       // 成功节点数
	StartTime   time.Time // 测试开始时间
	Watermark   string    // 水印文字
	FontPath    string    // 自定义字体路径（可空，自动探测）
	Single      bool      // 是否单线程测速
	LocalISP    string    // 本地运营商，如 "Shanghai China Telecom"
	Bandwidth   string    // 本地带宽，如 "1Gbps" / "500Mbps"
	Theme       Theme     // 配色主题，空值时 renderTables 自动随机选择
}


// LocalISPTitle 根据本地运营商和带宽信息生成标题。
// base 是标题前缀（应包含 " - "），defaultSuffix 是本地运营商未识别时的默认后缀。
// localISP 为空时返回 base + defaultSuffix；否则用 "运营商@带宽" 替换后缀，
// 并可选在单线程时加上 "单线程" 前缀。
func LocalISPTitle(base, defaultSuffix, localISP, bandwidth string, single bool) string {
	if localISP == "" {
		return base + defaultSuffix
	}
	suffix := localISP
	if bandwidth != "" {
		suffix += "@" + bandwidth
	}
	if single {
		suffix = "单线程" + suffix
	}
	return base + suffix
}

func savePNG(img *image.RGBA, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
