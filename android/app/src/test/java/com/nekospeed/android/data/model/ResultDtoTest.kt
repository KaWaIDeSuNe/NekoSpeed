package com.nekospeed.android.data.model

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ResultDtoTest {

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    private val sampleJson = """
    {
      "version": "0.1.0",
      "startTime": "2026-07-28T12:00:00Z",
      "elapsed": "1m23s",
      "total": 2,
      "succeeded": 1,
      "nodes": [
        {
          "index": 1,
          "name": "HK Node",
          "type": "Hysteria2",
          "server": "hk.example.com",
          "latency": {
            "tlsRTT": [120, 130],
            "httpsPing": [150, 160],
            "avgTlsRTT": 125,
            "avgHTTPS": 155,
            "packetLoss": 0.0,
            "httpCode": 200
          },
          "speed": {
            "perSecond": [1000000.0, 2000000.0],
            "avg": 1500000.0,
            "max": 2000000.0,
            "threads": 4
          },
          "media": [
            {"platform": "youtube", "status": "unlock", "region": "JP"}
          ],
          "ai": [
            {"platform": "openai", "status": "unlock", "region": "US"}
          ],
          "inbound": {
            "ip": "1.2.3.4",
            "country": "HK",
            "region": "",
            "city": "Hong Kong",
            "asn": "AS12345",
            "org": "Example",
            "isp": "Example ISP",
            "stack": "4"
          },
          "outbound": {
            "ip": "5.6.7.8",
            "country": "JP",
            "region": "",
            "city": "Tokyo",
            "asn": "AS67890",
            "org": "JP Example",
            "isp": "JP ISP",
            "stack": "4"
          }
        },
        {
          "index": 2,
          "name": "Failed Node",
          "type": "Vless",
          "server": "failed.example.com",
          "error": "timeout"
        }
      ],
      "topology": {
        "inbounds": [
          {"index": 1, "geo": {"ip": "1.2.3.4", "country": "HK", "city": "Hong Kong", "asn": "AS12345", "isp": "Example ISP", "stack": "4"}, "cluster": 1, "outCount": 1}
        ],
        "outbounds": [
          {"inboundIndex": 1, "geo": {"ip": "5.6.7.8", "country": "JP", "city": "Tokyo", "asn": "AS67890", "isp": "JP ISP", "stack": "4"}, "cluster": 1, "nodeName": "HK Node"}
        ]
      }
    }
    """.trimIndent()

    @Test
    fun `parse sample result json`() {
        val result = json.decodeFromString<NekoSpeedResult>(sampleJson)

        assertEquals("0.1.0", result.version)
        assertEquals(2, result.total)
        assertEquals(1, result.succeeded)
        assertEquals(2, result.nodes.size)

        val first = result.nodes[0]
        assertEquals("HK Node", first.name)
        assertTrue(first.isOK)
        assertNotNull(first.latency)
        assertEquals(155, first.latency?.avgHTTPS)
        assertNotNull(first.speed)
        assertEquals(1500000.0, first.speed?.avg ?: 0.0, 0.001)
        assertEquals(1, first.media?.size)
        assertEquals("youtube", first.media?.firstOrNull()?.platform)
        assertEquals("unlock", first.media?.firstOrNull()?.status)

        val second = result.nodes[1]
        assertEquals("timeout", second.error)
        assertTrue(!second.isOK)

        assertNotNull(result.topology)
        assertEquals(1, result.topology?.inbounds?.size)
        assertEquals(1, result.topology?.outbounds?.size)
        assertEquals("HK", result.topology?.inbounds?.firstOrNull()?.geo?.country)
    }
}
