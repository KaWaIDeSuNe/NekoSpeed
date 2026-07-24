package com.nekospeed.android.data.model

import android.content.Context
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import java.io.File

@Serializable
data class HistoryRecord(
    val id: Long,
    val startTime: String,
    val subName: String,
    val total: Int,
    val succeeded: Int,
    val elapsed: String,
    val tests: List<String>,
    val dir: String
)

object HistoryIndex {

    private val json = Json { ignoreUnknownKeys = true }

    suspend fun read(context: Context): List<HistoryRecord> = withContext(Dispatchers.IO) {
        val file = File(context.filesDir, "history/index.json")
        if (!file.exists()) return@withContext emptyList()
        try {
            val list = json.decodeFromString<List<HistoryRecord>>(file.readText())
            list.filter { File(it.dir).exists() }
        } catch (_: Exception) {
            emptyList()
        }
    }

    suspend fun write(context: Context, records: List<HistoryRecord>) = withContext(Dispatchers.IO) {
        val file = File(context.filesDir, "history/index.json")
        file.parentFile?.mkdirs()
        file.writeText(json.encodeToString(ListSerializer(HistoryRecord.serializer()), records))
    }
}
