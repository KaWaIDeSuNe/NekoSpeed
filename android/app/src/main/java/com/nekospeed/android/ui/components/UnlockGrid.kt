package com.nekospeed.android.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.nekospeed.android.data.model.NodeResult
import com.nekospeed.android.data.model.UnlockResult

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun UnlockGrid(
    nodes: List<NodeResult>,
    modifier: Modifier = Modifier
) {
    val allPlatforms = remember(nodes) {
        val platforms = linkedSetOf<String>()
        nodes.forEach { node ->
            node.media?.forEach { platforms.add(it.platform) }
            node.ai?.forEach { platforms.add(it.platform) }
        }
        platforms.toList()
    }

    if (allPlatforms.isEmpty()) {
        EmptyState(text = "未检测到解锁测试数据")
        return
    }

    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        items(nodes, key = { it.index }) { node ->
            UnlockNodeCard(
                node = node,
                allPlatforms = allPlatforms
            )
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun UnlockNodeCard(
    node: NodeResult,
    allPlatforms: List<String>
) {
    val unlockMap = remember(node.media, node.ai) {
        buildMap {
            node.media?.forEach { put(it.platform, it) }
            node.ai?.forEach { put(it.platform, it) }
        }
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
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = "${node.index}. ${node.name}",
                style = MaterialTheme.typography.titleSmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            FlowRow(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp)
            ) {
                allPlatforms.forEach { platform ->
                    val unlock = unlockMap[platform]
                    UnlockChip(platform = platform, result = unlock)
                }
            }
        }
    }
}

@Composable
private fun UnlockChip(platform: String, result: UnlockResult?) {
    val (text, color) = when (result?.status) {
        "unlock" -> {
            val label = result.region?.let { "${result.platform}($it)" } ?: result.platform
            label to Color(0xFF2E7D32)
        }
        "pending" -> (result.platform to Color(0xFFF9A825))
        "senttocn" -> (result.platform to Color(0xFFEF6C00))
        "failed" -> (result.platform to Color(0xFFD32F2F))
        "reqfailed" -> (result.platform to Color(0xFFD32F2F))
        else -> (platform to MaterialTheme.colorScheme.outline)
    }

    Card(
        colors = CardDefaults.cardColors(
            containerColor = color.copy(alpha = 0.12f)
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
