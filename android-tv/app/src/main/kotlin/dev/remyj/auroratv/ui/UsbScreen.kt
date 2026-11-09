package dev.remyj.auroratv.ui

import android.content.Intent
import android.media.MediaMetadataRetriever
import android.net.Uri
import android.provider.Settings
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.tv.material3.ClickableSurfaceDefaults
import androidx.tv.material3.Surface
import androidx.tv.material3.Text
import dev.remyj.auroratv.ui.components.ActionButton
import dev.remyj.auroratv.ui.theme.Aurora
import dev.remyj.auroratv.usb.formatBytes
import dev.remyj.auroratv.usb.formatDuration
import dev.remyj.auroratv.usb.isVideoFile
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext

/**
 * Source locale « Disque USB » : navigateur de fichiers du disque branché sur le Formovie.
 *
 * Lecture **directe** du point de montage (`/storage/…`), comme VLC : le Formovie n'a pas de
 * sélecteur de documents (DocumentsUI), le Storage Access Framework y plante donc. Trois états :
 * aucun disque, autorisation « Tous les fichiers » manquante, et le navigateur (dossiers puis
 * vidéos). La lecture est locale (ExoPlayer), sans passer par le NAS.
 */
@Composable
fun UsbScreen(vm: AppViewModel) {
    val context = LocalContext.current
    // Recontrôle l'autorisation au retour des réglages : l'accord « Tous les fichiers » n'est pas un
    // état Compose, il faut le relire quand l'activité reprend la main.
    var canBrowse by remember { mutableStateOf(vm.usbCanBrowse) }
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) canBrowse = vm.usbCanBrowse
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    if (vm.usbVolumes.isEmpty()) {
        UsbEmptyState()
        return
    }
    if (!canBrowse) {
        UsbStorageAccessPrompt {
            runCatching { context.startActivity(Intent(Settings.ACTION_SETTINGS)) }
        }
        return
    }
    val path = vm.usbPath
    if (path == null) {
        UsbEmptyState()
        return
    }
    val root = remember(path) { File(path) }
    UsbBrowser(root, vm.usbLabel ?: "Disque USB", vm.usbCapacity, onOpen = { file ->
        vm.push(Screen.UsbPlayer(Uri.fromFile(file).toString(), file.name))
    })
}

@Composable
private fun UsbEmptyState() {
    Column(
        Modifier.fillMaxSize().padding(horizontal = 32.dp),
        verticalArrangement = Arrangement.Center,
    ) {
        Text("Disque USB", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 30.sp)
        Text("Aucun disque branché.", color = Aurora.Mist, fontSize = 18.sp, modifier = Modifier.padding(top = 14.dp).fillMaxWidth())
        Text(
            "Branchez un disque USB sur le Formovie pour parcourir et lire les vidéos qu'il contient.",
            color = Aurora.Ash, fontSize = 15.sp, modifier = Modifier.padding(top = 6.dp).fillMaxWidth(),
        )
    }
}

@Composable
private fun UsbStorageAccessPrompt(onOpenSettings: () -> Unit) {
    val focus = remember { FocusRequester() }
    LaunchedEffect(Unit) { delay(60); runCatching { focus.requestFocus() } }
    Column(
        Modifier.fillMaxSize().padding(horizontal = 48.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text("AUTORISATION REQUISE", color = Aurora.Accent, fontSize = 11.sp, letterSpacing = 2.6.sp, fontWeight = FontWeight.SemiBold)
        Text("Autoriser l'accès au stockage", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 26.sp, modifier = Modifier.padding(top = 8.dp))
        Text(
            "Pour lire les vidéos du disque USB, Aurora a besoin de l'autorisation « Tous les fichiers ».\n" +
                "Ouvrez les réglages, puis Applications → AURORA → Autorisations.",
            color = Aurora.Mist, fontSize = 15.sp, lineHeight = 21.sp,
            modifier = Modifier.padding(top = 14.dp).width(680.dp),
        )
        ActionButton("Ouvrir les réglages", onClick = onOpenSettings, primary = true, modifier = Modifier.padding(top = 26.dp).focusRequester(focus))
    }
}

@Composable
private fun UsbBrowser(root: File, label: String, capacity: String?, onOpen: (File) -> Unit) {
    // Pile des dossiers ouverts ; le dernier est le dossier affiché.
    val path = remember(root) { mutableStateListOf(root) }
    val current = path.last()
    val entries by produceState<List<File>>(emptyList(), current.absolutePath) {
        value = withContext(Dispatchers.IO) {
            val files = current.listFiles()
            if (files == null) {
                android.util.Log.w("AuroraUsb", "Impossible de lister ${current.absolutePath}")
                emptyList()
            } else {
                val listed = files.filter { it.isDirectory || isVideoFile(it.name, null) }
                    .sortedWith(compareByDescending<File> { it.isDirectory }.thenBy { it.name.lowercase() })
                android.util.Log.i("AuroraUsb", "${current.absolutePath} -> ${listed.size} éléments")
                listed
            }
        }
    }
    // Retour = remonter d'un dossier ; à la racine, le gestionnaire global ramène à l'Accueil.
    BackHandler(enabled = path.size > 1) { path.removeAt(path.lastIndex) }
    // Focus sur la première tuile uniquement après être entré dans un dossier (les anciennes tuiles
    // disparaissent) : pas au premier affichage, sinon le navigateur volait le focus à la colonne
    // quand on survolait simplement « Disque USB ».
    var autoFocus by remember { mutableStateOf(false) }

    Column(Modifier.fillMaxSize()) {
        Text("Disque USB", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 30.sp, modifier = Modifier.padding(start = 32.dp, top = 20.dp))
        Text(
            listOfNotNull(label, capacity).joinToString(" · "),
            color = Aurora.Ash, fontSize = 13.sp, modifier = Modifier.padding(start = 32.dp, top = 3.dp),
        )
        Breadcrumb(path, label)
        if (entries.isEmpty()) {
            Message("Ce dossier est vide.")
        } else {
            LazyVerticalGrid(
                columns = GridCells.Fixed(5),
                contentPadding = PaddingValues(start = 32.dp, end = 32.dp, top = 12.dp, bottom = 48.dp),
                horizontalArrangement = Arrangement.spacedBy(22.dp),
                verticalArrangement = Arrangement.spacedBy(24.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                items(entries, key = { it.absolutePath }) { file ->
                    val first = file == entries.first()
                    if (file.isDirectory) {
                        FolderTile(file, onClick = { path.add(file); autoFocus = true }, autoFocus = autoFocus && first, onAutoFocused = { autoFocus = false })
                    } else {
                        VideoTile(file, onClick = { onOpen(file) }, autoFocus = autoFocus && first, onAutoFocused = { autoFocus = false })
                    }
                }
            }
        }
    }
}

@Composable
private fun Breadcrumb(path: List<File>, rootLabel: String) {
    Row(
        Modifier
            .padding(start = 32.dp, end = 32.dp, top = 16.dp)
            .background(Aurora.Night.copy(alpha = 0.9f), RoundedCornerShape(9.dp))
            .border(1.dp, Aurora.Accent.copy(alpha = 0.24f), RoundedCornerShape(9.dp))
            .padding(horizontal = 14.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        path.forEachIndexed { i, f ->
            if (i > 0) Text(" › ", color = Aurora.AccentDeep, fontSize = 13.sp)
            val name = if (i == 0) rootLabel else (f.name ?: "")
            val last = i == path.lastIndex
            Text(
                name, color = if (last) Aurora.Paper else Aurora.Ash,
                fontFamily = if (last) Aurora.Display else null, fontSize = 13.sp,
                maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.widthIn(max = 180.dp),
            )
        }
    }
}

/** Vignette de dossier : glyphe centré sur un fond sombre. */
@Composable
private fun FolderTile(file: File, onClick: () -> Unit, autoFocus: Boolean, onAutoFocused: () -> Unit) {
    UsbTile(file.name ?: "Dossier", "", onClick, autoFocus, onAutoFocused) {
        Box(
            Modifier.fillMaxSize().background(Brush.linearGradient(listOf(Aurora.Graphite, Aurora.Coal))),
            contentAlignment = Alignment.Center,
        ) {
            Text("📁", fontSize = 30.sp)
        }
    }
}

private data class VideoMeta(val thumb: ImageBitmap?, val durationMs: Long)

/** Vignette de vidéo : image extraite du fichier si possible, sinon glyphe « film ». */
@Composable
private fun VideoTile(file: File, onClick: () -> Unit, autoFocus: Boolean, onAutoFocused: () -> Unit) {
    val size = file.length()
    val meta by produceState<VideoMeta?>(null, file.absolutePath) {
        value = withContext(Dispatchers.IO) { extractVideoMeta(file) }
    }
    val metaLine = listOfNotNull(
        (meta?.durationMs ?: 0L).takeIf { it > 0 }?.let { formatDuration(it / 1000) },
        size.takeIf { it > 0 }?.let { formatBytes(it) },
    ).joinToString(" · ")
    UsbTile(file.name ?: "Vidéo", metaLine, onClick, autoFocus, onAutoFocused) {
        val thumb = meta?.thumb
        if (thumb != null) {
            Image(thumb, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
        } else {
            Box(Modifier.fillMaxSize().background(Brush.linearGradient(listOf(Aurora.Graphite, Aurora.Night))), contentAlignment = Alignment.Center) {
                Text("🎬", fontSize = 30.sp)
            }
        }
    }
}

private fun extractVideoMeta(file: File): VideoMeta? = runCatching {
    val r = MediaMetadataRetriever()
    try {
        r.setDataSource(file.absolutePath)
        val frame = r.getFrameAtTime(1_000_000)
        val durationMs = r.extractMetadata(MediaMetadataRetriever.METADATA_KEY_DURATION)?.toLongOrNull() ?: 0L
        VideoMeta(frame?.asImageBitmap(), durationMs)
    } finally {
        r.release()
    }
}.getOrNull()

/** Tuile du navigateur USB : même cadre verre et focus crème-or que les affiches. */
@Composable
private fun UsbTile(
    title: String, meta: String, onClick: () -> Unit, autoFocus: Boolean, onAutoFocused: () -> Unit,
    art: @Composable BoxScope.() -> Unit,
) {
    var focused by remember { mutableStateOf(false) }
    val requester = remember { FocusRequester() }
    LaunchedEffect(autoFocus) {
        if (!autoFocus) return@LaunchedEffect
        delay(60)
        runCatching { requester.requestFocus() }
        onAutoFocused()
    }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(7.dp)) {
        Box(Modifier.graphicsLayer { val s = if (focused) 1.04f else 1f; scaleX = s; scaleY = s }) {
            Surface(
                onClick = onClick,
                shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(10.dp)),
                colors = ClickableSurfaceDefaults.colors(containerColor = Color.Transparent, focusedContainerColor = Color.Transparent),
                scale = ClickableSurfaceDefaults.scale(focusedScale = 1f),
                modifier = Modifier
                    .fillMaxWidth()
                    .shadow(if (focused) 18.dp else 10.dp, RoundedCornerShape(10.dp), clip = false,
                        ambientColor = if (focused) Aurora.Accent else Color.Black,
                        spotColor = if (focused) Aurora.Accent else Color.Black)
                    .then(if (focused) Modifier.border(7.dp, Aurora.Accent.copy(alpha = 0.32f), RoundedCornerShape(10.dp)) else Modifier)
                    .focusRequester(requester)
                    .onFocusChanged { focused = it.isFocused },
            ) {
                Box(
                    Modifier
                        .background(Brush.linearGradient(listOf(Color(0x38FFFFFF), Color(0x0DFFFFFF), Color(0x14FFFFFF))), RoundedCornerShape(10.dp))
                        .border(if (focused) 4.dp else 1.dp, if (focused) Aurora.FocusGold else Aurora.Paper.copy(alpha = 0.34f), RoundedCornerShape(10.dp))
                        .padding(5.dp),
                ) {
                    Box(Modifier.fillMaxWidth().aspectRatio(16f / 9f).clip(RoundedCornerShape(6.dp)).background(Aurora.Coal)) { art() }
                }
            }
        }
        Text(title, color = if (focused) Aurora.FocusGold else Aurora.Mist, fontFamily = Aurora.Display, fontSize = 14.sp,
            fontWeight = if (focused) FontWeight.SemiBold else FontWeight.Normal,
            maxLines = 1, overflow = TextOverflow.Ellipsis)
        if (meta.isNotEmpty()) Text(meta, color = Aurora.Ash, fontSize = 11.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}
