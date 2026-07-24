package com.nekospeed.android.data

import android.content.Context
import com.nekospeed.android.data.model.HistoryIndex
import com.nekospeed.android.data.model.HistoryRecord
import com.nekospeed.android.data.model.NekoSpeedResult
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.io.File

class ResultRepository {

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    suspend fun loadFromFile(file: File): Result<NekoSpeedResult> = withContext(Dispatchers.IO) {
        try {
            val text = file.readText()
            val result = json.decodeFromString<NekoSpeedResult>(text)
            Result.Success(result)
        } catch (e: Exception) {
            Result.Error("解析 JSON 失败: ${e.message}")
        }
    }

    /** 把测试产物归档到 history/<ts>/ 并更新索引，返回归档记录 */
    suspend fun archiveResult(
        context: Context,
        outDir: File,
        result: NekoSpeedResult
    ): HistoryRecord? = withContext(Dispatchers.IO) {
        if (!outDir.exists()) return@withContext null
        val ts = java.text.SimpleDateFormat("yyyyMMdd_HHmmss", java.util.Locale.US)
            .format(java.util.Date())
        val dir = File(context.filesDir, "history/$ts")
        if (!dir.exists() && !dir.mkdirs()) return@withContext null

        outDir.listFiles { f ->
            f.name.endsWith("_result.json") || f.name.endsWith(".png")
        }?.forEach { f ->
            runCatching { f.copyTo(File(dir, f.name), overwrite = true) }
        }

        val subName = outDir.listFiles { f -> f.name.endsWith("_result.json") }
            ?.firstOrNull()
            ?.name
            ?.substringBefore("_")
            ?: "未命名"

        val record = HistoryRecord(
            id = System.currentTimeMillis(),
            startTime = result.startTime.ifBlank { ts },
            subName = subName,
            total = result.total,
            succeeded = result.succeeded,
            elapsed = result.elapsed,
            tests = emptyList(),
            dir = dir.absolutePath
        )
        val records = HistoryIndex.read(context) + record
        HistoryIndex.write(context, records)
        record
    }

    suspend fun loadHistory(context: Context): List<HistoryRecord> =
        HistoryIndex.read(context).sortedByDescending { it.id }

    suspend fun deleteHistory(context: Context, record: HistoryRecord) = withContext(Dispatchers.IO) {
        File(record.dir).deleteRecursively()
        val records = HistoryIndex.read(context).filterNot { it.id == record.id }
        HistoryIndex.write(context, records)
    }

    sealed class Result<out T> {
        data class Success<T>(val data: T) : Result<T>()
        data class Error(val message: String) : Result<Nothing>()
    }
}
