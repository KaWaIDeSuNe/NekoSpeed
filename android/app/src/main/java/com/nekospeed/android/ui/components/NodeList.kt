package com.nekospeed.android.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.nekospeed.android.data.model.NodeResult
import java.util.Locale

@Composable
fun NodeList(
    nodes: List<NodeResult>,
    modifier: Modifier = Modifier
) {
    if (nodes.isEmpty()) {
        EmptyState(text = "无节点数据")
        return
    }

    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        items(nodes, key = { it.index }) { node ->
            NodeCard(node = node)
        }
    }
}

@Composable
private fun NodeCard(node: NodeResult) {
    val statusColor = when {
        !node.isOK -> MaterialTheme.colorScheme.error
        node.speed != null -> Color(0xFF2E7D32) // green
        node.latency != null -> MaterialTheme.colorScheme.primary
        else -> MaterialTheme.colorScheme.outline
    }

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "${node.index}. ${node.name}",
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f)
                )
                Text(
                    text = node.type,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            Text(
                text = node.server,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                node.latency?.let {
                    MetricItem(label = "延迟", value = "${it.avgHTTPS}ms")
                }
                node.speed?.let {
                    MetricItem(label = "均速", value = formatSpeed(it.avg))
                    MetricItem(label = "峰值", value = formatSpeed(it.max))
                }
            }

            if (!node.isOK) {
                Text(
                    text = node.error ?: "节点异常",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }

            StatusChip(
                text = if (node.isOK) "可用" else "失败",
                color = statusColor
            )
        }
    }
}

@Composable
private fun MetricItem(label: String, value: String) {
    Column {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium
        )
    }
}

@Composable
internal fun StatusChip(text: String, color: Color) {
    Card(
        colors = CardDefaults.cardColors(
            containerColor = color.copy(alpha = 0.15f)
        )
    ) {
        Text(
            text = text,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            style = MaterialTheme.typography.labelSmall,
            color = color
        )
    }
}

internal fun formatSpeed(bytesPerSecond: Double): String {
    return when {
        bytesPerSecond >= 1_000_000_000 -> String.format(Locale.US, "%.2f GB/s", bytesPerSecond / 1_000_000_000)
        bytesPerSecond >= 1_000_000 -> String.format(Locale.US, "%.2f MB/s", bytesPerSecond / 1_000_000)
        bytesPerSecond >= 1_000 -> String.format(Locale.US, "%.2f KB/s", bytesPerSecond / 1_000)
        else -> String.format(Locale.US, "%.0f B/s", bytesPerSecond)
    }
}

@Composable
internal fun EmptyState(text: String) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}
