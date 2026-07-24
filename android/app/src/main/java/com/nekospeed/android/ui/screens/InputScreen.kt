package com.nekospeed.android.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.DarkMode
import androidx.compose.material.icons.filled.History
import androidx.compose.material.icons.filled.LightMode
import androidx.compose.material.icons.filled.Palette
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Start
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.nekospeed.android.ui.theme.ThemeMode
import com.nekospeed.android.util.VpnDetector
import com.nekospeed.android.viewmodel.RunViewModel
import com.nekospeed.android.viewmodel.SubInputMode
import com.nekospeed.android.viewmodel.TestType

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun InputScreen(
    viewModel: RunViewModel,
    themeMode: ThemeMode,
    onToggleTheme: () -> Unit,
    onStart: () -> Unit,
    onOpenHistory: () -> Unit
) {
    val context = LocalContext.current
    val input by viewModel.input.collectAsState()
    var advancedExpanded by rememberSaveable { mutableStateOf(false) }
    var sourceError by rememberSaveable { mutableStateOf(false) }
    var showVpnWarning by rememberSaveable { mutableStateOf(false) }

    val sourceBlank = when (input.mode) {
        SubInputMode.Url -> input.sub.isBlank()
        SubInputMode.Text -> input.content.isBlank()
    }

    val startTest = {
        if (sourceBlank) {
            sourceError = true
        } else {
            sourceError = false
            if (input.tests.isEmpty()) {
                viewModel.updateInput { copy(tests = TestType.ALL) }
            }
            if (VpnDetector.isVpnActive(context)) {
                showVpnWarning = true
            } else {
                onStart()
            }
        }
    }

    if (showVpnWarning) {
        AlertDialog(
            onDismissRequest = { showVpnWarning = false },
            title = { Text("VPN / 代理冲突提示") },
            text = { Text(VpnDetector.getWarningMessage()) },
            confirmButton = {
                TextButton(
                    onClick = {
                        showVpnWarning = false
                        onStart()
                    }
                ) {
                    Text("仍要运行")
                }
            },
            dismissButton = {
                TextButton(onClick = { showVpnWarning = false }) {
                    Text("取消")
                }
            }
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("NekoSpeed") },
                actions = {
                    IconButton(onClick = onOpenHistory) {
                        Icon(Icons.Default.History, contentDescription = "历史记录")
                    }
                    IconButton(onClick = onToggleTheme) {
                        val icon = when (themeMode) {
                            ThemeMode.System -> Icons.Default.Palette
                            ThemeMode.Light -> Icons.Default.LightMode
                            ThemeMode.Dark -> Icons.Default.DarkMode
                            ThemeMode.NekoSpeed -> Icons.Default.Palette
                        }
                        Icon(icon, contentDescription = "切换主题")
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(padding)
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = "订阅来源",
                style = MaterialTheme.typography.titleMedium
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                InputModeChip(
                    selected = input.mode == SubInputMode.Url,
                    onClick = { viewModel.updateInput { copy(mode = SubInputMode.Url) } },
                    label = "URL",
                    modifier = Modifier.weight(1f)
                )
                InputModeChip(
                    selected = input.mode == SubInputMode.Text,
                    onClick = { viewModel.updateInput { copy(mode = SubInputMode.Text) } },
                    label = "粘贴内容",
                    modifier = Modifier.weight(1f)
                )
            }

            if (input.mode == SubInputMode.Url) {
                OutlinedTextField(
                    value = input.sub,
                    onValueChange = {
                        sourceError = false
                        viewModel.updateInput { copy(sub = it) }
                    },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text("订阅 URL") },
                    placeholder = { Text("https://example.com/subscribe") },
                    isError = sourceError,
                    supportingText = {
                        if (sourceError) {
                            Text("请输入订阅链接", color = MaterialTheme.colorScheme.error)
                        } else {
                            Text("支持 base64 / 原始 clash 订阅链接")
                        }
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Uri,
                        imeAction = ImeAction.Next
                    ),
                    maxLines = 5,
                    minLines = 2
                )
            } else {
                OutlinedTextField(
                    value = input.content,
                    onValueChange = {
                        sourceError = false
                        viewModel.updateInput { copy(content = it) }
                    },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text("订阅内容") },
                    placeholder = { Text("粘贴 clash 配置 或 base64 分享链接...") },
                    isError = sourceError,
                    supportingText = {
                        if (sourceError) {
                            Text("请粘贴订阅内容", color = MaterialTheme.colorScheme.error)
                        } else {
                            Text("从 Clash / 浏览器复制完整订阅文本后粘贴")
                        }
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Text,
                        imeAction = ImeAction.Default
                    ),
                    maxLines = 12,
                    minLines = 6
                )
            }

            Text(
                text = "选择测试项",
                style = MaterialTheme.typography.titleMedium
            )

            FlowRow(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                TestType.ALL.forEach { test ->
                    val selected = test in input.tests
                    FilterChip(
                        selected = selected,
                        onClick = {
                            viewModel.updateInput {
                                val updated = if (selected) tests - test else tests + test
                                copy(tests = updated.distinct())
                            }
                        },
                        label = { Text(TestType.labels[test] ?: test) }
                    )
                }
            }

            Card(
                onClick = { advancedExpanded = !advancedExpanded },
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(Icons.Default.Settings, contentDescription = null)
                        Text(
                            text = "高级参数",
                            style = MaterialTheme.typography.titleMedium
                        )
                    }
                    Text(
                        text = if (advancedExpanded) "收起" else "展开",
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.primary
                    )
                }

                if (advancedExpanded) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp)
                            .padding(bottom = 16.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            NumericField(
                                label = "线程数",
                                value = input.threads,
                                onValueChange = { viewModel.updateInput { copy(threads = it.coerceAtLeast(1)) } },
                                modifier = Modifier.weight(1f)
                            )
                            NumericField(
                                label = "时长(秒)",
                                value = input.duration,
                                onValueChange = { viewModel.updateInput { copy(duration = it.coerceAtLeast(1)) } },
                                modifier = Modifier.weight(1f)
                            )
                        }

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            NumericField(
                                label = "并发节点",
                                value = input.concurrency,
                                onValueChange = { viewModel.updateInput { copy(concurrency = it.coerceAtLeast(1)) } },
                                modifier = Modifier.weight(1f)
                            )
                            NumericField(
                                label = "Ping 次数",
                                value = input.pingCount,
                                onValueChange = { viewModel.updateInput { copy(pingCount = it.coerceAtLeast(1)) } },
                                modifier = Modifier.weight(1f)
                            )
                        }

                        OutlinedTextField(
                            value = input.filter,
                            onValueChange = { viewModel.updateInput { copy(filter = it) } },
                            modifier = Modifier.fillMaxWidth(),
                            label = { Text("节点过滤（正则，可选）") },
                            placeholder = { Text("HK|Japan") },
                            singleLine = true
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            Button(
                onClick = startTest,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp)
            ) {
                Icon(Icons.Default.Start, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("开始测试")
            }

            Spacer(modifier = Modifier.height(16.dp))
        }
    }
}

@Composable
private fun InputModeChip(
    selected: Boolean,
    onClick: () -> Unit,
    label: String,
    modifier: Modifier = Modifier
) {
    FilterChip(
        selected = selected,
        onClick = onClick,
        label = { Text(label) },
        modifier = modifier
    )
}

@Composable
private fun NumericField(
    label: String,
    value: Int,
    onValueChange: (Int) -> Unit,
    modifier: Modifier = Modifier
) {
    OutlinedTextField(
        value = value.toString(),
        onValueChange = {
            val number = it.toIntOrNull() ?: 1
            onValueChange(number)
        },
        modifier = modifier,
        label = { Text(label) },
        keyboardOptions = KeyboardOptions(
            keyboardType = KeyboardType.Number,
            imeAction = ImeAction.Next
        ),
        singleLine = true
    )
}
