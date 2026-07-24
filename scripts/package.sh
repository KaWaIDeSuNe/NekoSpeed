#!/usr/bin/env bash
# 交叉编译并按平台打包分发：dist/release/nekospeed_<版本>_<os>_<arch>.{zip,tar.gz}
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION=$(git describe --tags --always 2>/dev/null || echo "v0.1.0")
./scripts/build.sh

REL="dist/release"
rm -rf "$REL"
mkdir -p "$REL"

for bin in dist/nekospeed-*; do
  base=$(basename "$bin")
  # nekospeed-windows-amd64.exe -> windows-amd64
  plat=${base#nekospeed-}
  plat=${plat%.exe}
  name="nekospeed_${VERSION}_${plat}"
  stage="dist/.stage/$name"
  rm -rf "$stage"
  mkdir -p "$stage"
  cp "$bin" "$stage/"
  cp README.md "$stage/"
  if [[ "$base" == *.exe ]]; then
    (cd dist/.stage && zip -q -r "../release/$name.zip" "$name")
    echo "==> $REL/$name.zip"
  else
    (cd dist/.stage && tar czf "../release/$name.tar.gz" "$name")
    echo "==> $REL/$name.tar.gz"
  fi
done
rm -rf dist/.stage
echo "完成，分发包在 $REL/"
