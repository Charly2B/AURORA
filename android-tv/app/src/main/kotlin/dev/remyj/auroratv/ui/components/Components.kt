package dev.remyj.auroratv.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import kotlinx.coroutines.delay
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.tv.material3.ClickableSurfaceDefaults
import androidx.tv.material3.Surface
import androidx.tv.material3.Text
import coil3.compose.AsyncImage
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.Card
import dev.remyj.auroratv.ui.theme.Aurora

private val CardShape = RoundedCornerShape(6.dp)

/**
 * Affiche (film, série). Les informations utiles de loin — vu, épisodes restants, progression —
 * sont posées SUR l'affiche, en pastilles : du petit texte sous la vignette est illisible à 3 m.
 */
/** Carte focalisée en dernier ([last]) et carte à refocaliser au retour sur l'écran ([restore]). */
class CardFocusMemory {
    var last: String? = null
    var restore: String? = null
}

val LocalCardFocus = staticCompositionLocalOf<CardFocusMemory?> { null }

/** Mémorise le focus de cette carte et le lui rend si l'écran demande une restauration. */
@Composable
private fun Modifier.rememberedFocus(key: String, requester: FocusRequester): Modifier {
    val memory = LocalCardFocus.current ?: return this
    LaunchedEffect(key) {
        if (memory.restore == key) {
            memory.restore = null
            delay(50)
            runCatching { requester.requestFocus() }
        }
    }
    return focusRequester(requester).onFocusChanged { if (it.isFocused) memory.last = key }
}

private val FrameShape = RoundedCornerShape(10.dp)
private val InnerShape = RoundedCornerShape(6.dp)
/** Cadre verre façon macOS (comme sur la TV Samsung) : verre clair en dégradé, liseré blanc, ombre douce. */
private val FrameGlass = Brush.linearGradient(listOf(Color(0x38FFFFFF), Color(0x0DFFFFFF), Color(0x14FFFFFF)))

@Composable
fun PosterCard(card: Card, client: AuroraClient, onClick: () -> Unit, modifier: Modifier = Modifier, width: Dp = 118.dp, focusKey: String = card.kind + card.id, onLongClick: (() -> Unit)? = null, autoFocus: Boolean = false, onAutoFocused: () -> Unit = {}) {
    // Pas de « ⋯ » sur une série : « vu » se règle par épisode, jamais pour toute la série d'un coup.
    // Une collection n'est pas une œuvre : son identifiant ne doit pas servir à écrire « vu ».
    val moreable = card.kind == "movie" || card.kind == "episode"
    var focused by remember { mutableStateOf(false) }
    val cardRequester = remember { FocusRequester() }
    var displayed by rememberWatchedCard(card)
    var menu by remember { mutableStateOf(false) }
    val longPress: (() -> Unit)? = onLongClick ?: if (moreable) ({ menu = true }) else null
    // Reprise de focus après un « Réessayer » ou un « Voir les N films » : la carte reprend la main,
    // puis le parent retire la demande (sinon une carte recomposée plus tard volerait le focus).
    LaunchedEffect(autoFocus) {
        if (!autoFocus) return@LaunchedEffect
        delay(60)
        runCatching { cardRequester.requestFocus() }
        onAutoFocused()
    }
    if (menu) WatchedMenu(displayed, client, onDismiss = { menu = false }, onChanged = { displayed = it })
    Column(modifier = modifier.width(width), verticalArrangement = Arrangement.spacedBy(7.dp)) {
        Box(Modifier.graphicsLayer { scaleX = if (focused) 1.045f else 1f; scaleY = scaleX }) {
            Surface(
                onClick = onClick,
                // Appui long = Vu / Pas vu, comme Plex : ◀/▶ restent aux affiches (un appui par carte).
                onLongClick = longPress,
                shape = ClickableSurfaceDefaults.shape(FrameShape),
                colors = ClickableSurfaceDefaults.colors(containerColor = Color.Transparent, focusedContainerColor = Color.Transparent),
                scale = ClickableSurfaceDefaults.scale(focusedScale = 1f),
                modifier = Modifier.fillMaxWidth()
                    .shadow(if (focused) 18.dp else 10.dp, FrameShape, clip = false,
                        ambientColor = if (focused) Aurora.Accent else Color.Black,
                        spotColor = if (focused) Aurora.Accent else Color.Black)
                    .then(if (focused) Modifier.border(7.dp, Aurora.Accent.copy(alpha = 0.32f), FrameShape) else Modifier)
                    .rememberedFocus(focusKey, cardRequester)
                    .onFocusChanged { focused = it.isFocused },
            ) {
                Box(Modifier.background(FrameGlass, FrameShape)
                    .border(if (focused) 4.dp else 1.dp, if (focused) Aurora.FocusGold else Aurora.Paper.copy(alpha = 0.34f), FrameShape)
                    .padding(5.dp)) {
                    Box(Modifier.fillMaxWidth().aspectRatio(2f / 3f).clip(InnerShape).background(Aurora.Coal)) {
                        Artwork(client.absolute(displayed.poster), client.absolute(displayed.thumb), displayed.title, displayed.year)
                        Badges(displayed, Modifier.align(Alignment.TopEnd).padding(5.dp))
                        if (displayed.progress > 0f && !displayed.watched) {
                            ProgressLine(displayed.progress, Modifier.align(Alignment.BottomCenter).padding(start = 6.dp, end = if (moreable) 36.dp else 6.dp, bottom = 6.dp))
                        }
                    }
                }
            }
            if (moreable) {
                // Repère visuel seulement : le D-pad ne s'y arrête plus (appui long sur l'affiche à la place).
                MoreButton(onClick = { menu = true }, modifier = Modifier.align(Alignment.BottomEnd).padding(9.dp)
                    .focusProperties { canFocus = false })
            }
        }
        Text(card.title, color = if (focused) Aurora.FocusGold else Aurora.Mist,
            fontFamily = Aurora.Display, fontSize = 14.sp, fontWeight = if (focused) FontWeight.SemiBold else FontWeight.Normal,
            textAlign = if (focused) TextAlign.Center else TextAlign.Start, maxLines = 1, overflow = TextOverflow.Ellipsis,
            modifier = Modifier.fillMaxWidth().background(if (focused) Aurora.Accent.copy(alpha = 0.18f) else Color.Transparent, RoundedCornerShape(4.dp))
                .padding(horizontal = 3.dp, vertical = 5.dp))
    }
}

/** Carte large 16:9 (reprise, épisodes) : vignette d'épisode ou fond du film. */
@Composable
fun WideCard(card: Card, client: AuroraClient, onClick: () -> Unit, modifier: Modifier = Modifier, width: Dp = 280.dp, focusKey: String = card.kind + card.id, onLongClick: (() -> Unit)? = null, autoFocus: Boolean = false, onAutoFocused: () -> Unit = {}) {
    // Même règle que les affiches : « ⋯ » sur un film ou un épisode, jamais sur une série ni une collection.
    val moreable = card.kind == "movie" || card.kind == "episode"
    var focused by remember { mutableStateOf(false) }
    val cardRequester = remember { FocusRequester() }
    var displayed by rememberWatchedCard(card)
    var menu by remember { mutableStateOf(false) }
    val longPress: (() -> Unit)? = onLongClick ?: if (moreable) ({ menu = true }) else null
    LaunchedEffect(autoFocus) {
        if (!autoFocus) return@LaunchedEffect
        delay(60)
        runCatching { cardRequester.requestFocus() }
        onAutoFocused()
    }
    if (menu) WatchedMenu(displayed, client, onDismiss = { menu = false }, onChanged = { displayed = it })
    val image = if (card.kind == "episode") card.thumb else (card.backdrop ?: card.thumb)
    Column(modifier = modifier.width(width), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Box {
        Surface(
            onClick = onClick,
            onLongClick = longPress,
            shape = ClickableSurfaceDefaults.shape(CardShape),
            colors = ClickableSurfaceDefaults.colors(containerColor = Aurora.Coal, focusedContainerColor = Aurora.Coal),
            scale = ClickableSurfaceDefaults.scale(focusedScale = 1.05f),
            modifier = Modifier.fillMaxWidth().aspectRatio(16f / 9f).rememberedFocus(focusKey, cardRequester)
                .onFocusChanged { focused = it.isFocused },
        ) {
            Box(Modifier.fillMaxSize()) {
                AsyncImage(
                    model = client.absolute(image),
                    contentDescription = card.title,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize(),
                )
                Box(Modifier.fillMaxSize().border(if (focused) 3.dp else 1.dp, if (focused) Aurora.Accent else Color(0x22FFFFFF), CardShape))
                if (displayed.watched) WatchedBadge(Modifier.align(Alignment.TopEnd).padding(6.dp))
                if (displayed.progress > 0f && !displayed.watched) {
                    ProgressLine(displayed.progress, Modifier.align(Alignment.BottomCenter).padding(start = 8.dp, end = if (moreable) 42.dp else 8.dp, bottom = 8.dp))
                }
            }
        }
        if (moreable) {
            MoreButton(onClick = { menu = true }, modifier = Modifier.align(Alignment.BottomEnd).padding(8.dp)
                .focusProperties { canFocus = false })
        }
        }
        if (card.kind == "episode") {
            Text(
                "${card.seriesTitle.orEmpty()} · ${episodeCode(card.season, card.episode, card.episodeEnd)}",
                color = Aurora.Accent, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis,
            )
        }
        Text(card.title, color = if (focused) Aurora.Paper else Aurora.Mist, fontFamily = Aurora.Display, fontSize = 15.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

/** Image d'affiche avec repli composé (image tirée de la vidéo + titre), comme sur le web. */
@Composable
fun Artwork(poster: String?, thumb: String?, title: String, year: Int?) {
    if (poster != null) {
        AsyncImage(model = poster, contentDescription = title, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
        return
    }
    Box(
        Modifier
            .fillMaxSize()
            .background(Brush.verticalGradient(listOf(Color(0xFF191C21), Color(0xFF0C0D10)))),
    ) {
        if (thumb != null) {
            AsyncImage(
                model = thumb, contentDescription = null, contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxWidth().fillMaxHeight(0.56f),
            )
            Box(
                Modifier.fillMaxWidth().fillMaxHeight(0.56f)
                    .background(Brush.verticalGradient(listOf(Color.Transparent, Color(0xFF131519)))),
            )
        }
        Column(Modifier.align(Alignment.BottomStart).padding(12.dp)) {
            Box(Modifier.width(28.dp).height(1.dp).background(Aurora.Accent))
            Text(
                title, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 15.sp, lineHeight = 17.sp,
                maxLines = 3, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 8.dp),
            )
            if (year != null) Text("$year", color = Aurora.Ash, fontSize = 11.sp, modifier = Modifier.padding(top = 4.dp))
        }
    }
}

@Composable
private fun Badges(card: Card, modifier: Modifier) {
    val unwatched = card.unwatched
    when {
        card.kind == "series" && unwatched != null && unwatched > 0 && unwatched < (card.total ?: 0) ->
            Box(
                modifier.background(Aurora.Accent, CircleShape).padding(horizontal = 8.dp, vertical = 2.dp),
            ) { Text("$unwatched", color = Aurora.Ink, fontSize = 13.sp, fontWeight = FontWeight.Bold) }
        (if (card.kind == "series") unwatched == 0 && (card.total ?: 0) > 0 else card.watched) -> WatchedBadge(modifier)
    }
}

@Composable
fun WatchedBadge(modifier: Modifier = Modifier) {
    // Coche « vu » verte (comme sur Mac et Samsung) ; rien du tout quand ce n'est pas vu.
    Box(modifier.size(22.dp).shadow(2.dp, CircleShape).background(Aurora.Seen, CircleShape), contentAlignment = Alignment.Center) {
        Text("✓", color = Color.White, fontSize = 12.sp, fontWeight = FontWeight.Bold)
    }
}

@Composable
fun ProgressLine(value: Float, modifier: Modifier = Modifier) {
    Box(modifier.fillMaxWidth().height(4.dp).clip(CircleShape).background(Color(0x40FFFFFF))) {
        Box(Modifier.fillMaxWidth(value.coerceIn(0f, 1f)).fillMaxHeight().background(Aurora.Accent))
    }
}

/** Rangée horizontale titrée. */
@Composable
fun Rail(title: String, cards: List<Card>, client: AuroraClient, wide: Boolean, onOpen: (Card) -> Unit, onLongPress: ((Card) -> Unit)? = null, autoFocusFirst: Boolean = false, onAutoFocused: () -> Unit = {}) {
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(title, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 22.sp, modifier = Modifier.padding(start = 32.dp))
        LazyRow(
            contentPadding = PaddingValues(horizontal = 32.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(22.dp),
        ) {
            items(cards, key = { it.kind + it.id }) { c ->
                // Clé préfixée par la rangée : une même œuvre peut figurer dans plusieurs rangées.
                val key = "$title/${c.kind}${c.id}"
                val long = onLongPress?.let { { it(c) } }
                val auto = autoFocusFirst && c == cards.first()
                if (wide) WideCard(c, client, onClick = { onOpen(c) }, focusKey = key, onLongClick = long, autoFocus = auto, onAutoFocused = onAutoFocused)
                else PosterCard(c, client, onClick = { onOpen(c) }, focusKey = key, onLongClick = long, autoFocus = auto, onAutoFocused = onAutoFocused)
            }
        }
    }
}

/**
 * Segment sélectionnable (dossiers, saisons, tris), à poser dans une [SegmentBar] comme sur Mac :
 * choisi = fond or translucide, focus = or plein.
 */
@Composable
fun Chip(label: String, selected: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier, count: Int? = null) {
    Surface(
        onClick = onClick,
        shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(6.dp)),
        colors = ClickableSurfaceDefaults.colors(
            containerColor = if (selected) Aurora.Accent.copy(alpha = 0.24f) else Color.Transparent,
            contentColor = if (selected) Aurora.Paper else Aurora.Mist,
            focusedContainerColor = Aurora.Accent,
            focusedContentColor = Aurora.Ink,
        ),
        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.04f),
        modifier = modifier,
    ) {
        Row(Modifier.padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Text(label, fontFamily = Aurora.Display, fontSize = 17.sp, maxLines = 1, softWrap = false)
            if (count != null) Text("  $count", fontSize = 13.sp)
        }
    }
}

/** Barre sombre au filet doré qui regroupe des [Chip] : lisible même sur un fond d'affiche clair. */
@Composable
fun SegmentBar(modifier: Modifier = Modifier, content: @Composable () -> Unit) {
    Row(
        modifier
            .background(Aurora.Night.copy(alpha = 0.9f), RoundedCornerShape(9.dp))
            .border(1.dp, Aurora.Accent.copy(alpha = 0.24f), RoundedCornerShape(9.dp))
            .padding(3.dp),
        horizontalArrangement = Arrangement.spacedBy(2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) { content() }
}

/** Petit titre en capitales espacées (« À LA UNE · FILM », « PARCOURIR »). */
@Composable
fun Eyebrow(text: String, modifier: Modifier = Modifier, color: Color = Aurora.Accent) {
    Text(text.uppercase(), color = color, fontSize = 13.sp, fontWeight = FontWeight.SemiBold, letterSpacing = 3.sp, modifier = modifier)
}

/** Bouton d'action (Lecture, Reprendre…). Le principal est plein et clair. */
@Composable
fun ActionButton(label: String, onClick: () -> Unit, modifier: Modifier = Modifier, primary: Boolean = false) {
    Surface(
        onClick = onClick,
        shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(6.dp)),
        colors = ClickableSurfaceDefaults.colors(
            containerColor = if (primary) Aurora.Cream else Aurora.Accent.copy(alpha = 0.14f),
            contentColor = if (primary) Aurora.Ink else Aurora.Paper,
            focusedContainerColor = Aurora.Accent,
            focusedContentColor = Aurora.Ink,
        ),
        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.06f),
        modifier = modifier,
    ) {
        Text(label, fontSize = 17.sp, fontWeight = FontWeight.SemiBold, maxLines = 1, softWrap = false, modifier = Modifier.padding(horizontal = 26.dp, vertical = 12.dp))
    }
}

fun episodeCode(season: Int?, episode: Int?, end: Int? = null): String {
    if (season == null || episode == null) return ""
    val e = "E%02d".format(episode) + (end?.let { "-E%02d".format(it) } ?: "")
    return if (season == 0) "Spécial $e" else "S%02d $e".format(season)
}

fun clock(seconds: Double): String {
    val s = seconds.toLong().coerceAtLeast(0)
    val h = s / 3600
    val m = (s % 3600) / 60
    val sec = s % 60
    return if (h > 0) "%d:%02d:%02d".format(h, m, sec) else "%d:%02d".format(m, sec)
}

fun runtimeLabel(minutes: Int?, seconds: Double?): String {
    val m = minutes ?: seconds?.let { (it / 60).toInt() } ?: 0
    if (m <= 0) return ""
    return if (m >= 60) "${m / 60} h %02d".format(m % 60) else "$m min"
}
