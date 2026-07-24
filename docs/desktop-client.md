# NekoSpeed 桌面端可视化客户端开发文档（Windows + macOS）

## 1. 目标

为 NekoSpeed 提供 **Windows `.exe` 和 macOS `.app` 原生可视化客户端**，且**不依赖 WebView / WebUI**。

核心功能：
- 输入/导入机场订阅
- 选择测试项（speed / media / ai / topo / nat）
- 实时显示运行日志
- 展示结果：节点列表、速度、解锁矩阵、拓扑
- 查看 NekoSpeed 生成的 PNG 结果图
- 导出/分享 JSON 与 PNG

## 2. 技术选型：Fyne

| 方案 | 是否 WebView | 跨平台 | 与 NekoSpeed 集成 | 推荐 |
|---|---|---|---|---|
| **Fyne** | ❌ 原生自绘 | ✅ Win/Mac/Linux | ✅ 同 Go，直接 import runner | **推荐** |
| Tauri / Wails / Electron | ✅ 浏览器内核 | ✅ Win/Mac | ❌ 需起 HTTP 服务 | 否 |
| Flutter Desktop | ❌ 原生 | ✅ Win/Mac | ⚠️ 需 Go FFI / 子进程 | 备选 |
| Avalonia / .NET MAUI | ❌ 原生 | ✅ Win/Mac | ⚠️ 需 C# + 调用二进制 | 备选 |
| Qt | ❌ 原生 | ✅ Win/Mac | ⚠️ C++ 生态重 | 备选 |

Fyne 是纯 Go 的跨平台 GUI 库，直接调用 NekoSpeed 内部 `runner.Run`，无需子进程、无需 WebView，最适合当前项目。

## 3. 架构

```
cmd/nekospeed-desktop/          # 桌面端入口
    ├── main.go                 # Fyne App 初始化 + 主题
    └── ui/
        ├── app.go              # 主窗口、路由、状态管理
        ├── input.go            # 订阅输入 / 测试参数表单
        ├── log.go              # 实时日志面板
        ├── result.go           # 结果页（Tab 容器）
        ├── nodelist.go         # 节点列表
        ├── speed.go            # 速度展示
        ├── unlock.go           # 解锁矩阵
        ├── topology.go         # 拓扑表
        └── image.go            # 展示 PNG 结果图

internal/runner/               # 直接复用 NekoSpeed 核心逻辑
```

## 4. 为什么不依赖 WebUI

NekoSpeed 的 `internal/webui` 虽然方便，但桌面端直接用 WebView 有以下缺点：

- 需要额外启动本地 HTTP 服务，增加端口和生命周期管理复杂度。
- 不同系统 WebView 表现不一致，调试困难。
- 原生交互（菜单、托盘、快捷键、文件拖拽）受限。
- 产物体积和运行时开销更高。

Fyne 方案直接调用 Go API，避开以上问题。

## 5. 核心集成方式

```go
package main

import (
    "context"
    "nekospeed/internal/runner"
)

func runTests(ctx context.Context, opts runner.Options, logf func(string, ...any)) (*runner.Result, error) {
    return runner.Run(ctx, opts, logf)
}
```

`runner.Run` 接收一个日志回调，桌面端 UI 可以实时把日志追加到面板：

```go
out, err := runner.Run(ctx, runner.Options{
    Source:      subURL,
    Tests:       "speed,media,ai",
    Threads:     8,
    Duration:    10,
    Concurrency: 1,
}, func(format string, args ...any) {
    ui.AppendLog(fmt.Sprintf(format, args...))
})
```

测试结果直接拿到 `*runner.Result` 结构体，渲染到原生 UI 组件。

## 6. 项目目录结构

```
NekoSpeed/
├── cmd/
│   └── nekospeed-desktop/         # 桌面端入口
│       ├── main.go
│       └── ui/                    # Fyne UI 模块
├── internal/runner/               # 已存在，桌面端直接复用
├── internal/render/               # 可复用 PNG 生成逻辑
├── scripts/
│   └── build-desktop.sh           # 编译 Win/Mac 桌面端
└── docs/desktop-client.md         # 本文档
```

## 7. UI 模块规划

### 7.1 输入页（input.go）

- 订阅 URL 输入框
- 测试项多选框：speed / media / ai / topo / nat
- 高级参数：threads、duration、concurrency、ping-count、filter
- 开始测试按钮
- 主题切换（浅色/深色）

### 7.2 运行页（log.go）

- 进度条（通过日志行数估算或 runner 返回状态）
- 实时日志滚动框
- 取消按钮

### 7.3 结果页（result.go）

Tab 导航：

| Tab | 内容 |
|---|---|
| 节点 | 节点列表：名称、类型、延迟、速度、状态 |
| 速度 | 平均速度、最高速度、每秒速度序列（表格或 PNG 图） |
| 解锁 | 媒体/AI 平台解锁矩阵，色块表示成功/失败 |
| 拓扑 | 入口/出口 IP、国家、地区、ASN、ISP |
| 图片 | 展示 NekoSpeed 生成的 PNG 结果图 |

### 7.4 数据展示

Fyne 组件选择：
- 列表/表格：`widget.List` / `widget.Table`
- 日志：`widget.Entry`（只读）
- 图片：`canvas.Image`
- 进度：`widget.ProgressBarInfinite` / `widget.ProgressBar`
- 弹窗：`dialog.ShowInformation` / `dialog.ShowFileSave`

## 8. 构建流程

### 8.1 依赖

```bash
go get fyne.io/fyne/v2
```

### 8.2 开发运行

```bash
go run ./cmd/nekospeed-desktop
```

### 8.3 交叉编译 Windows `.exe`

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  CC=x86_64-w64-mingw32-gcc \
  go build -ldflags "-H=windowsgui" -o dist/nekospeed-desktop-windows.exe ./cmd/nekospeed-desktop
```

> Windows 需要 mingw-w64 工具链；`-H=windowsgui` 避免启动控制台窗口。

### 8.4 交叉编译 macOS `.app`

```bash
GOOS=darwin GOARCH=amd64 go build -o dist/nekospeed-desktop-darwin-amd64 ./cmd/nekospeed-desktop
GOOS=darwin GOARCH=arm64 go build -o dist/nekospeed-desktop-darwin-arm64 ./cmd/nekospeed-desktop
```

使用 `fyne package` 可直接生成 `.app` 目录：

```bash
fyne package -os darwin -appID com.nekospeed.desktop -name NekoSpeed
fyne package -os windows -appID com.nekospeed.desktop -name NekoSpeed
```

Fyne 自带打包工具，推荐用 `fyne-cross` 或 CI 脚本统一构建。

### 8.5 推荐打包脚本

`scripts/build-desktop.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist

fyne package -os windows -appID com.nekospeed.desktop -name NekoSpeed \
  -sourceDir cmd/nekospeed-desktop

fyne package -os darwin -appID com.nekospeed.desktop -name NekoSpeed \
  -sourceDir cmd/nekospeed-desktop

echo "桌面端产物在 dist/"
```

## 9. 注意事项

- **CGO 依赖**：Fyne 依赖 OpenGL/Metal 等图形库，Windows 构建需要 mingw，macOS 用 Xcode Command Line Tools。
- **字体**：Fyne 内置基础字体，但中文显示需要加载系统字体（如 `PingFang SC`、`Microsoft YaHei`）。
- **网络权限**：桌面端无需特殊权限，但 Windows 防火墙首次联网可能弹提示。
- **本地代理冲突**：运行 NekoSpeed 时若系统开启全局代理，测速/入口分析结果可能失真，需提示用户关闭。
- **后台执行**：测试耗时较长，Fyne 主线程不能阻塞，测试放在 goroutine 中。
- **GPL 合规**：NekoSpeed 依赖 mihomo（GPL-3.0），分发桌面端需遵守 GPL，源码可访问。
- **代码签名**：macOS 分发建议签名，否则用户需“右键打开”或关闭 Gatekeeper。

## 10. 实现步骤

1. 引入 `fyne.io/fyne/v2` 依赖。
2. 创建 `cmd/nekospeed-desktop/main.go`，初始化 Fyne App 和主窗口。
3. 实现输入页：订阅、测试项、参数、主题切换。
4. 实现日志/运行页： goroutine 调用 `runner.Run`，实时回调更新 UI。
5. 实现结果页：节点列表、速度、解锁矩阵、拓扑、图片查看。
6. 实现文件导出：JSON/PNG 保存对话框。
7. 编写 `scripts/build-desktop.sh` 构建 Windows / macOS 产物。
8. 测试并打包 `.exe` 和 `.app`。

## 11. 与 Android 客户端对比

| 维度 | Android | 桌面端（Fyne） |
|---|---|---|
| 技术栈 | Kotlin + Jetpack Compose | Go + Fyne |
| 调用方式 | 嵌入 NekoSpeed 二进制，ProcessBuilder 执行 | 直接 import `internal/runner` 调用 |
| 是否需要子进程 | 是 | 否 |
| 是否需要 WebView | 否 | 否 |
| 结果解析 | 解析 JSON | 直接拿到 Go 结构体 |

---

> 仅供学习研究使用，请勿用于违反当地法律法规的场景。
