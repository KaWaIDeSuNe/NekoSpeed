package com.nekospeed.android.util

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.Build
import android.provider.Settings

object VpnDetector {

    fun isVpnActive(context: Context): Boolean {
        return isVpnTransportActive(context) || hasGlobalProxy(context) || hasSystemProxy()
    }

    private fun isVpnTransportActive(context: Context): Boolean {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
            ?: return false
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            cm.activeNetwork?.let { network ->
                cm.getNetworkCapabilities(network)?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true
            } ?: false
        } else {
            @Suppress("DEPRECATION")
            cm.allNetworks.any { network ->
                cm.getNetworkCapabilities(network)?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true
            }
        }
    }

    private fun hasGlobalProxy(context: Context): Boolean {
        return try {
            val proxy = Settings.Global.getString(context.contentResolver, Settings.Global.HTTP_PROXY)
            !proxy.isNullOrBlank()
        } catch (_: Exception) {
            false
        }
    }

    private fun hasSystemProxy(): Boolean {
        return !System.getProperty("http.proxyHost").isNullOrBlank()
    }

    fun getWarningMessage(): String {
        return "检测到设备已启用 VPN 或代理（如 Clash/V2ray）。\n\n" +
                "在 VPN/代理全局模式下运行测速，结果会严重失真（节点经过本地代理后再出站）。\n\n" +
                "建议关闭全局 VPN 后重新测试，或选择\"仍要运行\"继续。"
    }
}
