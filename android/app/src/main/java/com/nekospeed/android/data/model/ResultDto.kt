package com.nekospeed.android.data.model

import kotlinx.serialization.Serializable

@Serializable
data class NekoSpeedResult(
    val version: String,
    val startTime: String,
    val elapsed: String,
    val total: Int,
    val succeeded: Int,
    val nodes: List<NodeResult>,
    val topology: Topology? = null
)

@Serializable
data class NodeResult(
    val index: Int,
    val name: String,
    val type: String,
    val server: String,
    val error: String? = null,
    val latency: LatencyResult? = null,
    val speed: SpeedResult? = null,
    val nat: String? = null,
    val media: List<UnlockResult>? = null,
    val ai: List<UnlockResult>? = null,
    val inbound: GeoInfo? = null,
    val outbound: GeoInfo? = null
) {
    val isOK: Boolean
        get() = error.isNullOrEmpty()
}

@Serializable
data class LatencyResult(
    val tlsRTT: List<Int> = emptyList(),
    val httpsPing: List<Int> = emptyList(),
    val avgTlsRTT: Int = 0,
    val avgHTTPS: Int = 0,
    val packetLoss: Double = 0.0,
    val httpCode: Int = 0
)

@Serializable
data class SpeedResult(
    val perSecond: List<Double> = emptyList(),
    val avg: Double = 0.0,
    val max: Double = 0.0,
    val threads: Int = 0
)

@Serializable
data class UnlockResult(
    val platform: String,
    val status: String,
    val region: String? = null,
    val info: String? = null
) {
    val label: String
        get() = when (status) {
            "unlock" -> if (!info.isNullOrEmpty()) info else if (!region.isNullOrEmpty()) "解锁($region)" else "解锁"
            "pending" -> "待解锁"
            "senttocn" -> if (!region.isNullOrEmpty()) "送中($region)" else "送中"
            "failed" -> if (!region.isNullOrEmpty()) "失败($region)" else "失败"
            "reqfailed" -> "请求失败"
            else -> "N/A"
        }
}

@Serializable
data class GeoInfo(
    val ip: String = "",
    val country: String = "",
    val region: String = "",
    val city: String = "",
    val asn: String = "",
    val org: String = "",
    val isp: String = "",
    val stack: String = ""
)

@Serializable
data class Topology(
    val inbounds: List<InboundEntry> = emptyList(),
    val outbounds: List<OutboundEntry> = emptyList()
)

@Serializable
data class InboundEntry(
    val index: Int = 0,
    val geo: GeoInfo? = null,
    val cluster: Int = 0,
    val outCount: Int = 0
)

@Serializable
data class OutboundEntry(
    val inboundIndex: Int = 0,
    val geo: GeoInfo? = null,
    val cluster: Int = 0,
    val nodeName: String = ""
)
