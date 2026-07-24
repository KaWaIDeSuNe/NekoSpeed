package com.nekospeed.android.ui.components

import android.widget.Toast
import androidx.compose.foundation.Image
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.nekospeed.android.util.ShareHelper
import java.io.File

@Composable
fun ReportImagesTab(files: List<File>, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    var preview by remember { mutableStateOf<File?>(null) }

    if (files.isEmpty()) {
        Box(modifier = modifier, contentAlignment = Alignment.Center) {
            Text(
                text = "本次测试没有生成图片",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        return
    }

    LazyVerticalGrid(
        columns = GridCells.Fixed(2),
        modifier = modifier,
        contentPadding = androidx.compose.foundation.layout.PaddingValues(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(files, key = { it.name }) { file ->
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { preview = file }
            ) {
                AsyncImage(
                    model = file,
                    contentDescription = file.name,
                    modifier = Modifier
                        .fillMaxWidth()
                        .aspectRatio(1f),
                    contentScale = ContentScale.Crop
                )
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 8.dp, vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = file.name,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        modifier = Modifier.weight(1f)
                    )
                    IconButton(onClick = { ShareHelper.shareFile(context, file, "分享图片") }) {
                        Icon(Icons.Default.Share, contentDescription = "分享")
                    }
                    IconButton(
                        onClick = {
                            val uri = ShareHelper.saveImageToGallery(context, file)
                            Toast.makeText(
                                context,
                                if (uri != null) "已保存到相册 Pictures/NekoSpeed/" else "保存失败",
                                Toast.LENGTH_SHORT
                            ).show()
                        }
                    ) {
                        Icon(Icons.Default.Download, contentDescription = "保存到相册")
                    }
                }
            }
        }
    }

    if (preview != null) {
        ImagePreviewSheet(file = preview!!, onDismiss = { preview = null })
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ImagePreviewSheet(file: File, onDismiss: () -> Unit) {
    val context = LocalContext.current
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 24.dp)
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = file.name,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f)
                )
                TextButton(onClick = { ShareHelper.shareFile(context, file, "分享图片") }) {
                    Icon(Icons.Default.Share, contentDescription = null)
                    Spacer(Modifier.width(4.dp))
                    Text("分享")
                }
                TextButton(
                    onClick = {
                        val uri = ShareHelper.saveImageToGallery(context, file)
                        Toast.makeText(
                            context,
                            if (uri != null) "已保存到相册 Pictures/NekoSpeed/" else "保存失败",
                            Toast.LENGTH_SHORT
                        ).show()
                    }
                ) {
                    Icon(Icons.Default.Download, contentDescription = null)
                    Spacer(Modifier.width(4.dp))
                    Text("保存")
                }
            }
            Box(modifier = Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                AsyncImage(
                    model = file,
                    contentDescription = file.name,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp),
                    contentScale = ContentScale.Fit
                )
            }
        }
    }
}
