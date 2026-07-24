package com.nekospeed.android.viewmodel

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.nekospeed.android.data.ResultRepository
import com.nekospeed.android.data.model.HistoryRecord
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

class HistoryViewModel(application: Application) : AndroidViewModel(application) {

    private val repository = ResultRepository()

    private val _records = MutableStateFlow<List<HistoryRecord>>(emptyList())
    val records: StateFlow<List<HistoryRecord>> = _records.asStateFlow()

    fun refresh() {
        viewModelScope.launch {
            _records.value = repository.loadHistory(getApplication())
        }
    }

    fun delete(record: HistoryRecord) {
        viewModelScope.launch {
            repository.deleteHistory(getApplication(), record)
            refresh()
        }
    }
}
