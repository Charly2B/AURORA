package dev.remyj.auroratv.ui

import android.net.Uri
import android.view.ViewGroup
import androidx.compose.foundation.background
import androidx.compose.foundation.focusable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.PlayerView
import androidx.tv.material3.Text
import dev.remyj.auroratv.ui.components.ActionButton
import dev.remyj.auroratv.ui.theme.Aurora
import kotlinx.coroutines.delay

/**
 * Lecture locale d'une vidéo du disque USB : le même ExoPlayer, mais sans NAS (pas de remux, de
 * conversion ni de session). Commandes volontairement simples : lecture/pause, ±10/30 s, arrêt,
 * temps restant. Retour referme le lecteur et revient au navigateur.
 */
@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
@Composable
fun UsbPlayerScreen(vm: AppViewModel, screen: Screen.UsbPlayer) {
    val context = LocalContext.current
    val uri = remember(screen.uri) { Uri.parse(screen.uri) }
    val player = remember(uri) {
        ExoPlayer.Builder(context).build().apply {
            setMediaItem(MediaItem.fromUri(uri))
            prepare()
            playWhenReady = true
        }
    }
    var controlsVisible by remember { mutableStateOf(false) }
    var isPlaying by remember { mutableStateOf(true) }
    var positionMs by remember { mutableLongStateOf(0L) }
    var error by remember { mutableStateOf<String?>(null) }
    val rootFocus = remember { FocusRequester() }
    val playFocus = remember { FocusRequester() }
    val overlayFocus = remember { FocusRequester() }

    fun stop() = vm.back()

    DisposableEffect(uri) {
        val listener = object : Player.Listener {
            override fun onIsPlayingChanged(playing: Boolean) { isPlaying = playing }
            override fun onPlaybackStateChanged(state: Int) {
                if (state == Player.STATE_ENDED) vm.back()
            }
            override fun onPlayerError(e: PlaybackException) {
                error = "Impossible de lire ce fichier (${e.errorCodeName})."
            }
        }
        player.addListener(listener)
        onDispose {
            player.removeListener(listener)
            player.release()
        }
    }

    LaunchedEffect(controlsVisible) {
        while (controlsVisible) {
            positionMs = player.currentPosition
            delay(500)
        }
    }
    LaunchedEffect(controlsVisible, isPlaying) {
        if (controlsVisible && isPlaying) {
            delay(6_000)
            controlsVisible = false
        }
    }
    // Focus : panneau → bouton lecture/pause ; sinon le fond (sauf si un état recouvrant est affiché).
    LaunchedEffect(controlsVisible) {
        if (controlsVisible) {
            delay(60)
            runCatching { playFocus.requestFocus() }
        } else if (vm.usbVolumes.isNotEmpty() && error == null) {
            runCatching { rootFocus.requestFocus() }
        }
    }
    LaunchedEffect(vm.usbVolumes.isEmpty()) {
        if (vm.usbVolumes.isEmpty()) {
            delay(60)
            runCatching { overlayFocus.requestFocus() }
        }
    }

    // Touche Accueil : la lecture s'arrête quand l'activité passe en arrière-plan.
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_STOP) stop()
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    androidx.activity.compose.BackHandler {
        if (controlsVisible) controlsVisible = false else stop()
    }

    val diskGone = vm.usbVolumes.isEmpty()

    Box(
        Modifier
            .fillMaxSize()
            .background(Color.Black)
            .focusRequester(rootFocus)
            .onPreviewKeyEvent { event ->
                if (event.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
                // Panneau, erreur ou disque retiré : le fond laisse les touches aux boutons (sinon
                // les commandes restaient inatteignables), sauf lecture/pause et arrêt.
                val overlayActive = controlsVisible || vm.usbVolumes.isEmpty() || error != null
                if (overlayActive && event.key != Key.MediaStop && event.key != Key.MediaPlayPause) return@onPreviewKeyEvent false
                when (event.key) {
                    Key.Enter, Key.DirectionCenter -> { controlsVisible = !controlsVisible; true }
                    Key.MediaPlayPause -> { if (player.isPlaying) player.pause() else player.play(); controlsVisible = true; true }
                    Key.MediaPlay -> { player.play(); controlsVisible = true; true }
                    Key.MediaPause -> { player.pause(); controlsVisible = true; true }
                    Key.MediaStop -> { stop(); true }
                    Key.MediaRewind, Key.DirectionLeft -> { player.seekTo((player.currentPosition - 10_000).coerceAtLeast(0)); controlsVisible = true; true }
                    Key.MediaFastForward, Key.DirectionRight -> { player.seekTo(player.currentPosition + 30_000); controlsVisible = true; true }
                    Key.DirectionUp, Key.DirectionDown -> { controlsVisible = true; true }
                    else -> false
                }
            }
            .focusable(),
    ) {
        AndroidView(
            factory = { viewContext ->
                PlayerView(viewContext).apply {
                    this.player = player
                    useController = false
                    isFocusable = false
                    isFocusableInTouchMode = false
                    keepScreenOn = true
                    layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
                }
            },
            modifier = Modifier.fillMaxSize(),
        )

        if (controlsVisible && error == null && !diskGone) {
            LocalControlsPanel(
                title = screen.title,
                isPlaying = isPlaying,
                positionMs = positionMs,
                durationMs = player.duration,
                playFocus = playFocus,
                onSeek = { delta -> player.seekTo((player.currentPosition + delta).coerceAtLeast(0)); positionMs = player.currentPosition },
                onPlayPause = { if (player.isPlaying) player.pause() else player.play() },
                onStop = ::stop,
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }

        if (diskGone) {
            Box(Modifier.fillMaxSize().background(Color(0xE6000000)), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("Le disque a été débranché.", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 24.sp)
                    Text("Rebranchez-le pour continuer la lecture.", color = Aurora.Ash, fontSize = 15.sp, modifier = Modifier.padding(top = 8.dp))
                    ActionButton("Retour", onClick = ::stop, primary = true, modifier = Modifier.padding(top = 24.dp).focusRequester(overlayFocus))
                }
            }
        } else {
            error?.let { message -> LocalErrorPanel(message, ::stop) }
        }
    }
}

@Composable
private fun LocalControlsPanel(
    title: String,
    isPlaying: Boolean,
    positionMs: Long,
    durationMs: Long,
    playFocus: FocusRequester,
    onSeek: (Long) -> Unit,
    onPlayPause: () -> Unit,
    onStop: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier
            .fillMaxWidth()
            .background(Aurora.Night.copy(alpha = 0.95f))
            .padding(horizontal = 48.dp, vertical = 20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(verticalAlignment = Alignment.Bottom) {
            Text(title, color = Aurora.Paper, fontSize = 20.sp, fontWeight = FontWeight.Bold, maxLines = 1,
                overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f, fill = false))
            Text("LOCAL · USB", color = Aurora.Ash, fontSize = 13.sp, modifier = Modifier.padding(start = 14.dp, bottom = 2.dp))
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(formatTime(positionMs), color = Aurora.Paper, fontSize = 16.sp, modifier = Modifier.width(80.dp))
            LocalProgressBar(positionMs, durationMs, Modifier.weight(1f))
            Text(formatTime(durationMs), color = Aurora.Paper, fontSize = 16.sp, modifier = Modifier.width(80.dp).padding(start = 12.dp))
            Text("−" + formatTime((durationMs - positionMs).coerceAtLeast(0)), color = Aurora.Ash, fontSize = 13.sp, modifier = Modifier.width(74.dp).padding(start = 8.dp))
        }
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
            RoundButton(glyph = "↺", caption = "10 s", onClick = { onSeek(-10_000) })
            Spacer(Modifier.width(18.dp))
            RoundButton(glyph = if (isPlaying) "❚❚" else "▶", caption = if (isPlaying) "Pause" else "Lecture", big = true, onClick = onPlayPause, modifier = Modifier.focusRequester(playFocus))
            Spacer(Modifier.width(18.dp))
            RoundButton(glyph = "↻", caption = "30 s", onClick = { onSeek(30_000) })
            Spacer(Modifier.width(48.dp))
            RoundButton(glyph = "■", caption = "Arrêter", onClick = onStop)
        }
    }
}

@Composable
private fun LocalProgressBar(positionMs: Long, durationMs: Long, modifier: Modifier = Modifier) {
    val fraction = if (durationMs > 0) (positionMs.toFloat() / durationMs).coerceIn(0f, 1f) else 0f
    Box(modifier.height(8.dp).background(Color(0x33FFFFFF), RoundedCornerShape(4.dp))) {
        Box(Modifier.fillMaxWidth(fraction).height(8.dp).background(Aurora.Accent, RoundedCornerShape(4.dp)))
    }
}

@Composable
private fun LocalErrorPanel(message: String, onBack: () -> Unit) {
    val focus = remember { FocusRequester() }
    LaunchedEffect(Unit) { delay(60); runCatching { focus.requestFocus() } }
    Column(
        Modifier.fillMaxSize().background(Color(0xEB000000)).padding(64.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text("Lecture impossible", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 34.sp)
        Text(message, color = Aurora.Mist, fontSize = 17.sp, modifier = Modifier.padding(top = 12.dp).fillMaxWidth())
        ActionButton("Retour", onClick = onBack, primary = true, modifier = Modifier.padding(top = 28.dp).focusRequester(focus))
    }
}
