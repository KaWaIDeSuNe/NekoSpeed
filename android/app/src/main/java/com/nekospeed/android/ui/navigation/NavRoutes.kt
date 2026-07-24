package com.nekospeed.android.ui.navigation

sealed class NavRoutes(val route: String) {
    data object Input : NavRoutes("input")
    data object Running : NavRoutes("running")
    data object History : NavRoutes("history")

    data object Result : NavRoutes("result?file={file}") {
        const val ARG_FILE = "file"
        fun create(file: String? = null) = if (file != null) "result?file=$file" else "result"
    }
}
