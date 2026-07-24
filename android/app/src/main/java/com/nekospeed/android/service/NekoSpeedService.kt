package com.nekospeed.android.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import com.nekospeed.android.MainActivity
import com.nekospeed.android.R
import com.nekospeed.android.data.NekoSpeedRunner
import com.nekospeed.android.data.ResultRepository
import com.nekospeed.android.viewmodel.TestType
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.CancellationException

class NekoSpeedService : Service() {

    private var job: Job? = null
    private val serviceScope = CoroutineScope(Dispatchers.IO + Job())
    private val runner by lazy { NekoSpeedRunner(this) }
    private val repository by lazy { ResultRepository() }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    override fun onDestroy() {
        super.onDestroy()
        job?.cancel()
        serviceScope.cancel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START -> {
                if (job?.isActive == true) return START_NOT_STICKY

                val sub = intent.getStringExtra(EXTRA_SUB) ?: return START_NOT_STICKY
                val tests = intent.getStringArrayListExtra(EXTRA_TESTS) ?: TestType.ALL
                val threads = intent.getIntExtra(EXTRA_THREADS, 8)
                val duration = intent.getIntExtra(EXTRA_DURATION, 10)
                val concurrency = intent.getIntExtra(EXTRA_CONCURRENCY, 1)
                val pingCount = intent.getIntExtra(EXTRA_PING_COUNT, 4)
                val filter = intent.getStringExtra(EXTRA_FILTER)

                NekoSpeedServiceBus.reset()
                startForeground(NOTIFICATION_ID, buildNotification("准备运行...", true))
                startTest(
                    sub = sub,
                    tests = tests,
                    threads = threads,
                    duration = duration,
                    concurrency = concurrency,
                    pingCount = pingCount,
                    filter = filter
                )
            }
            ACTION_CANCEL -> {
                runner.cancel()
                job?.cancel()
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
        }
        return START_NOT_STICKY
    }

    private fun startTest(
        sub: String,
        tests: List<String>,
        threads: Int,
        duration: Int,
        concurrency: Int,
        pingCount: Int,
        filter: String?
    ) {
        job = serviceScope.launch {
            try {
                NekoSpeedServiceBus.setState(ServiceState.Preparing)
                updateNotification("正在释放测速二进制...", true)
                runner.ensureBinary()
                runner.ensureFont()

                NekoSpeedServiceBus.setState(ServiceState.Running)
                updateNotification("NekoSpeed 正在测速...", true)

                val result = runner.run(
                    sub = sub,
                    tests = tests,
                    threads = threads,
                    duration = duration,
                    concurrency = concurrency,
                    pingCount = pingCount,
                    filter = filter,
                    onLog = { line ->
                        NekoSpeedServiceBus.appendLog(line)
                    }
                )

                when (result) {
                    is NekoSpeedRunner.Result.Success -> {
                        val parseResult = repository.loadFromFile(result.file)
                        when (parseResult) {
                            is ResultRepository.Result.Success -> {
                                repository.archiveResult(
                                    this@NekoSpeedService,
                                    result.file.parentFile,
                                    parseResult.data
                                )
                                NekoSpeedServiceBus.setState(
                                    ServiceState.Done(
                                        result = parseResult.data,
                                        outDir = result.file.parentFile
                                    )
                                )
                                updateNotification("测速完成", false)
                            }
                            is ResultRepository.Result.Error -> {
                                NekoSpeedServiceBus.setState(ServiceState.Error(parseResult.message))
                                updateNotification("测速失败: ${parseResult.message}", false)
                            }
                        }
                    }
                    is NekoSpeedRunner.Result.Error -> {
                        NekoSpeedServiceBus.setState(ServiceState.Error(result.message))
                        updateNotification("测速失败: ${result.message}", false)
                    }
                }
            } catch (e: CancellationException) {
                NekoSpeedServiceBus.setState(ServiceState.Idle)
                updateNotification("测速已取消", false)
            } catch (e: Exception) {
                NekoSpeedServiceBus.setState(ServiceState.Error(e.message ?: "未知错误"))
                updateNotification("测速失败: ${e.message}", false)
            } finally {
                job = null
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
        }
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "NekoSpeed 测速服务",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "保持 NekoSpeed 测速在后台运行"
                setShowBadge(false)
            }
            val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            manager.createNotificationChannel(channel)
        }
    }

    private fun buildNotification(content: String, showCancel: Boolean): Notification {
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )

        val builder = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle("NekoSpeed")
            .setContentText(content)
            .setOngoing(showCancel)
            .setContentIntent(pendingIntent)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setOnlyAlertOnce(true)

        if (showCancel) {
            val cancelIntent = Intent(this, NekoSpeedService::class.java).apply {
                action = ACTION_CANCEL
            }
            val cancelPendingIntent = PendingIntent.getService(
                this,
                1,
                cancelIntent,
                PendingIntent.FLAG_IMMUTABLE
            )
            builder.addAction(
                R.drawable.ic_launcher_foreground,
                "取消",
                cancelPendingIntent
            )
        }

        return builder.build()
    }

    private fun updateNotification(content: String, showCancel: Boolean) {
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.notify(NOTIFICATION_ID, buildNotification(content, showCancel))
    }

    companion object {
        const val ACTION_START = "com.nekospeed.android.action.START"
        const val ACTION_CANCEL = "com.nekospeed.android.action.CANCEL"

        const val EXTRA_SUB = "sub"
        const val EXTRA_TESTS = "tests"
        const val EXTRA_THREADS = "threads"
        const val EXTRA_DURATION = "duration"
        const val EXTRA_CONCURRENCY = "concurrency"
        const val EXTRA_PING_COUNT = "ping_count"
        const val EXTRA_FILTER = "filter"

        private const val CHANNEL_ID = "nekospeed_service_channel"
        private const val NOTIFICATION_ID = 1001

        fun createStartIntent(
            context: Context,
            sub: String,
            tests: List<String>,
            threads: Int,
            duration: Int,
            concurrency: Int,
            pingCount: Int,
            filter: String?
        ): Intent {
            return Intent(context, NekoSpeedService::class.java).apply {
                action = ACTION_START
                putExtra(EXTRA_SUB, sub)
                putStringArrayListExtra(EXTRA_TESTS, ArrayList(tests))
                putExtra(EXTRA_THREADS, threads)
                putExtra(EXTRA_DURATION, duration)
                putExtra(EXTRA_CONCURRENCY, concurrency)
                putExtra(EXTRA_PING_COUNT, pingCount)
                putExtra(EXTRA_FILTER, filter)
            }
        }
    }
}
