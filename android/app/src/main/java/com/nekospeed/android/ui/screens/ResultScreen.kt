package com.nekospeed.android.ui.screens

import android.widget.Toast
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.nekospeed.android.data.ResultRepository
import com.nekospeed.android.data.model.NekoSpeedResult
import com.nekospeed.android.ui.components.NodeList
import com.nekospeed.android.ui.components.ReportImagesTab
import com.nekospeed.android.ui.components.TopologyTable
import com.nekospeed.android.ui.components.UnlockGrid
import com.nekospeed.android.util.ShareHelper
import java.io.File

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ResultScreen(
    result: NekoSpeedResult?,
    outDir: File?,
    onBack: () -> Unit,
    loadPath: String? = null
) {
    val context = LocalContext.current
    var selectedTab by rememberSaveable { mutableIntStateOf(0) }

    // 从历史进入时按归档路径加载
    val historyResult by produceState<NekoSpeedResult?>(initialValue = null, loadPath) {
        value = if (loadPath != null) {
            val repo = ResultRepository()
            val r = repo.loadFromFile(File(loadPath))
            (r as? ResultRepository.Result.Success)?.data
        } else {
            result
        }
    }

    val displayResult = historyResult ?: result

    if (displayResult == null) {
        androidx.compose.foundation.layout.Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = androidx.compose.ui.Alignment.Center
        ) {
            androidx.compose.material3.CircularProgressIndicator()
        }
        return
    }

    val displayOutDir = loadPath?.let { File(it).parentFile } ?: outDir

    val tabs = remember(displayResult, displayOutDir) {
        listOf<Pair<String, @Composable () -> Unit>>(
            "节点" to {
                NodeList(nodes = displayResult.nodes, modifier = Modifier.fillMaxSize())
            },
            "解锁" to {
                UnlockGrid(nodes = displayResult.nodes, modifier = Modifier.fillMaxSize())
            },
            "拓扑" to {
                TopologyTable(topology = displayResult.topology, modifier = Modifier.fillMaxSize())
            },
            "图片" to {
                ReportImagesTab(
                    files = displayOutDir?.let { dir ->
                        dir.listFiles { f -> f.name.endsWith(".png") }?.sorted()
                    } ?: emptyList(),
                    modifier = Modifier.fillMaxSize()
                )
            }
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("测试结果") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "返回")
                    }
                },
                actions = {
                    IconButton(
                        onClick = {
                            if (displayOutDir != null) {
                                ShareHelper.shareResult(context, displayOutDir)
                            } else {
                                Toast.makeText(context, "结果目录不存在", Toast.LENGTH_SHORT).show()
                            }
                        }
                    ) {
                        Icon(Icons.Default.Share, contentDescription = "分享")
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            Text(
                text = "共 ${displayResult.total} 个节点，成功 ${displayResult.succeeded} 个 · 耗时 ${displayResult.elapsed}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
            )

            TabRow(selectedTabIndex = selectedTab) {
                tabs.forEachIndexed { index, (title, _) ->
                    Tab(
                        selected = selectedTab == index,
                        onClick = { selectedTab = index },
                        text = { Text(title) }
                    )
                }
            }

            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 16.dp, vertical = 8.dp)
            ) {
                tabs[selectedTab].second()
            }
        }
    }
}
