package com.nekospeed.android.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Divider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.nekospeed.android.data.model.InboundEntry
import com.nekospeed.android.data.model.OutboundEntry
import com.nekospeed.android.data.model.Topology

@Composable
fun TopologyTable(
    topology: Topology?,
    modifier: Modifier = Modifier
) {
    val inbounds = topology?.inbounds ?: emptyList()
    val outbounds = topology?.outbounds ?: emptyList()

    if (inbounds.isEmpty() && outbounds.isEmpty()) {
        EmptyState(text = "未检测到拓扑测试数据")
        return
    }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        if (inbounds.isNotEmpty()) {
            Section(title = "入口聚合 (${inbounds.size})") {
                inbounds.forEach { entry ->
                    InboundRow(entry = entry)
                    Divider()
                }
            }
        }

        if (outbounds.isNotEmpty()) {
            Section(title = "出口明细 (${outbounds.size})") {
                outbounds.forEach { entry ->
                    OutboundRow(entry = entry)
                    Divider()
                }
            }
        }
    }
}

@Composable
private fun Section(
    title: String,
    content: @Composable () -> Unit
) {
    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium
        )
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceVariant
            )
        ) {
            Column(modifier = Modifier.fillMaxWidth()) {
                content()
            }
        }
    }
}

@Composable
private fun InboundRow(entry: InboundEntry) {
    val geo = entry.geo
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = "入口 #${entry.index}",
                style = MaterialTheme.typography.titleSmall
            )
            Text(
                text = "Cluster ${entry.cluster} · 出口 ${entry.outCount}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        if (geo != null) {
            Text(
                text = "${geo.ip} · ${geo.country} ${geo.city} · ${geo.isp}",
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            if (geo.asn.isNotBlank()) {
                Text(
                    text = geo.asn,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
private fun OutboundRow(entry: OutboundEntry) {
    val geo = entry.geo
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = "入口 #${entry.inboundIndex} · ${entry.nodeName}",
                style = MaterialTheme.typography.titleSmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f)
            )
            Text(
                text = "C${entry.cluster}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        if (geo != null) {
            Text(
                text = "${geo.ip} · ${geo.country} ${geo.city} · ${geo.isp}",
                style = MaterialTheme.typography.bodySmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            if (geo.asn.isNotBlank()) {
                Text(
                    text = geo.asn,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}
