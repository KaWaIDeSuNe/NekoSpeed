package com.nekospeed.android.data

import android.content.Context
import android.os.Build
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.BufferedReader
import java.io.File
import java.io.IOException
import java.io.InputStreamReader
import java.util.zip.GZIPInputStream

class NekoSpeedRunner(private val context: Context) {

    private val binary: File
        get() = File(context.filesDir, "nekospeed")

    private val font: File
        get() = File(context.filesDir, "NotoSansCJKsc-Regular.otf")

    private val fontAsset = "fonts/NotoSansCJKsc-Regular.otf"

    private val compressedAssets: List<String>
        get() = if (Build.SUPPORTED_ABIS.any { it.startsWith("arm64") }) {
            listOf("nekospeed-android-arm64.gz", "nekospeed-android-arm64")
        } else {
            listOf("nekospeed-android-amd64.gz", "nekospeed-android-amd64")
        }

    @Volatile
    private var currentProcess: Process? = null

    suspend fun ensureBinary() = withContext(Dispatchers.IO) {
        if (binary.exists()) return@withContext

        val assetName = compressedAssets.firstOrNull { assetExists(context, it) }
            ?: throw IllegalStateException("未找到 assets 中的二进制文件")

        context.assets.open(assetName).use { input ->
            val stream = if (assetName.endsWith(".gz")) GZIPInputStream(input) else input
            binary.outputStream().use { output ->
                stream.copyTo(output)
            }
        }
        binary.setExecutable(true)
    }

    /** 解压内置中文字体（厂商 ROM 系统字体解析失败时经 -font 兜底） */
    suspend fun ensureFont() = withContext(Dispatchers.IO) {
        if (font.exists()) return@withContext
        if (!assetExists(context, fontAsset)) return@withContext
        context.assets.open(fontAsset).use { input ->
            font.outputStream().use { output ->
                input.copyTo(output)
            }
        }
    }

    private fun assetExists(context: Context, name: String): Boolean {
        return try {
            context.assets.open(name).close()
            true
        } catch (_: Exception) {
            false
        }
    }

    suspend fun run(
        sub: String,
        tests: List<String>,
        threads: Int = 8,
        duration: Int = 10,
        concurrency: Int = 1,
        pingCount: Int = 4,
        filter: String? = null,
        onLog: (String) -> Unit = {}
    ): Result = withContext(Dispatchers.IO) {
        val outDir = File(context.filesDir, "out")
        outDir.deleteRecursively()
        outDir.mkdirs()

        val args = mutableListOf(
            binary.absolutePath,
            "-sub", sub,
            "-tests", tests.joinToString(","),
            "-o", outDir.absolutePath,
            "-threads", threads.toString(),
            "-duration", duration.toString(),
            "-concurrency", concurrency.toString(),
            "-ping-count", pingCount.toString()
        )
        if (font.exists()) {
            args.addAll(listOf("-font", font.absolutePath))
        }
        if (!filter.isNullOrBlank()) {
            args.addAll(listOf("-filter", filter))
        }

        val process = try {
            startProcess(outDir, args)
        } catch (e: IOException) {
            startProcessViaLinker(outDir, args)
        }
        currentProcess = process

        try {
            val reader = BufferedReader(InputStreamReader(process.inputStream))
            var line: String?
            while (reader.readLine().also { line = it } != null) {
                onLog(line!!)
            }
            val exitCode = process.waitFor()
            val resultFile = outDir.listFiles { f ->
                f.name.endsWith("_result.json")
            }?.maxByOrNull { it.lastModified() }

            if (exitCode != 0 || resultFile == null) {
                Result.Error("进程退出码=$exitCode，结果文件不存在")
            } else {
                Result.Success(resultFile)
            }
        } catch (e: CancellationException) {
            process.destroyForcibly()
            throw e
        } catch (e: Exception) {
            process.destroyForcibly()
            Result.Error(e.message ?: "未知错误")
        } finally {
            currentProcess = null
        }
    }

    private fun startProcess(outDir: File, args: List<String>): Process =
        ProcessBuilder(args)
            .redirectErrorStream(true)
            .directory(outDir)
            .start()

    /** Android 14+ SELinux 拒绝直接 execve app_data_file，回退经动态链接器执行 */
    private fun startProcessViaLinker(outDir: File, args: List<String>): Process =
        ProcessBuilder(listOf("/system/bin/linker64") + args)
            .redirectErrorStream(true)
            .directory(outDir)
            .start()

    fun cancel() {
        currentProcess?.destroyForcibly()
    }

    fun getOutDir(): File = File(context.filesDir, "out")

    fun getPngFiles(): List<File> = getOutDir().listFiles { f ->
        f.name.endsWith(".png")
    }?.sorted() ?: emptyList()

    sealed class Result {
        data class Success(val file: File) : Result()
        data class Error(val message: String) : Result()
    }
}
