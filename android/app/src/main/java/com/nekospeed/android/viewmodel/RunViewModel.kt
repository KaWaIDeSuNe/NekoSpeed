package com.nekospeed.android.viewmodel

import android.app.Application
import android.content.Intent
import androidx.core.content.ContextCompat
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.nekospeed.android.data.model.NekoSpeedResult
import com.nekospeed.android.service.NekoSpeedService
import com.nekospeed.android.service.NekoSpeedServiceBus
import com.nekospeed.android.service.ServiceState
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.io.File

private const val DEFAULT_THREADS = 8
private const val DEFAULT_DURATION = 10
private const val DEFAULT_CONCURRENCY = 1
private const val DEFAULT_PING_COUNT = 4

enum class SubInputMode {
    Url,
    Text
}

data class TestInput(
    val mode: SubInputMode = SubInputMode.Url,
    val sub: String = "",
    val content: String = "",
    val tests: List<String> = TestType.ALL,
    val threads: Int = DEFAULT_THREADS,
    val duration: Int = DEFAULT_DURATION,
    val concurrency: Int = DEFAULT_CONCURRENCY,
    val pingCount: Int = DEFAULT_PING_COUNT,
    val filter: String = ""
) {
    val isValid: Boolean
        get() = when (mode) {
            SubInputMode.Url -> sub.isNotBlank() && tests.isNotEmpty()
            SubInputMode.Text -> content.isNotBlank() && tests.isNotEmpty()
        }
}

object TestType {
    const val SPEED = "speed"
    const val MEDIA = "media"
    const val AI = "ai"
    const val TOPO = "topo"
    const val NAT = "nat"
    val ALL = listOf(SPEED, MEDIA, AI, TOPO, NAT)

    val labels = mapOf(
        SPEED to "测速",
        MEDIA to "流媒体",
        AI to "AI",
        TOPO to "拓扑",
        NAT to "NAT"
    )
}

class RunViewModel(application: Application) : AndroidViewModel(application) {

    private val _input = MutableStateFlow(TestInput())
    val input: StateFlow<TestInput> = _input.asStateFlow()

    private val _state = MutableStateFlow<UiState>(UiState.Idle)
    val state: StateFlow<UiState> = _state.asStateFlow()

    private val _logs = MutableStateFlow<List<String>>(emptyList())
    val logs: StateFlow<List<String>> = _logs.asStateFlow()

    private var syncJob: Job? = null

    init {
        syncServiceBus()
    }

    private fun syncServiceBus() {
        syncJob = viewModelScope.launch {
            launch {
                NekoSpeedServiceBus.state.collect { serviceState ->
                    _state.value = when (serviceState) {
                        is ServiceState.Idle -> UiState.Idle
                        is ServiceState.Preparing -> UiState.Preparing
                        is ServiceState.Running -> UiState.Running
                        is ServiceState.Done -> UiState.Done(
                            result = serviceState.result,
                            resultFile = serviceState.outDir
                        )
                        is ServiceState.Error -> UiState.Error(serviceState.message)
                    }
                }
            }
            launch {
                NekoSpeedServiceBus.logs.collect { lines ->
                    _logs.value = lines
                }
            }
        }
    }

    fun updateInput(transform: TestInput.() -> TestInput) {
        _input.value = _input.value.transform()
    }

    fun setInput(value: TestInput) {
        _input.value = value
    }

    fun startTest() {
        val current = _input.value
        if (!current.isValid) {
            _state.value = UiState.Error(
                when (current.mode) {
                    SubInputMode.Url -> "请输入订阅链接并至少选择一项测试"
                    SubInputMode.Text -> "请粘贴订阅内容并至少选择一项测试"
                }
            )
            return
        }

        val subSource = when (current.mode) {
            SubInputMode.Url -> saveSourceToFile(current.sub.trim())
            SubInputMode.Text -> saveSourceToFile(current.content.trim())
        }

        startTest(
            sub = subSource,
            tests = current.tests,
            threads = current.threads,
            duration = current.duration,
            concurrency = current.concurrency,
            pingCount = current.pingCount,
            filter = current.filter.takeIf { it.isNotBlank() }
        )
    }

    /** 订阅来源落盘后传文件路径给二进制，避免 URL（含 token）暴露在进程命令行 */
    private fun saveSourceToFile(content: String): String {
        val file = File(getApplication<Application>().filesDir, "sub.txt")
        file.writeText(content)
        return file.absolutePath
    }

    fun startTest(
        sub: String,
        tests: List<String>,
        threads: Int,
        duration: Int,
        concurrency: Int,
        pingCount: Int,
        filter: String?
    ) {
        NekoSpeedServiceBus.reset()

        val intent = NekoSpeedService.createStartIntent(
            context = getApplication(),
            sub = sub,
            tests = tests,
            threads = threads,
            duration = duration,
            concurrency = concurrency,
            pingCount = pingCount,
            filter = filter
        )
        ContextCompat.startForegroundService(getApplication(), intent)
    }

    fun cancelTest() {
        val app = getApplication<Application>()
        val intent = Intent(app, NekoSpeedService::class.java).apply {
            action = NekoSpeedService.ACTION_CANCEL
        }
        app.startService(intent)
    }

    fun reset() {
        NekoSpeedServiceBus.reset()
    }

    override fun onCleared() {
        super.onCleared()
        syncJob?.cancel()
    }

    sealed class UiState {
        data object Idle : UiState()
        data object Preparing : UiState()
        data object Running : UiState()
        data class Done(
            val result: NekoSpeedResult,
            val resultFile: File?
        ) : UiState()
        data class Error(val message: String) : UiState()
    }
}
