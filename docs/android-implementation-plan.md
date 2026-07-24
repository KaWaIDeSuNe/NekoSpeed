# NekoSpeed Android 客户端 — Phase 0-1 实施计划

> 制定日期：2026-07-28。范围经确认：仅 Phase 0（正确性修复）+ Phase 1（构建工具链 + 可编译 APK），完整 UI 留待 Session 2。

**目标**：修复全部阻塞性正确性问题，补齐构建工具链，产出可在 Android Studio 一次编译成功的 APK 项目（UI 占位，完整 UI 下个 Session）。

## 关键决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| JSON 契约修复侧 | 改 Go 端加 json tag | 一处修改、契约干净；webui 不消费这些键不受影响 |
| Android 字体方案 | 扩展 Go 字体搜索路径 | 零成本，绝大多数设备有效；个别设备缺字体留作 backlog |
| 首期范围 | Phase 0-1 | 先修正确性问题并确保能编译 APK，UI 下个会话 |
| 启动图标 | 矢量占位图标 | 零外部资源，后续可替换 |
| Go 工具链 | 项目内 `.tools/go`（go1.25.0） | 本机无系统 Go |

## Phase 0：正确性修复

### 0.1 Go 端 JSON 契约（`internal/model/model.go`）
为 7 个结构体补小写 json tag，与 Kotlin DTO 对齐：

| 结构体 | Tags |
|---|---|
| `LatencyResult` | `tlsRTT` `httpsPing` `avgTlsRTT` `avgHTTPS` `packetLoss` `httpCode` |
| `SpeedResult` | `perSecond` `avg` `max` `threads` |
| `UnlockResult` | `platform` `status` `region` `info` |
| `GeoInfo` | `ip` `country` `region` `city` `asn` `org` `isp` `stack` |
| `Topology` | `inbounds` `outbounds` |
| `InboundEntry` | `index` `geo` `cluster` `outCount` |
| `OutboundEntry` | `inboundIndex` `geo` `cluster` `nodeName` |

已确认安全：webui 不消费这些键；Go 测试直接访问字段不依赖 tag。

**新增回归测试** `internal/report/json_test.go`：断言输出含 `avgTlsRTT`/`perSecond`/`inbounds`，不含 `AvgTLSRTT`。

### 0.2 Kotlin DTO 对齐（`ResultDto.kt`）
- `LatencyResult` 补 `tlsRTT`/`httpsPing`/`httpCode` 可选字段（保真）
- `TopoEntry` 拆分为 `InboundEntry`（index/geo/cluster/outCount）与 `OutboundEntry`（inboundIndex/geo/cluster/nodeName），`Topology` 相应改两个列表

### 0.3 结果文件发现（`NekoSpeedRunner.kt`）
- 运行前清空 `outDir`（`deleteRecursively()` + `mkdirs()`）
- 结束后 glob `*_result.json` 取最新，替代硬编码 `result.json`（适配 Go 端 `<prefix>_<timestamp>_result.json` 命名，runner.go:206）

### 0.4 取消杀死进程（`NekoSpeedRunner.kt` + `RunViewModel.kt`）
- Runner 持有 `@Volatile currentProcess`，新增 `cancel()` 调 `destroyForcibly()`
- `ViewModel.cancelTest()` 先 `runner.cancel()` 再 `runJob?.cancel()`——进程死亡使 `waitFor` 返回，阻塞协程自然结束
- 日志环形限制 500 行防无界增长

### 0.5 Android 字体路径（`internal/render/font.go`）
`fontCandidates` 追加（仅含 CJK 字形字体，避免 sanitize 把中文清空）：
```
/system/fonts/NotoSansCJK-Regular.ttc
/system/fonts/NotoSansSC-Regular.otf
/system/fonts/DroidSansFallbackFull.ttf
/system/fonts/DroidSansFallback.ttf
```
桌面端行为不变，`render_test.go` 兜底回归。

## Phase 1：构建工具链 + 可编译 APK

### 1.1 `scripts/build.sh` 增加 android 目标
targets 追加 `android arm64`，产物直接输出到 `android/app/src/main/assets/nekospeed-android-arm64`。

### 1.2 `.gitignore` 追加
```
android/app/src/main/assets/nekospeed-android-arm64
android/local.properties
android/.gradle/
android/**/build/
```
（约 23MB 二进制不入库，由脚本产出）

### 1.3 Gradle Wrapper 补齐
- 手写标准 `gradlew` / `gradlew.bat` 脚本
- `gradle-wrapper.jar`：从 Gradle 官方渠道获取 8.5 对应版本；失败则降级为 Android Studio 打开后执行一次 `gradle wrapper`

### 1.4 最小入口类（Manifest 已引用，缺失即编译失败）
- `NekoSpeedApp.kt`：空 `Application` 子类占位
- `MainActivity.kt`：`ComponentActivity` + material3 `Scaffold` 占位页（项目名 + "UI 开发中"），不接导航

### 1.5 矢量启动图标（minSdk 26，adaptive icon 全覆盖，无需 PNG 兜底）
- `res/drawable/ic_launcher_foreground.xml`：矢量占位图（速度仪表盘样式）
- `res/values/colors.xml`：`ic_launcher_background` 品牌色
- `res/mipmap-anydpi-v26/ic_launcher.xml`：adaptive icon 组合

### 1.6 交叉编译并验证二进制
用 `.tools/go/bin/go` 执行 `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"`，产物 `file` 验证为 ELF aarch64。

### 1.7 文档同步
`docs/android-progress.md`：勾选完成项，记录决策；注明 APK 编译验证步骤（Android Studio 打开 `android/` → 同步 → assembleDebug）。

## 验证清单

| 项 | 命令/方式 | 环境 |
|---|---|---|
| Go 编译 | `.tools/go/bin/go build ./...` | 本机 |
| Go 测试（含新契约测试） | `.tools/go/bin/go test ./...` | 本机 |
| arm64 交叉编译 | 1.6 步骤 + `file` 验证 ELF aarch64 | 本机 |
| APK 编译 | `./gradlew assembleDebug` | Android Studio 环境（手动验证） |
| 真机冒烟（释放二进制+跑订阅） | 安装 APK 实测 | 同上，下个 Session |

## 风险与备注
- **字体运行时风险**：极个别设备无 CJK 系统字体仍会导致运行失败，已记录为后续"内置字体兜底" backlog 项
- **Go 二进制体积**：约 23MB，APK 中可考虑压缩存储（后续优化项，本期不做）
- **Kotlin 单元测试**：需 JDK17 环境才能跑，连同 DTO 解析测试一并延后到 Session 2
- **预计改动文件**：Go 2 个（model.go、font.go）+ 新增 1 个测试；Kotlin 3 个（ResultDto、NekoSpeedRunner、RunViewModel）+ 新增 2 个入口类；资源 3 个新文件；脚本/配置 3 个；文档 2 个
