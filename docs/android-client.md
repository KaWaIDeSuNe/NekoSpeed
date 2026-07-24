# NekoSpeed Android 客户端开发文档

## 1. 目标

为 NekoSpeed 打造一个 Android 原生 GUI，核心功能：

- 输入/导入机场订阅
- 选择测试项（speed / media / ai / topo / nat）
- 执行测速并在手机上展示结果
- 支持导出/分享 JSON 与 PNG
- 主题切换（light / dark / NekoSpeed 主题色）

## 2. 架构

```
Android App (Kotlin + Jetpack Compose)
        │
        ├── assets/nekospeed-android-arm64
        │     └─ 运行时释放到 files/nekospeed
        │
        ├──  ProcessBuilder 调用 NekoSpeed 二进制
        │     -sub <url> -tests <...> -o files/out -json
        │
        └──  解析 result.json 渲染原生 UI
```

## 3. 技术栈

- **UI**：Jetpack Compose + Material3
- **架构**：MVVM + ViewModel + StateFlow
- **并发**：Kotlin Coroutines
- **进程调用**：Runtime.exec / ProcessBuilder
- **JSON 解析**：Kotlinx Serialization
- **图片显示**：Coil（展示 NekoSpeed 生成的 PNG 结果）

## 4. 为什么选嵌入二进制方案

NekoSpeed 已可成功交叉编译：

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o nekospeed-android-arm64 ./cmd/nekospeed
```

- 产物约 23 MB，可再压缩
- 无需重写测速/解锁/拓扑逻辑
- 和桌面版保持同步：更新 Go 代码后重新编译替换即可

## 5. 项目结构

```
NekoSpeed/
├── android/                          # Android 客户端
│   ├── app/
│   │   ├── src/main/
│   │   │   ├── assets/
│   │   │   │   └── nekospeed-android-arm64
│   │   │   ├── java/com/example/nekospeed/
│   │   │   │   ├── MainActivity.kt
│   │   │   │   ├── NekoSpeedApp.kt
│   │   │   │   ├── data/
│   │   │   │   │   ├── NekoSpeedRunner.kt      # 二进制调用
│   │   │   │   │   ├── ResultRepository.kt
│   │   │   │   │   └── model/
│   │   │   │   │       └── ResultDto.kt
│   │   │   │   ├── ui/
│   │   │   │   │   ├── screens/
│   │   │   │   │   │   ├── InputScreen.kt
│   │   │   │   │   │   ├── RunningScreen.kt
│   │   │   │   │   │   └── ResultScreen.kt
│   │   │   │   │   └── components/
│   │   │   │   │       ├── NodeList.kt
│   │   │   │   │       ├── SpeedBar.kt
│   │   │   │   │       └── UnlockGrid.kt
│   │   │   │   └── viewmodel/
│   │   │   │       └── RunViewModel.kt
│   │   └── build.gradle.kts
│   └── build.gradle.kts
├── scripts/
│   └── build.sh                       # 追加 android arm64 目标
└── ...
```

## 6. 关键模块

### 6.1 二进制释放与执行

```kotlin
class NekoSpeedRunner(private val context: Context) {
    private val binary: File
        get() = File(context.filesDir, "nekospeed")

    suspend fun ensureBinary() = withContext(Dispatchers.IO) {
        if (!binary.exists()) {
            context.assets.open("nekospeed-android-arm64").use { input ->
                binary.outputStream().use { output -> input.copyTo(output) }
            }
            binary.setExecutable(true)
        }
    }

    suspend fun run(
        sub: String,
        tests: List<String>,
        concurrency: Int = 1
    ): File = withContext(Dispatchers.IO) {
        val outDir = File(context.filesDir, "out").apply { mkdirs() }
        val process = ProcessBuilder(
            binary.absolutePath,
            "-sub", sub,
            "-tests", tests.joinToString(","),
            "-o", outDir.absolutePath,
            "-concurrency", concurrency.toString()
        ).redirectErrorStream(true).start()
        process.waitFor()
        File(outDir, "result.json")
    }
}
```

### 6.2 数据模型

复用 NekoSpeed `result.json` 结构：

```kotlin
@Serializable
data class Result(
    val version: String,
    val startTime: String,
    val elapsed: String,
    val total: Int,
    val succeeded: Int,
    val nodes: List<Node>
)

@Serializable
data class Node(
    val index: Int,
    val name: String,
    val type: String,
    val server: String,
    val latency: Latency,
    val media: List<Unlock>? = null,
    val ai: List<Unlock>? = null,
    val inbound: Geo? = null,
    val outbound: Geo? = null
)
```

### 6.3 UI 页面

| 页面 | 功能 |
|---|---|
| 输入页 | 粘贴订阅 URL / 选择测试项 / 开始 / 主题切换 |
| 运行页 | 显示进度、日志输出、取消按钮 |
| 结果页 | Tab：节点列表 / 速度 / 解锁 / 拓扑 |
| 节点列表 | 名称、延迟、平均速度、最高速度 |
| 解锁矩阵 | 媒体/AI 平台 × 节点，色块表示成功/失败 |
| 拓扑 | 入口/出口 IP、国家、ASN |

## 7. 功能范围

完整版包含：

- 全部测试：speed / media / ai / topo / nat
- 解锁矩阵（媒体 + AI 平台色块）
- 拓扑图（入口/出口聚合表）
- 主题切换（light / dark / NekoSpeed 主题色）
- 订阅输入、测试参数设置、结果分享

## 8. 构建流程

1. 在 NekoSpeed 项目根目录执行：

```bash
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build \
  -trimpath -ldflags "-s -w" \
  -o android/app/src/main/assets/nekospeed-android-arm64 \
  ./cmd/nekospeed
```

2. 在 Android Studio 打开 `android/` 目录，编译 APK。

3. 建议把步骤 1 加入 `scripts/build.sh` 和 CI。

## 9. 注意事项

- **网络权限**：AndroidManifest 加 `INTERNET`
- **本地代理冲突**：若手机开着 Clash/V2ray 全局 VPN，测速结果可能失真；建议提示用户关闭
- **后台执行**：测试可能耗时 10 分钟，用 `WorkManager` 或前台服务避免被杀
- **权限**：Android 10+ 写文件用私有目录，无需额外权限
- **GPL 合规**：依赖 mihomo，分发 APK 需遵守 GPL-3.0，源码可访问
- **字体与渲染**：PNG 由 NekoSpeed 生成，手机上显示即可；UI 用原生 Compose 避免字体问题

## 10. 实现步骤

1. [x] 修改 `scripts/build.sh` 加 `android arm64` 目标
2. [x] 创建 `android/` 项目骨架
3. [x] 实现 `NekoSpeedRunner` 二进制调用
4. [x] 实现 `ResultRepository` JSON 解析
5. [x] 实现 Compose UI：输入页 / 运行页 / 结果页
6. [x] 实现解锁矩阵与拓扑表
7. [x] 实现主题切换（System / Light / Dark / NekoSpeed）
8. [x] 处理后台运行、VPN 冲突提示、分享 JSON/PNG（Session 3）
9. [x] 订阅拉取兼容：重试 / HTTP/1.1 降级 / 完整 UA
10. [x] 粘贴订阅内容模式（绕过 URL 拉取失败）

---

> 仅供学习研究使用，请勿用于违反当地法律法规的场景。
