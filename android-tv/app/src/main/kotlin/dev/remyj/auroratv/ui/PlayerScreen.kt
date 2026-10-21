package dev.remyj.auroratv.ui

import android.content.Intent
import android.net.Uri
import android.view.ViewGroup
import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
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
import androidx.media3.common.C
import androidx.media3.common.Format
import androidx.media3.common.MediaItem
import androidx.media3.common.MimeTypes
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.TrackSelectionOverride
import androidx.media3.common.Tracks
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlaybackException
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import androidx.tv.material3.Button
import androidx.tv.material3.Text
import dev.remyj.auroratv.net.ApiException
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.Card
import dev.remyj.auroratv.net.PlayRequest
import dev.remyj.auroratv.net.PlayResponse
import dev.remyj.auroratv.net.Sessions
import dev.remyj.auroratv.ui.components.episodeCode
import dev.remyj.auroratv.ui.theme.Aurora
import kotlinx.coroutines.delay
import kotlinx.coroutines.Job
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.withContext
import kotlinx.coroutines.launch

/** Une piste proposée à la sélection (audio ou sous-titres). */
private data class TrackChoice(
    val label: String,
    val groupIndex: Int,
    val trackIndex: Int,
    val selected: Boolean,
)

private enum class TrackMenu { AUDIO, TEXT }

/** Lecture en cours : réponse du NAS + ce qu'il faut pour la reprise. */
private class Session(val play: PlayResponse, val title: String, val itemId: Long, val kind: String)

/**
 * Lecteur AURORA sur la box (ExoPlayer).
 *
 * Le NAS choisit : lecture directe du fichier (presque toujours sur la box : MKV, HEVC, AC3…), sinon
 * remux / conversion en HLS. Une session HLS démarre à un décalage : position absolue = offset + position
 * du lecteur. Un saut hors de la partie produite relance une session à la nouvelle position.
 *
 * Commandes reprises de PlexSeries TV (rodées sur cette box) : OK ouvre le panneau, ◀/▶ sautent,
 * piste française par défaut, la touche Accueil arrête la lecture.
 */
@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
@Composable
fun PlayerScreen(vm: AppViewModel, screen: Screen.Player) {
    val client = vm.client ?: return
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var session by remember { mutableStateOf<Session?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var triedRemux by remember { mutableStateOf(false) }
    var triedTranscode by remember { mutableStateOf(false) }
    // Mode imposé après un échec du décodeur : conservé pour les sessions suivantes (saut, piste audio),
    // sinon le NAS rechoisirait la lecture directe qui vient d'échouer.
    var forcedMode by remember { mutableStateOf<String?>(null) }
    // Piste audio choisie côté NAS (HLS) : reportée sur les sessions suivantes (saut lointain).
    var audioChoice by remember { mutableStateOf<Int?>(null) }
    // Génération de la dernière demande /play : une réponse plus ancienne (sauts rapprochés) est ignorée
    // et sa session arrêtée, sinon l'app garderait une session que le NAS vient de tuer.
    val startGen = remember { intArrayOf(0) }
    var pendingSeek by remember { mutableStateOf<Double?>(null) }
    var seekJob by remember { mutableStateOf<Job?>(null) }
    var networkRetried by remember { mutableStateOf(false) }
    var frenchApplied by remember { mutableStateOf(false) }
    var controlsVisible by remember { mutableStateOf(false) }
    var trackMenu by remember { mutableStateOf<TrackMenu?>(null) }
    var isPlaying by remember { mutableStateOf(true) }
    var positionMs by remember { mutableLongStateOf(0L) }
    var tracks by remember { mutableStateOf<Tracks?>(null) }
    var interaction by remember { mutableIntStateOf(0) }
    var next by remember { mutableStateOf<Card?>(null) }
    var countdown by remember { mutableStateOf<Int?>(null) }
    var ended by remember { mutableStateOf(false) }
    var restarts by remember { mutableIntStateOf(0) }

    val rootFocus = remember { FocusRequester() }
    val seekFocus = remember { FocusRequester() }
    val playFocus = remember { FocusRequester() }
    val prefs = vm.me?.prefs
    val prefLang = prefs?.audioLang?.let(::iso2)

    val player = remember {
        val headers = Sessions.cookieHeader(client.baseUrl)?.let { mapOf("Cookie" to it) } ?: emptyMap()
        val http = DefaultHttpDataSource.Factory().setDefaultRequestProperties(headers).setAllowCrossProtocolRedirects(true)
        ExoPlayer.Builder(context).setMediaSourceFactory(DefaultMediaSourceFactory(http)).build().apply {
            // Les box sont en en-US : sans préférence, les fichiers multipistes partaient en anglais.
            trackSelectionParameters = trackSelectionParameters.buildUpon().apply {
                if (prefLang != null) setPreferredAudioLanguages(prefLang, if (prefLang == "fr") "fra" else prefLang, if (prefLang == "fr") "fre" else prefLang)
                when (prefs?.subMode) {
                    "off" -> setTrackTypeDisabled(C.TRACK_TYPE_TEXT, true)
                    "always" -> setPreferredTextLanguage(iso2(prefs.subLang.ifEmpty { prefs.audioLang }))
                    else -> Unit // automatique : sous-titres forcés uniquement (comportement par défaut d'ExoPlayer)
                }
            }.build()
            playWhenReady = true
        }
    }

    /** Position absolue dans l'œuvre (secondes). */
    fun absolute(): Double = (session?.play?.offset ?: 0.0) + player.currentPosition / 1000.0

    /** (Re)demande la lecture au NAS à la position [start] et branche la source. */
    fun startAt(start: Double, audioIndex: Int? = null, forceMode: String? = null) {
        if (audioIndex != null) audioChoice = audioIndex
        val gen = ++startGen[0]
        scope.launch {
            error = null
            try {
                val detail = client.item(screen.itemId)
                val file = detail.files.firstOrNull { it.id == screen.fileId }
                    ?: detail.files.firstOrNull { it.id == detail.state.lastFileId }
                    ?: detail.files.firstOrNull()
                    ?: throw ApiException(404, "Aucun fichier vidéo pour cette œuvre.")
                val from = if (start >= 0) start
                else if (!screen.restart && !detail.state.watched && detail.state.position > 0) detail.state.position else 0.0
                // Pas d'arrêt anticipé de l'ancienne session : le NAS l'arrête lui-même au /play suivant (même
                // fichier), et l'arrêter avant ferait échouer l'ancien lecteur, dont l'erreur relancerait l'ancienne position.
                // Non annulable : si l'écran se ferme pendant la requête, on doit encore arrêter la session créée.
                val resp = withContext(NonCancellable) {
                    client.play(PlayRequest(fileId = file.id, caps = vm.caps, start = from, audioIndex = audioChoice, forceMode = forceMode ?: forcedMode))
                }
                if (gen != startGen[0]) {
                    resp.sessionId?.let { id -> GlobalScope.launch { client.stopSession(id) } }
                    return@launch
                }
                pendingSeek = null
                // Passage HLS → direct (ex. « Réessayer ») : le NAS ne démarre rien, donc n'arrête pas l'ancienne session.
                if (resp.mode == "direct") session?.play?.sessionId?.let { old -> GlobalScope.launch { client.stopSession(old) } }
                val title = if (detail.kind == "episode") "${detail.seriesTitle.orEmpty()} · ${episodeCode(detail.season, detail.episode, detail.episodeEnd)} · ${detail.title}" else detail.title
                session = Session(resp, title, detail.id, detail.kind)
                frenchApplied = false
                // HLS : playlist EVENT sans fin = « direct » pour ExoPlayer, qui démarrerait au bord produit
                // (jusqu'à plusieurs minutes plus loin en remux). On impose le début de la session.
                if (resp.mode == "direct") player.setMediaItem(buildMediaItem(client, resp))
                else player.setMediaItem(buildMediaItem(client, resp), 0L)
                player.prepare()
                if (resp.mode == "direct" && from > 0) player.seekTo((from * 1000).toLong())
                player.play()
            } catch (e: ApiException) {
                if (gen == startGen[0]) {
                    pendingSeek = null
                    error = e.message
                }
            }
        }
    }

    /** Position visée : un saut en attente compte, sinon des appuis rapides ne s'additionneraient pas. */
    fun current(): Double = pendingSeek ?: absolute()

    fun seekTo(targetSec: Double) {
        val s = session ?: return
        val total = s.play.duration
        val target = targetSec.coerceIn(0.0, if (total > 1) total - 1 else targetSec)
        if (s.play.mode == "direct") {
            player.seekTo((target * 1000).toLong())
            return
        }
        seekJob?.cancel()
        // HLS : dans la partie déjà produite, saut local ; sinon (ou retour arrière > 60 s, segments
        // déjà effacés par le NAS), nouvelle session.
        val local = target - s.play.offset
        val produced = if (player.duration > 0) player.duration / 1000.0 else 0.0
        if (local >= 0 && local <= produced - 2 && target >= absolute() - 60) {
            pendingSeek = null
            player.seekTo((local * 1000).toLong())
        }
        else {
            // Appuis rapprochés au-delà de la partie produite : une seule nouvelle session, à la position finale.
            pendingSeek = target
            seekJob?.cancel()
            seekJob = scope.launch {
                delay(500)
                restarts++
                startAt(target)
            }
        }
    }

    fun saveProgress() {
        val s = session ?: return
        val pos = absolute()
        // GlobalScope : appelé aussi pendant la fermeture de l'écran, dont le scope est déjà annulé.
        if (pos > 0) GlobalScope.launch { client.progress(s.itemId, s.play.fileId, pos, s.play.duration) }
    }

    fun stop() {
        saveProgress()
        vm.back()
    }

    fun showControls() {
        controlsVisible = true
        interaction++
    }

    LaunchedEffect(Unit) { startAt(-1.0) }

    DisposableEffect(Unit) {
        val listener = object : Player.Listener {
            override fun onPlayerError(e: PlaybackException) {
                val s = session
                // Décodeur vidéo de la box en échec (ex. HEVC 10 bits sur une puce qui ne fait que le
                // 8 bits) : le remux recopierait la même vidéo, seule la conversion par le NAS aide.
                val videoDecoder = e is ExoPlaybackException && e.type == ExoPlaybackException.TYPE_RENDERER &&
                    MimeTypes.isVideo(e.rendererFormat?.sampleMimeType)
                val network = e.errorCode / 1000 == 2 // famille ERROR_CODE_IO_* (réseau, HTTP 404/410…)
                when {
                    // Session HLS fermée par le NAS (pause > 90 s) ou coupure : une nouvelle session, une fois.
                    s != null && network && s.play.mode != "direct" && !networkRetried -> {
                        networkRetried = true
                        startAt(absolute())
                    }
                    s != null && videoDecoder && s.play.mode != "transcode" && !triedTranscode -> {
                        triedTranscode = true
                        forcedMode = "transcode"
                        startAt(absolute())
                    }
                    // Sinon (conteneur, audio) : remux par le NAS, puis VLC en dernier recours.
                    s != null && s.play.mode == "direct" && !triedRemux -> {
                        triedRemux = true
                        forcedMode = "remux"
                        startAt(absolute())
                    }
                    else -> error = "Le lecteur n'a pas pu lire ce fichier (${e.errorCodeName})."
                }
            }

            override fun onIsPlayingChanged(playing: Boolean) {
                isPlaying = playing
                if (playing) networkRetried = false
                if (!playing) saveProgress()
            }

            override fun onTracksChanged(updated: Tracks) {
                // Une fois par média : ensuite le choix manuel de l'utilisateur prime.
                if (prefLang == "fr" && !frenchApplied && updated.groups.any { it.type == C.TRACK_TYPE_AUDIO }) {
                    frenchApplied = true
                    selectFrenchAudio(player, updated)
                }
                tracks = updated
            }

            override fun onPlaybackStateChanged(state: Int) {
                if (state != Player.STATE_ENDED) return
                ended = true
                saveProgress()
                val s = session ?: return
                if (s.kind == "episode") {
                    scope.launch {
                        next = try { client.next(s.itemId) } catch (e: ApiException) { null }
                        if (next == null) vm.back() // dernier épisode : pas d'écran noir (annulé si l'écran est déjà fermé)
                        else if (prefs?.autoplayNext != false) countdown = 8
                    }
                } else {
                    vm.back()
                }
            }
        }
        player.addListener(listener)
        onDispose {
            startGen[0]++ // une réponse /play encore en vol arrêtera elle-même sa session
            saveProgress()
            session?.play?.sessionId?.let { id -> GlobalScope.launch { client.stopSession(id) } }
            player.removeListener(listener)
            player.release()
        }
    }

    // Touche Accueil : la vue reste composée et la lecture continuait en fond ; on coupe dès ON_STOP.
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_STOP) {
                player.pause()
                stop()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    // Position enregistrée toutes les 10 s : alimente « Continuer à regarder » et la reprise.
    LaunchedEffect(session) {
        while (true) {
            delay(10_000)
            if (player.isPlaying) saveProgress()
        }
    }

    LaunchedEffect(controlsVisible, session) {
        while (controlsVisible) {
            positionMs = (absolute() * 1000).toLong()
            delay(500)
        }
    }

    // Le focus attend que le panneau soit attaché, sinon les touches restent au conteneur.
    LaunchedEffect(controlsVisible) {
        if (controlsVisible) {
            delay(60)
            runCatching { seekFocus.requestFocus() }
        } else {
            trackMenu = null
            runCatching { rootFocus.requestFocus() }
        }
    }
    // Masquage auto après 6 s, sauf en pause (le panneau reste, comme sur Plex).
    LaunchedEffect(controlsVisible, interaction, trackMenu, isPlaying) {
        if (controlsVisible && trackMenu == null && isPlaying) {
            delay(6_000)
            controlsVisible = false
        }
    }
    // Épisode suivant : compte à rebours annulable.
    LaunchedEffect(countdown, next) {
        val c = countdown ?: return@LaunchedEffect
        val n = next ?: return@LaunchedEffect
        if (c <= 0) {
            vm.replace(Screen.Player(n.id))
            return@LaunchedEffect
        }
        delay(1_000)
        countdown = c - 1
    }

    androidx.activity.compose.BackHandler {
        when {
            countdown != null -> countdown = null
            trackMenu != null -> trackMenu = null
            controlsVisible -> controlsVisible = false
            else -> stop()
        }
    }

    val s = session
    val audioChoices = when {
        s == null -> emptyList()
        // En HLS, le NAS n'envoie qu'une piste audio : en changer = nouvelle session.
        s.play.mode != "direct" -> s.play.audio.map { a ->
            TrackChoice(audioLabel(a.language, a.codec, a.channels, a.title), SERVER_AUDIO, a.index, a.stream == s.play.audioStream)
        }
        else -> tracks?.let { trackChoices(it, C.TRACK_TYPE_AUDIO) }.orEmpty()
    }
    val textChoices = tracks?.let { trackChoices(it, C.TRACK_TYPE_TEXT) }.orEmpty()

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black)
            .focusRequester(rootFocus)
            .onPreviewKeyEvent { event ->
                if (event.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
                if (controlsVisible && event.key != Key.MediaStop && event.key != Key.MediaPlayPause) return@onPreviewKeyEvent false
                if (countdown != null || (ended && next != null)) return@onPreviewKeyEvent false
                when (event.key) {
                    Key.Enter, Key.DirectionCenter -> { showControls(); true }
                    Key.MediaPlayPause -> { if (player.isPlaying) player.pause() else player.play(); showControls(); true }
                    Key.MediaPlay -> { player.play(); showControls(); true }
                    Key.MediaPause -> { player.pause(); showControls(); true }
                    Key.MediaStop -> { stop(); true }
                    Key.MediaRewind, Key.DirectionLeft -> { seekTo(current() - REWIND_MS / 1000.0); showControls(); true }
                    Key.MediaFastForward, Key.DirectionRight -> { seekTo(current() + FORWARD_MS / 1000.0); showControls(); true }
                    Key.DirectionUp, Key.DirectionDown -> { showControls(); true }
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
                    keepScreenOn = true // sinon la veille de la box coupe le film
                    layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
                }
            },
            modifier = Modifier.fillMaxSize(),
        )

        if (s == null && error == null) {
            Text("Préparation de la lecture…", color = Aurora.Mist, fontSize = 18.sp, modifier = Modifier.align(Alignment.Center))
        }

        if (controlsVisible && error == null && s != null) {
            ControlsPanel(
                title = s.title,
                isPlaying = isPlaying,
                positionMs = positionMs,
                durationMs = (s.play.duration * 1000).toLong(),
                audioChoices = audioChoices,
                textChoices = textChoices,
                trackMenu = trackMenu,
                seekFocus = seekFocus,
                playFocus = playFocus,
                onSeek = { delta -> seekTo(current() + delta / 1000.0); positionMs = (current() * 1000).toLong(); interaction++ },
                onPlayPause = { if (player.isPlaying) player.pause() else player.play(); interaction++ },
                onStop = ::stop,
                onOpenMenu = { menu -> trackMenu = menu },
                onCloseMenu = { trackMenu = null },
                onChooseTrack = { type, choice ->
                    if (choice.groupIndex == SERVER_AUDIO) startAt(absolute(), audioIndex = choice.trackIndex)
                    else applyTrackChoice(player, type, choice)
                    trackMenu = null
                    interaction++
                },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }

        // Fin d'épisode : épisode suivant (compte à rebours si l'enchaînement est activé).
        val n = next
        if (ended && n != null && error == null) {
            NextEpisodeCard(
                card = n, countdown = countdown,
                onPlay = { vm.replace(Screen.Player(n.id)) },
                onCancel = { countdown = null; stop() },
                modifier = Modifier.align(Alignment.BottomEnd).padding(48.dp),
            )
        }

        error?.let { message ->
            ErrorPanel(
                message = message,
                onRetry = { triedRemux = false; triedTranscode = false; networkRetried = false; forcedMode = null; startAt(if (s != null) absolute() else -1.0) },
                onVlc = {
                    scope.launch {
                        val resp = runCatching {
                            client.play(PlayRequest(fileId = s?.play?.fileId ?: 0, caps = vm.caps, start = 0.0, external = true))
                        }.getOrNull()
                        // Le NAS a pu ouvrir une session HLS (remux/conversion) : inutile pour VLC.
                        resp?.sessionId?.let { id -> client.stopSession(id) }
                        resp?.signedUrl?.let { openInVlc(context, client.absolute(it)!!, s?.title.orEmpty(), absolute()) }
                    }
                },
                vlcAvailable = s != null,
                onBack = { stop() },
            )
        }
    }
}

private const val SERVER_AUDIO = -2

/** Code ISO 639-2 du profil AURORA (« fre », « ger »…) → code à 2 lettres d'ExoPlayer ; inconnu = aucune préférence. */
private fun iso2(code: String): String? = when (code.lowercase()) {
    "fre", "fra", "fr" -> "fr"
    "eng", "en" -> "en"
    "ger", "deu", "de" -> "de"
    "spa", "es" -> "es"
    "ita", "it" -> "it"
    "jpn", "ja" -> "ja"
    "por", "pt" -> "pt"
    "dut", "nld", "nl" -> "nl"
    "kor", "ko" -> "ko"
    "chi", "zho", "zh" -> "zh"
    "rus", "ru" -> "ru"
    "ara", "ar" -> "ar"
    else -> null
}

/** Nom lisible d'une langue ISO 639 (« fr », « fre », « eng »…) ; « und » = non renseignée. */
private fun languageName(code: String): String = when (code.lowercase()) {
    "", "und" -> "Langue inconnue"
    "fre", "fra", "fr" -> "Français"
    "eng", "en" -> "Anglais"
    else -> java.util.Locale.forLanguageTag(code).getDisplayLanguage(java.util.Locale.FRENCH)
        .takeIf { it.isNotBlank() && !it.equals(code, ignoreCase = true) }
        ?.replaceFirstChar { it.uppercase() } ?: code.uppercase()
}

private fun audioLabel(lang: String, codec: String, channels: Int, title: String): String {
    val l = languageName(lang)
    val ch = when (channels) { 1 -> "mono"; 2 -> "stéréo"; 6 -> "5.1"; 8 -> "7.1"; else -> "" }
    return listOf(l, codec.uppercase(), ch, title).filter { it.isNotBlank() }.joinToString(" · ")
}

/** Source à lire, avec les sous-titres fournis par le NAS (externes, ou tous en HLS). */
@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
private fun buildMediaItem(client: AuroraClient, play: PlayResponse): MediaItem {
    val subtitles = play.subtitles
        .filter { it.text && it.url != null && (play.mode != "direct" || it.external) }
        .map { sub ->
            MediaItem.SubtitleConfiguration.Builder(Uri.parse(client.absolute(sub.url)))
                .setMimeType(MimeTypes.TEXT_VTT)
                .setLanguage(sub.language.ifEmpty { null })
                .setLabel(sub.label)
                .setSelectionFlags(if (sub.forced) C.SELECTION_FLAG_FORCED else 0)
                .build()
        }
    return MediaItem.Builder()
        .setUri(client.absolute(play.url))
        .apply {
            if (play.mode != "direct") {
                setMimeType(MimeTypes.APPLICATION_M3U8)
                // Pas de rattrapage du « direct » (vitesse jusqu'à 1,03×) : c'est un film, pas une chaîne.
                setLiveConfiguration(MediaItem.LiveConfiguration.Builder().setMinPlaybackSpeed(1f).setMaxPlaybackSpeed(1f).build())
            }
        }
        .setSubtitleConfigurations(subtitles)
        .build()
}

@Composable
private fun NextEpisodeCard(card: Card, countdown: Int?, onPlay: () -> Unit, onCancel: () -> Unit, modifier: Modifier) {
    val focus = remember { FocusRequester() }
    LaunchedEffect(Unit) { delay(60); runCatching { focus.requestFocus() } }
    Column(
        modifier.width(420.dp).background(Color(0xF215181C), RoundedCornerShape(16.dp)).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Text(if (countdown != null) "ÉPISODE SUIVANT DANS $countdown S" else "ÉPISODE SUIVANT", color = Aurora.Ash, fontSize = 13.sp, letterSpacing = 2.sp)
        Text(episodeCode(card.season, card.episode, card.episodeEnd), color = Aurora.Accent, fontSize = 14.sp)
        Text(card.title, color = Aurora.Paper, fontSize = 20.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Button(onClick = onPlay, modifier = Modifier.focusRequester(focus)) { Text("Lire") }
            Button(onClick = onCancel) { Text(if (countdown != null) "Annuler" else "Retour") }
        }
    }
}

@Composable
private fun ErrorPanel(message: String, onRetry: () -> Unit, onVlc: () -> Unit, vlcAvailable: Boolean, onBack: () -> Unit) {
    val focus = remember { FocusRequester() }
    LaunchedEffect(Unit) { delay(60); runCatching { focus.requestFocus() } }
    Column(
        Modifier.fillMaxSize().background(Color(0xEB000000)).padding(64.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text("Lecture impossible", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 34.sp)
        Text(message, color = Aurora.Mist, fontSize = 17.sp, modifier = Modifier.padding(top = 12.dp).width(820.dp))
        Row(Modifier.padding(top = 28.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            Button(onClick = onRetry, modifier = Modifier.focusRequester(focus)) { Text("Réessayer") }
            if (vlcAvailable) Button(onClick = onVlc) { Text("Ouvrir dans VLC") }
            Button(onClick = onBack) { Text("Retour") }
        }
    }
}

/** Repli VLC avec un lien signé (VLC ne transmet pas la session AURORA). */
private fun openInVlc(context: android.content.Context, url: String, title: String, positionSec: Double) {
    val intent = Intent(Intent.ACTION_VIEW).apply {
        setDataAndTypeAndNormalize(Uri.parse(url), "video/*")
        putExtra("title", title)
        if (positionSec > 0) putExtra("position", (positionSec * 1000).toLong())
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }
    val vlc = Intent(intent).setPackage("org.videolan.vlc")
    runCatching { context.startActivity(vlc) }
        .recoverCatching { context.startActivity(Intent.createChooser(intent, title)) }
}


@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
@Composable
private fun ControlsPanel(
    title: String,
    isPlaying: Boolean,
    positionMs: Long,
    durationMs: Long,
    audioChoices: List<TrackChoice>,
    textChoices: List<TrackChoice>,
    trackMenu: TrackMenu?,
    seekFocus: FocusRequester,
    playFocus: FocusRequester,
    onSeek: (Long) -> Unit,
    onPlayPause: () -> Unit,
    onStop: () -> Unit,
    onOpenMenu: (TrackMenu) -> Unit,
    onCloseMenu: () -> Unit,
    onChooseTrack: (Int, TrackChoice) -> Unit,
    modifier: Modifier = Modifier,
) {

    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(Color(0xF2070A10))
            .padding(horizontal = 48.dp, vertical = 20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (trackMenu != null) {
            TrackMenuPanel(
                menu = trackMenu,
                choices = if (trackMenu == TrackMenu.AUDIO) audioChoices else textChoices,
                onChoose = { choice ->
                    onChooseTrack(
                        if (trackMenu == TrackMenu.AUDIO) C.TRACK_TYPE_AUDIO else C.TRACK_TYPE_TEXT,
                        choice,
                    )
                },
                onClose = onCloseMenu,
            )
            return@Column
        }

        Text(
            text = title,
            color = Aurora.Paper,
            fontSize = 20.sp,
            fontWeight = FontWeight.Bold,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )

        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = formatTime(positionMs),
                color = Aurora.Paper,
                fontSize = 16.sp,
                modifier = Modifier.width(80.dp),
            )
            SeekBar(
                positionMs = positionMs,
                durationMs = durationMs,
                onSeek = onSeek,
                onLeave = { runCatching { playFocus.requestFocus() } },
                modifier = Modifier.weight(1f).focusRequester(seekFocus),
            )
            Text(
                text = formatTime(durationMs),
                color = Aurora.Paper,
                fontSize = 16.sp,
                modifier = Modifier.width(80.dp).padding(start = 12.dp),
            )
        }

        // Rangée centrée, comme sur Plex : saut arrière, lecture/pause, saut avant.
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            RoundButton(glyph = "↺", caption = "10 s", onClick = { onSeek(-REWIND_MS) })
            Spacer(Modifier.width(18.dp))
            RoundButton(
                glyph = if (isPlaying) "❚❚" else "▶",
                caption = if (isPlaying) "Pause" else "Lecture",
                big = true,
                onClick = onPlayPause,
                modifier = Modifier.focusRequester(playFocus),
            )
            Spacer(Modifier.width(18.dp))
            RoundButton(glyph = "↻", caption = "30 s", onClick = { onSeek(FORWARD_MS) })

            Spacer(Modifier.width(48.dp))

            // Un seul choix possible = rien à choisir : le bouton reste visible pour
            // renseigner, mais grisé et non focusable.
            RoundButton(
                glyph = "💬",
                caption = subtitleCaption(textChoices),
                enabled = textChoices.size > 1,
                onClick = { onOpenMenu(TrackMenu.TEXT) },
            )
            Spacer(Modifier.width(18.dp))
            RoundButton(
                glyph = "🔊",
                caption = audioChoices.firstOrNull { it.selected }?.label ?: "Audio",
                enabled = audioChoices.size > 1,
                onClick = { onOpenMenu(TrackMenu.AUDIO) },
            )
            Spacer(Modifier.width(18.dp))
            RoundButton(glyph = "■", caption = "Arrêter", onClick = onStop)
        }
    }
}

@Composable
private fun TrackMenuPanel(
    menu: TrackMenu,
    choices: List<TrackChoice>,
    onChoose: (TrackChoice) -> Unit,
    onClose: () -> Unit,
) {
    val first = remember(menu) { FocusRequester() }
    LaunchedEffect(menu) {
        delay(60)
        runCatching { first.requestFocus() }
    }

    Text(
        text = if (menu == TrackMenu.AUDIO) "Piste audio" else "Sous-titres",
        color = Aurora.Paper,
        fontSize = 20.sp,
        fontWeight = FontWeight.Bold,
    )

    if (choices.isEmpty()) {
        Text(
            text = "Aucune piste disponible pour ce fichier.",
            color = Aurora.Paper,
            fontSize = 16.sp,
        )
        return
    }

    LazyColumn(
        modifier = Modifier.height(220.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        items(choices) { choice ->
            val isFirst = choice == choices.first()
            MenuRow(
                label = choice.label,
                current = choice.selected,
                onSelect = { onChoose(choice) },
                onClose = onClose,
                modifier = if (isFirst) Modifier.focusRequester(first) else Modifier,
            )
        }
    }
}

@Composable
private fun MenuRow(
    label: String,
    current: Boolean,
    onSelect: () -> Unit,
    onClose: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var focused by remember { mutableStateOf(false) }

    Box(
        modifier = modifier
            .fillMaxWidth()
            .onFocusChanged { focused = it.isFocused }
            .onPreviewKeyEvent { event ->
                if (event.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
                when (event.key) {
                    Key.Enter, Key.DirectionCenter -> { onSelect(); true }
                    Key.DirectionLeft -> { onClose(); true }
                    else -> false
                }
            }
            .background(
                color = if (focused) Aurora.Paper else Color(0x22FFFFFF),
                shape = RoundedCornerShape(6.dp),
            )
            .padding(horizontal = 16.dp, vertical = 10.dp)
            .focusable(),
    ) {
        Text(
            text = if (current) "•  $label" else "    $label",
            color = if (focused) Aurora.Ink else Aurora.Paper,
            fontSize = 16.sp,
            fontWeight = if (current) FontWeight.Bold else FontWeight.Normal,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * Barre de progression focusable : c'est elle qui reçoit ◀/▶ pour sauter, ce qui permet
 * d'enchaîner les sauts. Les flèches ne peuvent pas servir à la fois au saut et à la
 * navigation entre boutons — il fallait leur donner un propriétaire clair.
 */
@Composable
private fun SeekBar(
    positionMs: Long,
    durationMs: Long,
    onSeek: (Long) -> Unit,
    onLeave: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var focused by remember { mutableStateOf(false) }
    val fraction = if (durationMs > 0) {
        (positionMs.toFloat() / durationMs).coerceIn(0f, 1f)
    } else {
        0f
    }
    val thickness = if (focused) 14.dp else 8.dp

    Box(
        modifier = modifier
            .onFocusChanged { focused = it.isFocused }
            .onPreviewKeyEvent { event ->
                if (event.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
                when (event.key) {
                    Key.DirectionLeft -> { onSeek(-REWIND_MS); true }
                    Key.DirectionRight -> { onSeek(FORWARD_MS); true }
                    Key.DirectionDown -> { onLeave(); true }
                    else -> false
                }
            }
            .focusable()
            .padding(vertical = 8.dp),
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(thickness)
                .background(
                    if (focused) Color(0x66FFFFFF) else Color(0x33FFFFFF),
                    RoundedCornerShape(thickness / 2),
                ),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth(fraction)
                    .height(thickness)
                    .background(Aurora.Accent, RoundedCornerShape(thickness / 2)),
            )
        }
    }
}

@Composable
private fun RoundButton(
    glyph: String,
    caption: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    big: Boolean = false,
    enabled: Boolean = true,
) {
    var focused by remember { mutableStateOf(false) }
    val diameter = if (big) 72.dp else 56.dp

    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Box(
            modifier = modifier
                .size(diameter)
                .onFocusChanged { focused = it.isFocused }
                .onPreviewKeyEvent { event ->
                    if (event.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
                    if (event.key == Key.Enter || event.key == Key.DirectionCenter) {
                        onClick()
                        true
                    } else {
                        false
                    }
                }
                // Un bouton sans effet ne doit pas prendre le focus : sinon la navigation
                // s'y arrête et la touche OK n'y produit rien, ce qui donne l'impression
                // que la télécommande ne répond pas.
                .then(if (enabled) Modifier.focusable() else Modifier)
                .background(
                    color = when {
                        focused -> Aurora.Paper
                        else -> Color(0x33FFFFFF)
                    },
                    shape = CircleShape,
                )
                .border(
                    width = if (focused) 3.dp else 1.dp,
                    color = if (focused) Aurora.Paper else Color(0x55FFFFFF),
                    shape = CircleShape,
                ),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = glyph,
                color = when {
                    focused -> Aurora.Ink
                    enabled -> Aurora.Paper
                    else -> Color(0x66FFFFFF)
                },
                fontSize = if (big) 26.sp else 20.sp,
                fontWeight = FontWeight.Bold,
            )
        }
        Text(
            text = caption,
            color = if (enabled) Aurora.Paper else Color(0x66FFFFFF),
            fontSize = 13.sp,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(top = 4.dp).width(diameter + 28.dp),
        )
    }
}

// MARK: - Pistes

/**
 * Construit l'élément à lire, avec les sous-titres externes déclarés par Plex : le
 * lecteur ne découvre que les pistes présentes dans le conteneur, pas les `.srt` voisins.
 */
@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
private fun trackChoices(tracks: Tracks, type: Int): List<TrackChoice> {
    val choices = mutableListOf<TrackChoice>()
    if (type == C.TRACK_TYPE_TEXT) {
        val anySelected = tracks.groups
            .filter { it.type == C.TRACK_TYPE_TEXT }
            .any { group -> (0 until group.length).any { group.isTrackSelected(it) } }
        choices += TrackChoice("Aucun", NO_TRACK, NO_TRACK, selected = !anySelected)
    }

    tracks.groups.forEachIndexed { groupIndex, group ->
        if (group.type != type) return@forEachIndexed
        for (trackIndex in 0 until group.length) {
            if (!group.isTrackSupported(trackIndex)) continue
            choices += TrackChoice(
                label = describe(group.getTrackFormat(trackIndex)),
                groupIndex = groupIndex,
                trackIndex = trackIndex,
                selected = group.isTrackSelected(trackIndex),
            )
        }
    }
    return choices
}

@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
private fun applyTrackChoice(player: ExoPlayer, type: Int, choice: TrackChoice) {
    if (choice.groupIndex == NO_TRACK) {
        player.trackSelectionParameters = player.trackSelectionParameters.buildUpon()
            .setTrackTypeDisabled(type, true)
            .build()
        return
    }
    val group = player.currentTracks.groups.getOrNull(choice.groupIndex) ?: return
    player.trackSelectionParameters = player.trackSelectionParameters.buildUpon()
        .setTrackTypeDisabled(type, false)
        .setOverrideForType(TrackSelectionOverride(group.mediaTrackGroup, choice.trackIndex))
        .build()
}

private fun describe(format: Format): String {
    val name = format.label?.takeIf { it.isNotBlank() }
        ?: format.language?.takeIf { it.isNotBlank() }?.let { languageName(it) }
        ?: "Piste"
    val channels = format.channelCount.takeIf { it > 0 }?.let { " · $it ch" }.orEmpty()
    return name + channels
}

private fun subtitleCaption(choices: List<TrackChoice>): String {
    if (choices.size <= 1) return "Aucun sous-titre"
    val selected = choices.firstOrNull { it.selected } ?: return "Sous-titres"
    return if (selected.groupIndex == NO_TRACK) "Sous-titres" else selected.label
}

/**
 * Rattrapage de la piste audio française.
 *
 * `setPreferredAudioLanguages` suffit quand la piste est étiquetée `fr`. Beaucoup de
 * fichiers ne le sont pas : la VF y est en langue « und » et ne se reconnaît qu'à son
 * libellé. On ne force que si la piste retenue n'est pas déjà française.
 */
@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
private fun selectFrenchAudio(player: ExoPlayer, tracks: Tracks) {
    val audioGroups = tracks.groups.filter { it.type == C.TRACK_TYPE_AUDIO }
    if (audioGroups.isEmpty()) return

    val selectedIsFrench = audioGroups.any { group ->
        (0 until group.length).any { index ->
            group.isTrackSelected(index) && isFrench(group.getTrackFormat(index))
        }
    }
    if (selectedIsFrench) return

    for (group in audioGroups) {
        for (index in 0 until group.length) {
            if (!group.isTrackSupported(index)) continue
            if (!isFrench(group.getTrackFormat(index))) continue
            player.trackSelectionParameters = player.trackSelectionParameters
                .buildUpon()
                .setOverrideForType(TrackSelectionOverride(group.mediaTrackGroup, index))
                .build()
            return
        }
    }
}

private fun isFrench(format: Format): Boolean {
    val language = format.language?.lowercase()
    if (language != null && (language.take(2) == "fr" || language == "fra" || language == "fre")) {
        return true
    }
    val label = (format.label ?: "").lowercase()
    return FRENCH_LABELS.any { label.contains(it) }
}

private val FRENCH_LABELS = listOf("vff", "vfq", "vf", "french", "français", "francais")

/** Retour court, avance longue : on revient sur ce qu'on a raté, on saute une séquence. */
private const val REWIND_MS = 10_000L
private const val FORWARD_MS = 30_000L
private const val NO_TRACK = -1

private fun seekBy(player: ExoPlayer, deltaMs: Long) {
    val duration = player.duration
    var target = (player.currentPosition + deltaMs).coerceAtLeast(0)
    if (duration > 0) target = target.coerceAtMost(duration - 1_000)
    player.seekTo(target)
}

private fun formatTime(ms: Long): String {
    if (ms <= 0) return "0:00"
    val totalSeconds = ms / 1000
    val hours = totalSeconds / 3600
    val minutes = (totalSeconds % 3600) / 60
    val seconds = totalSeconds % 60
    return if (hours > 0) {
        "%d:%02d:%02d".format(hours, minutes, seconds)
    } else {
        "%d:%02d".format(minutes, seconds)
    }
}
