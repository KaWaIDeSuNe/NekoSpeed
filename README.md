# NekoSpeed

机场（代理订阅）测速工具。参考 [miaospeed](https://github.com/AirportR/miaospeed) 架构（mihomo 内核作拨号层），输入订阅链接或 Clash 配置，一次性跑完测试并输出 MiaoKo 风格的 PNG 结果图 + JSON。

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

> 本项目以 **GPL-3.0** 开源。因核心依赖 [mihomo](https://github.com/MetaCubeX/mihomo)（GPL-3.0）静态链接进二进制，本项目整体遵循 GPL-3.0 授权。详见 [LICENSE](LICENSE) 与下方[开源与合规](#开源与合规)章节。

## 功能

- **测速**：多/单线程下载测速（每秒速度序列、平均、峰值）、TLS RTT、HTTPS 延迟（httptrace 精确计时，TLS1.3）
- **本地网络标注**：自动探测本机公网 IP 的运营商/城市，并在测速图/综合图标题显示（如 `上海电信@1Gbps`），可手动指定带宽
- **流媒体解锁**：YouTube / Netflix / Disney+ / B站港澳台 / TikTok / DAZN / Abema / 巴哈姆特 / Steam 货币 / Spotify
- **AI 解锁**：OpenAI / Copilot / Gemini / MetaAI / Claude / Perplexity / Sora
- **线路分析**：入口（节点服务器 IP 归属）/ 出口（落地 IP 归属）聚合表，GeoLite2 mmdb 或在线 API
- **UDP NAT 类型**：STUN（RFC 5780）检测 FullCone / Restricted / PortRestricted / Symmetric
- **内嵌 Web UI**：`-web` 一键启动浏览器操作界面（填订阅、实时日志、查看/下载结果图），零额外文件
- **IP 预检**：多公共 DNS 采样节点服务器 IP + TCP 探测，锁定当前可达 IP，避免 DDNS 轮换 IP 池中被封锁的死 IP 导致整节点失败
- **输出文件名**：自动从订阅 URL/文件/内容中提取订阅名称，作为输出文件前缀（如 `一分机场_20260724_105252_speed.png`），也可用 `-name` 手动指定
- **随机主题**：以 16 款柔和浅色主题为主（带 subtle 背景花纹），另有 4 款深色精选与 `default`，每次运行随机一套；也可通过 `-theme` 指定

## 构建

需要 Go 1.22+：

```bash
go build -o nekospeed ./cmd/nekospeed
```

### 跨平台编译（含 Windows）

```bash
./scripts/build.sh
```

产物在 `dist/`：`nekospeed-windows-amd64.exe`、`nekospeed-windows-arm64.exe`、`nekospeed-darwin-*`、`nekospeed-linux-*`。

## Windows 部署

方式一（推荐）：拿源码在任意有 Go 的机器上执行 `./scripts/build.sh`，把 `dist/nekospeed-windows-amd64.exe` 拷到 Windows 机器即可，**无需安装任何依赖**（单文件、CGO 关闭）。

方式二：在 Windows 本机构建——安装 [Go for Windows](https://go.dev/dl/) 后在 PowerShell 执行：

```powershell
go build -o nekospeed.exe ./cmd/nekospeed
```

使用（PowerShell / CMD 均可）：

```powershell
.\nekospeed.exe -sub "https://example.com/sub" -o out
.\nekospeed.exe -sub "https://example.com/sub" -tests speed -single
```

### Windows 部署 WebUI

WebUI 已内嵌在二进制中（`go:embed`），无需任何额外文件：

```powershell
.\nekospeed-windows-amd64.exe -web 18189
```

- 启动后自动打开浏览器，或手动访问 `http://localhost:18189`
- 局域网其他设备访问：`http://<Windows的IP>:18189`
- 浏览器里填订阅 → 选测试项 → 实时日志 → 查看/下载 PNG 结果图与 JSON
- 长期运行可用 `start /min` 后台运行，或注册为计划任务开机自启

Windows 注意事项：

- **字体**：自动探测 `C:\Windows\Fonts\msyh.ttc`（微软雅黑）与 `simhei.ttf`，正常 Windows 系统开箱即用；渲染报字体错误时用 `-font` 指定一个 ttf/ttc 中文字体。
- **TUN 类代理**：本机若开着 Clash/V2rayN 的 TUN 模式，请关闭后再测，否则入口分析与 NAT 结果会失真。
- 结果图默认写到 `out\` 目录，Windows 防火墙若提示联网请选择允许。

## 用法

```bash
# 全量测试（速度+流媒体+AI+线路+NAT）
./nekospeed -sub "https://example.com/sub" -o out/

# 只测速度，单线程
./nekospeed -sub "https://example.com/sub" -tests speed -single

# 本地 Clash 配置 + 过滤节点 + GeoIP 数据库
./nekospeed -config clash.yaml -tests speed,media -filter "香港|日本" -mmdb GeoLite2-ASN.mmdb

# 启动内嵌 Web UI（浏览器操作，实时日志+查看结果图）
./nekospeed -web 18189
```

### 参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `-sub` | - | 订阅链接或本地文件（Clash YAML 或分享链接订阅，自动识别） |
| `-config` | - | 本地 Clash 配置文件（等价 `-sub` 本地文件） |
| `-tests` | `all` | 逗号分隔：`speed,media,ai,topo,nat` 或 `all` |
| `-threads` | 8 | 测速线程数（1-32） |
| `-duration` | 10 | 测速时长秒（1-30） |
| `-single` | false | 单线程测速 |
| `-filter` | - | 节点名过滤正则 |
| `-concurrency` | 1 | 同时测试的节点数（默认 1，避免节点间抢占带宽；可加大快速筛查） |
| `-o` | `out` | 输出目录 |
| `-mmdb` | - | MaxMind GeoIP2 mmdb 路径（不给则用 ip-api.com / ipinfo.io） |
| `-font` | - | 自定义中文字体（默认自动探测 PingFang / Noto CJK / 雅黑） |
| `-watermark` | 无 | 图片水印文字（默认无水印） |
| `-bandwidth` | 自动 | 本地带宽标签（如 `1Gbps` / `1000Mbps`），未指定时取所有节点测速峰值 |
| `-name` | 自动 | 订阅名称，用于输出文件名前缀（覆盖自动提取） |
| `-theme` | 随机 | 渲染主题，如 `mint_light` / `rose_light` / `nord_dark` / `default`，为空时随机 |
| `-speed-url` | - | 自定义测速文件 URL（默认 Google / Cloudflare 大文件） |
| `-ping-url` | `http://www.gstatic.com/generate_204` | 延迟测试地址 |
| `-ping-count` | 4 | 每节点延迟测试次数 |
| `-preflight` | true | 测试前多 DNS 采样服务器 IP 并 TCP 探测，锁定可达 IP（应对 DDNS 轮换 IP 被封锁） |
| `-web` | - | 启动内嵌 Web UI（如 `:18189` 或 `18189`），非空即启用 |

### 支持的订阅格式

- Clash YAML（`proxies:` 列表），支持 `anytls` 等 mihomo 全部协议
- 分享链接订阅（base64 或明文逐行）：`ss://` `ssr://` `vmess://` `vless://` `trojan://` `hysteria2://`/`hy2://` `tuic://` `snell://`（`anytls://` 分享链接暂不支持，请用 Clash YAML 形式）

### 排障

本地测试出现"全部节点丢包 100%、速度为 0 但显示成功"时，先看 [docs/troubleshooting.md](docs/troubleshooting.md)：常见原因是机场 DDNS 轮换 IP 中部分被运营商/GFW 封锁，重跑一次通常可恢复。

### 输出

输出目录下按时间戳生成：

- `*_speed.png` — 测速图（延迟色块、平均/最高速度、每秒速度迷你柱状图、UDP 类型）
- `*_media.png` — 流媒体解锁图
- `*_ai.png` — AI 可用性检测图
- `*_full.png` — 综合图（测速 + 解锁合并宽图，选 speed 且选 media/ai 时生成）
- `*_topology.png` — 线路分析图（入口/出口表，选 topo 时生成）
- `*_result.json` — 全部原始数据（延迟序列、每秒速度、解锁状态、GeoIP、NAT）

## 架构

```
cmd/nekospeed     CLI 入口与编排
internal/
  subscription    订阅拉取与解析（分享链接 → clash proxy map）
  nodepool        mihomo adapter.ParseProxy 拨号封装
  tester          延迟（httptrace）/ 下载测速 / STUN NAT
  unlock          流媒体与 AI 平台解锁检测
  geo             GeoIP（mmdb / 在线 API）与出入口聚合
  render          PNG 表格图渲染（x/image 手绘，无外部绘图依赖）
  report          JSON 输出
  model           共享数据结构
```

## Android 客户端

Android 原生客户端（Jetpack Compose + 预编译 Go 二进制），源码位于 `android/`，APK 发布在 [GitHub Releases](https://github.com/KaWaIDeSuNe/NekoSpeed/releases)。

- **功能**：订阅测速（URL / 粘贴内容）、五项测试（测速/流媒体/AI/拓扑/NAT）、节点/解锁/拓扑/图片四 Tab 报告、报告图预览与保存相册、历史记录（自动归档可查看/删除）、前台服务保活、VPN 冲突检测
- **适配**：arm64 / amd64 双架构；内置 NotoSansCJKsc 中文字体（厂商 ROM 如 HyperOS 魔改系统字体时兜底）；兼容 Android 14+ SELinux（经 `linker64` 执行）
- **隐私**：订阅 URL 落盘后以文件路径传给二进制，订阅 token 不暴露于进程命令行；订阅解析失败日志脱敏
- **构建**：
  ```bash
  ./scripts/build.sh          # 交叉编译 arm64/amd64 二进制 → assets（含字体下载）
  cd android
  ./gradlew assembleDebug     # 调试版
  ./gradlew assembleRelease   # 正式版（需 android/keystore.properties 签名配置）
  ```
- **签名**：正式版需自备 keystore（生成后务必备份，丢失将无法升级覆盖已安装版本）

## 开源与合规

本项目以 **GPL-3.0** 发布（见 [LICENSE](LICENSE)），以下为第三方依赖与数据的合规说明：

### mihomo（GPL-3.0）

- 版本：`github.com/metacubex/mihomo v1.19.29`
- 用途：`internal/nodepool` 通过 `adapter.ParseProxy` 解析节点配置并作拨号层，支持 mihomo 全部代理协议（anytls 等）
- 合规：mihomo 以 GPL-3.0 授权，本项目静态链接并整体以 GPL-3.0 开源，即满足 GPL-3.0 的分发义务。分发二进制（CLI / WebUI / Android APK）时请一并提供本仓库源码。

### 中文字体（SIL OFL 1.1）

- Android 客户端内置 `NotoSansCJKsc-Regular.otf`（思源黑体，Google Noto CJK 项目）
- 许可：[SIL Open Font License 1.1](https://openfontlicense.org/)，允许自由使用、修改与再分发（再分发需保留许可声明）
- 用途：厂商 ROM 系统字体（如 HyperOS）可能无法被正确解析时，经 `-font` 参数兜底渲染报告图

### GeoIP 数据与在线 API

- `GeoLite2` mmdb 数据库（如使用）：MaxMind 要求署名，[CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/)，需保留 MaxMind 版权声明
- 在线查询（未提供 mmdb 时）：`ip-api.com`（非商业用途免费）、`ipinfo.io`（免费版需按条款署名），请自行确认各自使用条款

### 其他第三方依赖

均为宽松许可，按各自 LICENSE 要求保留版权声明：

| 依赖 | 许可 |
|---|---|
| `golang.org/x/image` | BSD-3-Clause |
| `github.com/miekg/dns` | BSD-3-Clause |
| `gopkg.in/yaml.v3` | MIT |
| `github.com/pion/stun` | MIT |
| `github.com/oschwald/maxminddb-golang` | MIT |
| Android：Compose / Navigation / Coil 等 | Apache-2.0 |

### 免责声明

- 本项目仅供网络质量测试与学习交流，请遵守所在地区法律法规与机场服务条款
- 测速与解锁检测基于各平台公开页面特征，结果仅供参考，平台改版可能导致检测项失效

## 说明

- 测速与解锁结果受本地网络、节点负载影响，仅供参考。
