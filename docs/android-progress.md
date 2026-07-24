# NekoSpeed Android 客户端 — 实施进度

## 1. 项目概述

为 NekoSpeed 打造 Android 原生 GUI。架构：预编译 Go 二进制嵌入 APK assets → ProcessBuilder 调用 → 解析 result.json → Jetpack Compose 渲染。

## 2. 项目文件结构

```
android/
├── app/
│   ├── src/main/
│   │   ├── assets/
│   │   │   ├── .gitkeep
│   │   │   └── nekospeed-android-arm64     # Go 交叉编译产物（需手动构建）
│   │   ├── java/com/nekospeed/android/
│   │   │   ├── MainActivity.kt
│   │   │   ├── NekoSpeedApp.kt
│   │   │   ├── data/
│   │   │   │   ├── NekoSpeedRunner.kt
│   │   │   │   ├── ResultRepository.kt
│   │   │   │   └── model/ResultDto.kt
│   │   │   ├── ui/
│   │   │   │   ├── navigation/NavRoutes.kt
│   │   │   │   ├── screens/
│   │   │   │   │   ├── InputScreen.kt
│   │   │   │   │   ├── RunningScreen.kt
│   │   │   │   │   └── ResultScreen.kt
│   │   │   │   ├── components/
│   │   │   │   │   ├── NodeList.kt
│   │   │   │   │   ├── UnlockGrid.kt
│   │   │   │   │   └── TopologyTable.kt
│   │   │   │   └── theme/
│   │   │   │       ├── Color.kt
│   │   │   │       ├── Theme.kt
│   │   │   │       └── Type.kt
│   │   │   └── viewmodel/
│   │   │       └── RunViewModel.kt
│   │   ├── AndroidManifest.xml
│   │   └── res/values/
│   │       ├── strings.xml
│   │       └── themes.xml
│   └── build.gradle.kts
├── build.gradle.kts
├── gradle.properties
├── gradle/wrapper/gradle-wrapper.properties
├── local.properties
└── settings.gradle.kts
```

## 3. 当前进度

| 模块 | 文件 | 状态 | 备注 |
|------|------|:----:|------|
| 文档 | `docs/android-progress.md` | ✅ | 本文档 |
| 文档 | `docs/android-implementation-plan.md` | ✅ | Phase 0-1 实施计划（2026-07-28） |
| 构建系统 | `android/settings.gradle.kts` | ✅ | |
| 构建系统 | `android/build.gradle.kts` | ✅ | |
| 构建系统 | `android/app/build.gradle.kts` | ✅ | |
| 构建系统 | `android/gradle.properties` | ✅ | |
| 构建系统 | `android/gradle/wrapper/gradle-wrapper.properties` | ✅ | Gradle 8.5 |
| 构建系统 | `android/gradlew` / `gradlew.bat` / `gradle-wrapper.jar` | ✅ | |
| 清单 | `android/app/src/main/AndroidManifest.xml` | ✅ | |
| 资源 | `android/app/src/main/res/values/strings.xml` | ✅ | |
| 资源 | `android/app/src/main/res/values/themes.xml` | ✅ | |
| 资源 | `android/app/src/main/res/values/colors.xml` | ✅ | 品牌色 #D63031 |
| 资源 | `res/drawable/ic_launcher_foreground.xml` + `mipmap-anydpi-v26/ic_launcher.xml` | ✅ | 矢量占位图标（仪表盘） |
| 数据层 | `data/model/ResultDto.kt` | ✅ | 已与 Go JSON 契约对齐（0.1/0.2） |
| 数据层 | `data/NekoSpeedRunner.kt` | ✅ | 结果文件 glob + 取消杀进程 + gzip 解压（0.3/0.4/Session 3） |
| 数据层 | `data/ResultRepository.kt` | ✅ | |
| 服务层 | `service/NekoSpeedService.kt` | ✅ | 前台服务运行二进制，通知栏取消 |
| 服务层 | `service/NekoSpeedServiceState.kt` | ✅ | ServiceState + 日志总线 |
| 工具 | `util/ShareHelper.kt` | ✅ | FileProvider 分享 JSON/PNG |
| 工具 | `util/VpnDetector.kt` | ✅ | VPN/代理冲突检测 |
| ViewModel | `viewmodel/RunViewModel.kt` | ✅ | 日志环形 500 行，前台服务协调 |
| 主题 | `ui/theme/Color.kt` | ✅ | Material3 静态配色 + 品牌红 |
| 主题 | `ui/theme/Theme.kt` | ✅ | System/Light/Dark/NekoSpeed 四模式 |
| 主题 | `ui/theme/Type.kt` | ✅ | Typography |
| UI | `ui/navigation/NavRoutes.kt` | ✅ | 单 Activity + Compose Navigation |
| UI | `ui/screens/InputScreen.kt` | ✅ | 订阅输入 + 测试项 + 高级参数 + 主题切换 |
| UI | `ui/screens/RunningScreen.kt` | ✅ | 实时日志 + 进度 + 取消 + 自动滚动 |
| UI | `ui/screens/ResultScreen.kt` | ✅ | 节点 / 解锁 / 拓扑 Tab |
| UI | `ui/components/NodeList.kt` | ✅ | 节点列表卡片 |
| UI | `ui/components/UnlockGrid.kt` | ✅ | 平台 × 节点 解锁状态色块 |
| UI | `ui/components/TopologyTable.kt` | ✅ | 入口 / 出口明细 |
| Activity | `MainActivity.kt` | ✅ | 接入导航，替换占位页 |
| App | `NekoSpeedApp.kt` | ✅ | 空 Application 占位 |
| 构建脚本 | `scripts/build.sh`（android 目标） | ✅ | 产物 gzip 后直出 assets |
| 二进制 | `assets/nekospeed-android-arm64.gz` | ✅ | go1.25.0 交叉编译，gzip 后约 10MB（不入 git） |
| 测试 | `src/test/.../ResultDtoTest.kt` | ✅ | Kotlin DTO 解析契约测试 |
| Go 契约 | `internal/model/model.go` json tags | ✅ | 7 个结构体小写 tag（0.1） |
| Go 契约 | `internal/report/json_test.go` | ✅ | 键名契约回归测试 |
| Go 字体 | `internal/render/font.go` Android 路径 | ✅ | 4 个 /system/fonts 候选（0.5） |

## 4. 实施路线图

### Session 1（已完成 2026-07-28）：项目骨架 + 数据层 + ViewModel + Phase 0-1
- [x] 创建全部 Gradle 构建文件
- [x] 创建 AndroidManifest + 资源文件
- [x] 实现数据层：ResultDto、NekoSpeedRunner、ResultRepository
- [x] 实现 RunViewModel
- [x] Phase 0.1：Go model.go 补小写 json tag（7 个结构体）+ report 契约测试
- [x] Phase 0.2：Kotlin DTO 对齐（LatencyResult 补字段、TopoEntry 拆分 Inbound/Outbound）
- [x] Phase 0.3：结果文件发现改 glob `*_result.json`（适配 Go 端时间戳命名）
- [x] Phase 0.4：取消测试时 destroyForcibly 杀死进程 + 日志环形 500 行
- [x] Phase 0.5：font.go 追加 Android 系统字体路径（修复 PNG 渲染找不到字体导致整跑失败）
- [x] Phase 1.1：scripts/build.sh 追加 android arm64 目标（产物直出 assets）
- [x] Phase 1.2：.gitignore 追加（二进制/local.properties/.gradle/build）
- [x] Phase 1.3：Gradle wrapper 三件套（gradlew/gradlew.bat/gradle-wrapper.jar 8.5）
- [x] Phase 1.4：最小入口类 MainActivity（占位页）+ NekoSpeedApp
- [x] Phase 1.5：矢量启动图标（品牌红 #D63031 + 仪表盘前景）
- [x] Phase 1.6：交叉编译 arm64 二进制（23MB ELF aarch64，已验证）
- [x] Go 验证：build + 全量 test 通过（含新契约测试）

### Session 2（已完成 2026-07-29）：UI 页面与组件
- [x] 主题系统（Color / Type / Theme）支持 System / Light / Dark / NekoSpeed 四模式
- [x] 导航路由（NavRoutes）与 MainActivity 接入
- [x] InputScreen（订阅输入 + 测试项多选 + 高级参数 + 主题切换）
- [x] RunningScreen（实时日志 LazyColumn + 进度 + 取消 + 自动滚动到底部）
- [x] ResultScreen（节点 / 解锁 / 拓扑 三 Tab）
- [x] NodeList 组件（节点卡片：延迟、均速、峰值、状态）
- [x] UnlockGrid 组件（平台 × 节点 解锁状态色块）
- [x] TopologyTable 组件（入口聚合 / 出口明细）
- [x] MainActivity 接入导航（替换占位页）
- [x] 新增 Kotlin DTO 解析单元测试 `ResultDtoTest`
- [x] `app/build.gradle.kts` 追加 `junit` 测试依赖

### Session 3（已完成 2026-07-29）：构建、调试与发布准备（代码层面）
- [x] 二进制压缩存储：assets 改为 gzip，`ensureBinary()` 运行时解压；`scripts/build.sh` 自动压缩（23MB → 约 10MB）
- [x] 分享 JSON/PNG：`FileProvider` + `ShareHelper` + 系统分享 Intent
- [x] VPN 冲突检测：`VpnDetector` 检查 VPN/代理，输入页弹警告对话框
- [x] 处理后台运行：`NekoSpeedService` 前台服务 + `dataSync` 类型 + 通知栏取消按钮，ViewModel 通过 `NekoSpeedServiceBus` 同步状态/日志
- [x] 性能优化：日志 `itemsIndexed` 加 key、解锁矩阵 `unlockMap` remember、结果 Tab 列表 remember
- [x] 订阅拉取兼容：重试 + HTTP/1.1 降级 + 完整 UA/Accept 头
- [x] 粘贴订阅内容：Android InputScreen 支持 URL / 粘贴内容两种来源
- [ ] Android Studio 打开 android/ 同步并 assembleDebug 验证（本机无 SDK/JDK17，需在开发机执行）
- [ ] 真机冒烟：释放二进制 + 跑真实订阅 + 查看结果/日志

### Session 4（已完成 2026-08-03）：本机模拟器构建验证 + 真机冒烟（Intel Mac + Android 14 x86_64 模拟器）
- [x] 环境：Temurin JDK 17（brew 镜像无瓶包）+ cmdline-tools + emulator 37.2.2 + NDK r27d（腾讯/清华镜像绕 GFW；adb 5037 端口被僵尸守护进程抢占需清理后按序启动）
- [x] 双架构二进制：`scripts/build.sh` 支持 android/amd64（NDK clang 外部链接）+ android/arm64 改 cgo 动态链接（统一 bionic/linker64）
- [x] `NekoSpeedRunner` 按 `Build.SUPPORTED_ABIS` 选择 arm64/amd64 asset
- [x] 修复首编错误：`settings.gradle.kts` dependencyResolution→dependencyResolutionManagement、Button 缺 import、Theme.kt ColorWhite/Black 前向引用、RunViewModel getApplication 泛型、ResultScreen tabs 需 `@Composable () -> Unit` 显式类型
- [x] 修复 AndroidManifest 缺 `ACCESS_NETWORK_STATE`（VpnDetector 崩溃）
- [x] SELinux 修复：Android 14 拒绝直接 execve app_data_file（execute_no_trans），`NekoSpeedRunner` 增加 `/system/bin/linker64` 回退执行（enforcing 模式实测通过，进程树 app→linker64→nekospeed）
- [x] 完整冒烟 1（permissive）：43 节点 × 5 测试项（speed,media,ai,topo,nat），17m50s，43/43 成功；result.json + 5 PNG 产物；三 Tab 渲染 + 分享 2 files 正常
- [x] 快速冒烟 2（enforcing + linker64 回退）：speed-only，9m34s，43/43 成功
- [x] 已知限制：模拟器 IME 输入 `input text` 会丢尾部字符（需 ≤20 字符分段输入）；AOSP 镜像无 `cmd clipboard` 服务

### Session 5（已完成 2026-08-03）：真机字体兜底（红米 Note 12 Turbo / HyperOS 3.0.5）
- [x] 真机报错定位：`/system/fonts/NotoSansCJK-Regular.ttc: sfnt: invalid CFF table` — 厂商 ROM 魔改字体，x/image/opentype 解析失败；测试数据正常，仅 PNG 渲染阶段失败
- [x] 内置字体：`assets/fonts/NotoSansCJKsc-Regular.otf`（思源黑体完整版 16MB，SIL OFL 1.1 可再分发；jsDelivr 下载，build.sh 幂等步骤）
- [x] `NekoSpeedRunner` 新增 `ensureFont()`（assets→files）+ `-font` 参数传递；`NekoSpeedService` 启动时调用
- [x] `font.go` 追加厂商候选（MiSans/HarmonyOS/OnePlus/OPPO/vivo/SourceHanSans 等）作双保险
- [x] 模拟器回归：命令行 `-font` 渲染 speed.png 成功（8/8 节点）；UI 全流程 speed-only 43/43 成功，进程命令行确认含 `-font`（12m0s）
- [x] APK 交付：`android/app/build/outputs/apk/debug/app-debug.apk`（~63MB，含双架构二进制+字体）

### Session 6（已完成 2026-08-03）：结果归档 + 历史记录 + 图片管理
- [x] 结果归档：测试成功后拷贝 result.json+PNG 到 `files/history/<ts>/` + `index.json` 索引（`ResultRepository.archiveResult`，`NekoSpeedService` 完成时触发）
- [x] 历史记录：`HistoryScreen` 列表（时间/订阅名/成功数/耗时/图片数）+ 删除（`HistoryViewModel`、`HistoryRecord` 模型 + `HistoryIndex` 读写）
- [x] 历史详情：点击记录从归档重新加载 result.json → 复用 `ResultScreen`（三 Tab + 图片 + 分享）；`Result` 路由支持可选 `file` 参数，`onBack` 统一 `popBackStack()` 自适应来源
- [x] 图片管理：ResultScreen 新增第 4 个"图片"Tab（`ReportImages` 缩略网格 + 全屏底部弹层）
- [x] 保存到相册：`ShareHelper.saveImageToGallery` MediaStore 写入 `Pictures/NekoSpeed/`（Android 10+ 无需权限）
- [x] 单图分享：`ShareHelper.shareFile`；顶栏"分享全部"保留
- [x] 入口：InputScreen 顶栏"历史"图标 → History 路由
- [x] 测试：`HistoryRecordTest` round-trip 单测（与 `ResultDtoTest` 均通过）
- [x] 模拟器验证：43/43 speed-only 9m58s → 归档+index 正确 → 历史列表/详情/图片预览/保存相册/单图分享/删除 全链路通过

### Session 7（已完成 2026-08-03）：启动图标（猫主题）
- [x] 图标提示词 → `opencode run` 调用 `opencode-go/mimo-v2.5` 生成 2 张（冲刺猫/猫脸仪表盘）；mimo 后续生成卡顿改用 `opencode-go/glm-5.2` 补齐 3 张
- [x] `opencode-go/gpt-5.6-luna` 本地区不可用（403 region），由 glm-5.2 替代生成 5 张（卡通/黑猫/猫尾弧/几何/猫爪）
- [x] 共 10 张 SVG（1024²，品牌红 #D63031 + 白猫），qlmanage 渲染 PNG 预览页 `/tmp/neko_icons/preview.html`
- [x] 用户选定 8 号"猫尾速度弧" → 手工转换 SVG 为 VectorDrawable `ic_launcher_foreground.xml`（含 scale 0.60 安全区、速度刻度线），背景沿用 `@color/ic_launcher_background`=#D63031
- [x] assembleDebug + 安装启动正常，桌面截图 `desktop_icon.png` 供确认

### Session 8（已完成 2026-08-04）：v0.2.0 正式版发布
- [x] 签名：keytool 生成 `android/keystore/nekospeed-release.jks`（RSA 2048/10000 天）+ `keystore.properties`（均 gitignore，需本地备份）
- [x] `app/build.gradle.kts`：signingConfigs.release + buildTypes.release（R8 minify + shrinkResources）+ `proguard-rules.pro`（Compose/kotlinx-serialization/Coil keep）；versionCode 2 / versionName 0.2.0
- [x] `assembleRelease` 成功（R8 后 63MB→41MB），`apksigner verify` 签名通过
- [x] 代码入库：android/ 全部代码 + Go 改动 + 文档首次提交并推送 main（二进制/字体/keystore 由 build.sh 生成，不入库）
- [x] 发布：gh CLI 2.97.0（brew 无瓶包，直下 GitHub 二进制；token 缺 read:org 用 GH_TOKEN 环境变量驱动）→ tag v0.2.0 → `gh release create`（附 app-release.apk 42MB + 更新说明 + GPL 声明）
- [x] Release：https://github.com/KaWaIDeSuNe/NekoSpeed/releases/tag/v0.2.0
- [x] R8 冒烟：模拟器安装 release APK（签名不同需先卸载 debug）→ 启动/InputScreen/历史页序列化全部正常
- [ ] 待办：keystore 文件+密码异地备份（丢失则无法升级覆盖已装 App）；真机安装 release 自测

## 5. 关键决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| 包名 | `com.nekospeed.android` | 与项目名称一致 |
| 最低 SDK | 26 (Android 8.0) | 覆盖 95%+ 设备；ProcessBuilder API 完善 |
| 目标 SDK | 34 | 当前最新稳定 |
| 编译 SDK | 34 | |
| Kotlin | 1.9.22 | 稳定，Compose 支持好 |
| AGP | 8.2.2 | 稳定版本 |
| Compose BOM | 2024.02.00 | |
| Compose Compiler | 1.5.10 | 匹配 Kotlin 1.9.22 |
| 序列化 | Kotlinx Serialization 1.6.2 | 原生 Kotlin 支持，无需反射 |
| 图片加载 | Coil 2.5.0 | Compose 原生支持 |
| 导航 | Compose Navigation | 官方推荐 |
| 架构 | MVVM + ViewModel + StateFlow | 标准 Android 架构 |
| 并发 | Kotlin Coroutines + Dispatchers.IO | 二进制调用/JSON 解析在 IO 线程 |
| 主题 | Material3 动态取色 + 自定义 | light/dark + NekoSpeed 主题色 |
| JSON 契约修复侧 | Go 端加 json tag | 一处修改契约干净；webui 不消费这些键（2026-07-28） |
| Android 字体 | 扩展 Go 字体搜索路径 | 零成本覆盖绝大多数设备；无 CJK 字体设备留 backlog（2026-07-28） |
| 结果文件发现 | glob `*_result.json` 取最新 | 适配 Go 端 `<前缀>_<时间戳>_result.json` 命名（2026-07-28） |
| 取消机制 | destroyForcibly 杀进程 | 阻塞 waitFor 无法响应协程取消，杀进程使其自然返回（2026-07-28） |
| Android 14+ 执行二进制 | 直接 execve 失败时回退 `/system/bin/linker64` 执行 | SELinux 拒绝 untrusted_app 对 app_data_file 的 execute_no_trans（execute 放行），经系统链接器加载即可绕过；实测 enforcing 通过（2026-08-03） |
| 中文字体 | APK 内置 NotoSansCJKsc-Regular.otf + `-font` 参数 | 厂商 ROM（如 HyperOS）魔改系统字体导致 opentype 解析失败（invalid CFF table），内置字体不依赖系统环境；SIL OFL 1.1 可再分发（2026-08-03） |
| 历史存储 | 文件归档 `files/history/<ts>/` + `index.json` | 不引入 Room/KSP（受限网络下新依赖有下载风险）；功能等价（列表/详情/删除）；测试产物自动归档避免被下次测试清空（2026-08-03） |
| 结果图可见性 | 图片 Tab 预览 + MediaStore 保存相册 + 单图分享 | PNG 存在应用私有目录用户不可见；相册 `Pictures/NekoSpeed/` 手动保存避免刷屏（2026-08-03） |
| 启动图标 | 矢量 adaptive icon（品牌红 #D63031） | 零外部资源，后续可替换（2026-07-28） |
| 品牌色 | #D63031 | 取自 render 默认主题 IndexFG 红色（2026-07-28） |
| 订阅拉取兼容 | 增加重试 / HTTP/1.1 降级 / 完整 UA | 部分机场会按 JA3/UA/HTTP 版本风控；仍失败可粘贴内容绕过（2026-07-29） |
| 粘贴内容模式 | Android 端保存为本地 `sub.txt` | 绕过 URL 拉取失败场景（2026-07-29） |

## 6. 注意事项 / 备忘

- **二进制交叉编译**：
  ```bash
  GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build \
    -trimpath -ldflags "-s -w" \
    -o android/app/src/main/assets/nekospeed-android-arm64 \
    ./cmd/nekospeed
  gzip -f android/app/src/main/assets/nekospeed-android-arm64
  ```
  构建脚本 `scripts/build.sh` 已自动完成最后一步 gzip 压缩。运行时 `NekoSpeedRunner.ensureBinary()` 检测到 `.gz` 后缀会 GZIP 解压。
- **订阅拉取**：`internal/subscription/subscription.go` 已增加重试、HTTP/1.1 降级、更完整请求头；仍失败时可切换为“粘贴订阅内容”模式
- **粘贴订阅内容**：Android InputScreen 支持 URL / 粘贴内容 两种来源；粘贴内容会保存为 `files/sub.txt` 后传给 Go 二进制
- **网络权限**：AndroidManifest 需 `INTERNET` 权限
- **前台服务权限**：目标 SDK 34 需同时声明 `FOREGROUND_SERVICE` 与 `FOREGROUND_SERVICE_DATA_SYNC`
- **本地代理冲突**：手机开着 Clash/V2ray 全局 VPN 时测速结果会失真，启动前弹窗提示用户
- **后台执行**：`NekoSpeedService` 前台服务保持测速进程，通知栏提供取消按钮
- **Gradle JDK**：需 Java 17+
- **GPL 合规**：依赖 mihomo（GPL-3.0），分发 APK 需遵守 GPL

## 7. 未来规划

### 短期（已完成）
- [x] 前台服务保活（防止测试被系统杀死）
- [x] 分享功能：分享 JSON / PNG
- [x] VPN 冲突检测与提示
- [x] 二进制压缩存储

### 中期
- [ ] WorkManager 后台定时测试
- [ ] 本地历史记录（Room 数据库）
- [ ] 多订阅管理
- [ ] 节点筛选与搜索
- [ ] 深色模式跟随系统（已实现 System 模式，可细化默认策略）

### 长期
- [ ] 内置 WebSocket 实时测速进度（取代 stdout 轮询）
- [ ] 桌面端（Fyne）同步实现
- [ ] CI/CD：GitHub Actions 自动交叉编译 + 构建 APK
- [ ] F-Droid / GitHub Releases 分发

## 8. 参考

- `docs/android-client.md` — 产品需求与技术方案
- `internal/report/json.go` — JSON 输出结构定义
- `internal/model/model.go` — 核心数据模型
- `scripts/build.sh` — 编译入口（待扩展 Android 目标）
