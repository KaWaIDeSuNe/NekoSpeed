// Package assets 保存项目内嵌资源（背景图、字体等），供 webui 与 render 共用。
package assets

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp"
)

func init() {
	// 确保 image.Decode 能识别 jpeg / png / webp。
	_ = jpeg.DecodeConfig
	_ = png.DecodeConfig
}

//go:embed all:bg
var bgFS embed.FS

// imageExts 认为是图片的后缀列表。
var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".gif": true, ".svg": true, ".avif": true,
}

// isImage 根据文件名判断是否为图片。
func isImage(name string) bool {
	return imageExts[strings.ToLower(filepath.Ext(name))]
}

// ListBackgroundImages 返回 bg 目录下所有图片文件名。
func ListBackgroundImages() ([]string, error) {
	entries, err := fs.ReadDir(bgFS, "bg")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !isImage(e.Name()) {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// RandomBackgroundImage 从 bg 目录随机选一张图片并解码。
// 返回解码后的图片、文件名，或 nil 表示没有可用图片。
func RandomBackgroundImage() (image.Image, string, error) {
	names, err := ListBackgroundImages()
	if err != nil {
		return nil, "", err
	}
	if len(names) == 0 {
		return nil, "", fmt.Errorf("no background images")
	}
	name := names[rand.Intn(len(names))]
	data, err := bgFS.ReadFile("bg/" + name)
	if err != nil {
		return nil, "", err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	return img, name, nil
}

// RandomBackgroundURL 返回随机背景图片在 Web UI 中的 URL 路径。
func RandomBackgroundURL() string {
	names, err := ListBackgroundImages()
	if err != nil || len(names) == 0 {
		return ""
	}
	return "/bg/" + names[rand.Intn(len(names))]
}

// BGHandler 返回一个 HTTP handler，用于 /bg/ 下的背景图片静态文件。
func BGHandler() http.Handler {
	sub, err := fs.Sub(bgFS, "bg")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/bg/", http.FileServer(http.FS(sub)))
}

// RandomBGHandler 返回 JSON {"url": "/bg/xxx.jpg"}，用于 Web UI 随机背景。
func RandomBGHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]string{"url": RandomBackgroundURL()})
	}
}
