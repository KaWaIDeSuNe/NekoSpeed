#!/usr/bin/env bash
# 交叉编译各平台 NekoSpeed 二进制到 dist/
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
mkdir -p dist

# 内置中文字体（厂商 ROM 系统字体解析失败时经 -font 兜底；SIL OFL 许可）
font_dst="android/app/src/main/assets/fonts/NotoSansCJKsc-Regular.otf"
if [ ! -f "$font_dst" ]; then
  echo "==> 下载内置中文字体 NotoSansCJKsc-Regular.otf (~16MB)"
  mkdir -p "$(dirname "$font_dst")"
  curl -fsSL -o "$font_dst" \
    "https://cdn.jsdelivr.net/gh/notofonts/noto-cjk@main/Sans/OTF/SimplifiedChinese/NotoSansCJKsc-Regular.otf" \
    || echo "!! 字体下载失败：请手动放置到 $font_dst"
fi


targets=(
  "windows amd64 .exe"
  "windows arm64 .exe"
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
)

for t in "${targets[@]}"; do
  read -r goos goarch ext <<<"$t"
  out="dist/nekospeed-${goos}-${goarch}${ext}"
  echo "==> $out"
  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch \
    go build -trimpath -ldflags "-s -w" -o "$out" ./cmd/nekospeed
done

ndk_bin="${ANDROID_NDK:-$HOME/Library/Android/sdk/ndk/*/toolchains/llvm/prebuilt/darwin-x86_64/bin}"
arm64_cc=$(ls $ndk_bin/aarch64-linux-android21-clang 2>/dev/null | head -1)
amd64_cc=$(ls $ndk_bin/x86_64-linux-android21-clang 2>/dev/null | head -1)

for android_arch in arm64 amd64; do
  android_out="android/app/src/main/assets/nekospeed-android-${android_arch}"
  echo "==> $android_out (raw)"
  if [ -z "$arm64_cc" ] && [ -z "$amd64_cc" ]; then
    echo "!! 未找到 NDK clang（aarch64/x86_64-linux-android21-clang），跳过 Android 构建"
    continue
  fi
  # 统一走 NDK cgo 动态链接（bionic），适配 Android 14+ SELinux 需经 linker64 执行
  cc=$([ "$android_arch" = "arm64" ] && echo "$arm64_cc" || echo "$amd64_cc")
  CGO_ENABLED=1 GOOS=android GOARCH=$android_arch CC="$cc" \
    go build -trimpath -ldflags "-s -w" -o "$android_out" ./cmd/nekospeed

  echo "==> $android_out.gz (compressed)"
  gzip -f "$android_out"
done

echo "完成，桌面产物在 dist/，Android 产物在 android/app/src/main/assets/"
