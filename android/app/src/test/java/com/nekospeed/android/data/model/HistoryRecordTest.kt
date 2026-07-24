package com.nekospeed.android.data.model

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test

class HistoryRecordTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun roundTrip() {
        val record = HistoryRecord(
            id = 1700000000000L,
            startTime = "2026-08-03 15:30:00",
            subName = "紅葉",
            total = 43,
            succeeded = 43,
            elapsed = "12m0s",
            tests = listOf("speed"),
            dir = "/data/data/com.nekospeed.android/files/history/20260803_153000"
        )
        val encoded = json.encodeToString(
            kotlinx.serialization.builtins.ListSerializer(HistoryRecord.serializer()),
            listOf(record)
        )
        val decoded = json.decodeFromString(
            kotlinx.serialization.builtins.ListSerializer(HistoryRecord.serializer()),
            encoded
        )
        assertEquals(1, decoded.size)
        assertEquals(record, decoded.first())
    }
}
