package com.nekospeed.android.util

import android.content.ContentValues
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Environment
import android.provider.MediaStore
import androidx.core.content.FileProvider
import java.io.File

object ShareHelper {

    fun shareResult(context: Context, outDir: File) {
        val files = collectShareFiles(outDir)
        if (files.isEmpty()) {
            context.toast("没有可分享的文件")
            return
        }

        val uris = files.map { getUriForFile(context, it) }
        val intent = if (uris.size == 1) {
            Intent(Intent.ACTION_SEND).apply {
                type = getMimeType(uris.first())
                putExtra(Intent.EXTRA_STREAM, uris.first())
            }
        } else {
            Intent(Intent.ACTION_SEND_MULTIPLE).apply {
                type = "*/*"
                putParcelableArrayListExtra(Intent.EXTRA_STREAM, ArrayList(uris))
            }
        }

        startChooser(context, intent, "分享 NekoSpeed 结果")
    }

    fun shareFile(context: Context, file: File, title: String = "分享") {
        if (!file.exists()) {
            context.toast("文件不存在")
            return
        }
        val uri = getUriForFile(context, file)
        val intent = Intent(Intent.ACTION_SEND).apply {
            type = getMimeType(uri)
            putExtra(Intent.EXTRA_STREAM, uri)
        }
        startChooser(context, intent, title)
    }

    /** 保存 PNG 到系统相册 Pictures/NekoSpeed/，返回 MediaStore Uri */
    fun saveImageToGallery(context: Context, file: File): Uri? {
        if (!file.exists()) return null
        val values = ContentValues().apply {
            put(MediaStore.Images.Media.DISPLAY_NAME, file.name)
            put(MediaStore.Images.Media.MIME_TYPE, "image/png")
            put(
                MediaStore.Images.Media.RELATIVE_PATH,
                Environment.DIRECTORY_PICTURES + "/NekoSpeed"
            )
            put(MediaStore.Images.Media.IS_PENDING, 1)
        }
        val uri = context.contentResolver
            .insert(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, values) ?: return null
        val ok = runCatching {
            context.contentResolver.openOutputStream(uri)?.use { out ->
                file.inputStream().use { it.copyTo(out) }
            } ?: return null
        }.isSuccess
        values.clear()
        values.put(MediaStore.Images.Media.IS_PENDING, 0)
        context.contentResolver.update(uri, values, null, null)
        return if (ok) uri else null
    }

    private fun startChooser(context: Context, intent: Intent, title: String) {
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        val chooser = Intent.createChooser(intent, title)
        chooser.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        context.startActivity(chooser)
    }

    private fun collectShareFiles(outDir: File): List<File> {
        if (!outDir.exists() || !outDir.isDirectory) return emptyList()
        return outDir.listFiles { f: File ->
            f.name.endsWith("_result.json") || f.name.endsWith(".png")
        }?.sortedBy { it.lastModified() } ?: emptyList()
    }

    private fun getUriForFile(context: Context, file: File): Uri {
        return FileProvider.getUriForFile(
            context,
            "${context.packageName}.fileprovider",
            file
        )
    }

    private fun getMimeType(uri: Uri): String {
        return when (uri.path?.substringAfterLast('.')) {
            "json" -> "application/json"
            "png" -> "image/png"
            else -> "*/*"
        }
    }

    private fun Context.toast(message: String) {
        android.widget.Toast.makeText(this, message, android.widget.Toast.LENGTH_SHORT).show()
    }
}
