package com.nekospeed.android.service

import com.nekospeed.android.data.model.NekoSpeedResult
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

sealed class ServiceState {
    data object Idle : ServiceState()
    data object Preparing : ServiceState()
    data object Running : ServiceState()
    data class Done(
        val result: NekoSpeedResult,
        val outDir: java.io.File?
    ) : ServiceState()
    data class Error(val message: String) : ServiceState()
}

object NekoSpeedServiceBus {
    private const val MAX_LOG_LINES = 500

    private val _state = MutableStateFlow<ServiceState>(ServiceState.Idle)
    val state: StateFlow<ServiceState> = _state.asStateFlow()

    private val _logs = MutableStateFlow<List<String>>(emptyList())
    val logs: StateFlow<List<String>> = _logs.asStateFlow()

    fun reset() {
        _state.value = ServiceState.Idle
        _logs.value = emptyList()
    }

    internal fun setState(value: ServiceState) {
        _state.value = value
    }

    internal fun appendLog(line: String) {
        _logs.value = (_logs.value + line).takeLast(MAX_LOG_LINES)
    }
}
