package com.nekospeed.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.nekospeed.android.ui.navigation.NavRoutes
import com.nekospeed.android.ui.screens.HistoryScreen
import com.nekospeed.android.ui.screens.InputScreen
import com.nekospeed.android.ui.screens.ResultScreen
import com.nekospeed.android.ui.screens.RunningScreen
import com.nekospeed.android.ui.theme.NekoSpeedTheme
import com.nekospeed.android.ui.theme.ThemeMode
import com.nekospeed.android.viewmodel.RunViewModel

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            var themeMode by rememberSaveable { mutableStateOf(ThemeMode.System) }
            val toggleTheme: () -> Unit = {
                themeMode = when (themeMode) {
                    ThemeMode.System -> ThemeMode.NekoSpeed
                    ThemeMode.Light -> ThemeMode.Dark
                    ThemeMode.Dark -> ThemeMode.Light
                    ThemeMode.NekoSpeed -> ThemeMode.System
                }
            }

            NekoSpeedTheme(themeMode = themeMode) {
                val navController = rememberNavController()
                val viewModel: RunViewModel = viewModel()

                NavHost(
                    navController = navController,
                    startDestination = NavRoutes.Input.route
                ) {
                    composable(NavRoutes.Input.route) {
                        InputScreen(
                            viewModel = viewModel,
                            themeMode = themeMode,
                            onToggleTheme = toggleTheme,
                            onStart = {
                                viewModel.startTest()
                                navController.navigate(NavRoutes.Running.route)
                            },
                            onOpenHistory = {
                                navController.navigate(NavRoutes.History.route)
                            }
                        )
                    }

                    composable(NavRoutes.Running.route) {
                        RunningScreen(
                            viewModel = viewModel,
                            onCancel = {
                                viewModel.cancelTest()
                                navController.popBackStack(
                                    NavRoutes.Input.route,
                                    inclusive = false
                                )
                            },
                            onDone = {
                                navController.navigate(NavRoutes.Result.create()) {
                                    popUpTo(NavRoutes.Running.route) { inclusive = true }
                                }
                            }
                        )
                    }

                    composable(
                        route = NavRoutes.Result.route,
                        arguments = listOf(
                            navArgument(NavRoutes.Result.ARG_FILE) {
                                type = NavType.StringType
                                nullable = true
                                defaultValue = null
                            }
                        )
                    ) { backStackEntry ->
                        val loadPath = backStackEntry.arguments
                            ?.getString(NavRoutes.Result.ARG_FILE)
                        val state by viewModel.state.collectAsState()
                        val done = state as? RunViewModel.UiState.Done

                        if (loadPath != null) {
                            ResultScreen(
                                result = null,
                                outDir = null,
                                loadPath = loadPath,
                                onBack = { navController.popBackStack() }
                            )
                        } else if (done != null) {
                            ResultScreen(
                                result = done.result,
                                outDir = done.resultFile,
                                onBack = {
                                    navController.popBackStack(
                                        NavRoutes.Input.route,
                                        inclusive = false
                                    )
                                }
                            )
                        } else {
                            navController.popBackStack(
                                NavRoutes.Input.route,
                                inclusive = false
                            )
                        }
                    }

                    composable(NavRoutes.History.route) {
                        HistoryScreen(
                            onBack = { navController.popBackStack() },
                            onOpen = { record ->
                                val json = java.io.File(record.dir)
                                    .listFiles { f -> f.name.endsWith("_result.json") }
                                    ?.firstOrNull()
                                if (json != null) {
                                    navController.navigate(
                                        NavRoutes.Result.create(file = json.absolutePath)
                                    )
                                }
                            }
                        )
                    }
                }
            }
        }
    }
}
