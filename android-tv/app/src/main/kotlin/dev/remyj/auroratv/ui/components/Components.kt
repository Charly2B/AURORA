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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import kotlinx.coroutines.delay
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
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

private val CardShape = RoundedCornerShape(10.dp)

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
private fun Modifier.rememberedFocus(key: String): Modifier {
    val memory = LocalCardFocus.current ?: return this
    val requester = remember { FocusRequester() }
    LaunchedEffect(key) {
        if (memory.restore == key) {
            memory.restore = null
            delay(50)
            runCatching { requester.requestFocus() }
        }
    }
    return focusRequester(requester).onFocusChanged { if (it.isFocused) memory.last = key }
}

@Composable
fun PosterCard(card: Card, client: AuroraClient, onClick: () -> Unit, modifier: Modifier = Modifier, width: Dp = 150.dp, focusKey: String = card.kind + card.id, onLongClick: (() -> Unit)? = null) {
    var focused by remember { mutableStateOf(false) }
    Column(modifier = modifier.width(width), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Surface(
            onClick = onClick,
            onLongClick = onLongClick,
            shape = ClickableSurfaceDefaults.shape(CardShape),
            colors = ClickableSurfaceDefaults.colors(containerColor = Aurora.Coal, focusedContainerColor = Aurora.Coal),
            scale = ClickableSurfaceDefaults.scale(focusedScale = 1.06f),
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(2f / 3f)
                .rememberedFocus(focusKey)
                .onFocusChanged { focused = it.isFocused },
        ) {
            Box(Modifier.fillMaxSize()) {
                Artwork(client.absolute(card.poster), client.absolute(card.thumb), card.title, card.year)
                Box(
                    Modifier
                        .fillMaxSize()
                        .border(if (focused) 3.dp else 1.dp, if (focused) Aurora.Accent else Color(0x22FFFFFF), CardShape),
                )
                Badges(card, Modifier.align(Alignment.TopEnd).padding(6.dp))
                if (card.progress > 0f) ProgressLine(card.progress, Modifier.align(Alignment.BottomCenter).padding(8.dp))
            }
        }
        Text(
            card.title,
            color = if (focused) Aurora.Paper else Aurora.Mist,
            fontSize = 14.sp,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/** Carte large 16:9 (reprise, épisodes) : vignette d'épisode ou fond du film. */
@Composable
fun WideCard(card: Card, client: AuroraClient, onClick: () -> Unit, modifier: Modifier = Modifier, width: Dp = 280.dp, focusKey: String = card.kind + card.id, onLongClick: (() -> Unit)? = null) {
    var focused by remember { mutableStateOf(false) }
    val image = if (card.kind == "episode") card.thumb else (card.backdrop ?: card.thumb)
    Column(modifier = modifier.width(width), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Surface(
            onClick = onClick,
            onLongClick = onLongClick,
            shape = ClickableSurfaceDefaults.shape(CardShape),
            colors = ClickableSurfaceDefaults.colors(containerColor = Aurora.Coal, focusedContainerColor = Aurora.Coal),
            scale = ClickableSurfaceDefaults.scale(focusedScale = 1.05f),
            modifier = Modifier.fillMaxWidth().aspectRatio(16f / 9f).rememberedFocus(focusKey).onFocusChanged { focused = it.isFocused },
        ) {
            Box(Modifier.fillMaxSize()) {
                AsyncImage(
                    model = client.absolute(image),
                    contentDescription = card.title,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize(),
                )
                Box(Modifier.fillMaxSize().border(if (focused) 3.dp else 1.dp, if (focused) Aurora.Accent else Color(0x22FFFFFF), CardShape))
                if (card.watched) WatchedBadge(Modifier.align(Alignment.TopEnd).padding(6.dp))
                if (card.progress > 0f) ProgressLine(card.progress, Modifier.align(Alignment.BottomCenter).padding(8.dp))
            }
        }
        if (card.kind == "episode") {
            Text(
                "${card.seriesTitle.orEmpty()} · ${episodeCode(card.season, card.episode, card.episodeEnd)}",
                color = Aurora.Accent, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis,
            )
        }
        Text(card.title, color = if (focused) Aurora.Paper else Aurora.Mist, fontSize = 14.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
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
        card.watched || (card.kind == "series" && unwatched == 0 && (card.total ?: 0) > 0) -> WatchedBadge(modifier)
    }
}

@Composable
fun WatchedBadge(modifier: Modifier = Modifier) {
    Box(modifier.size(24.dp).background(Color(0xCC08090B), CircleShape), contentAlignment = Alignment.Center) {
        Text("✓", color = Aurora.Accent, fontSize = 13.sp, fontWeight = FontWeight.Bold)
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
fun Rail(title: String, cards: List<Card>, client: AuroraClient, wide: Boolean, onOpen: (Card) -> Unit, onLongPress: ((Card) -> Unit)? = null) {
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(title, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 24.sp, modifier = Modifier.padding(start = 48.dp))
        LazyRow(
            contentPadding = PaddingValues(horizontal = 48.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(18.dp),
        ) {
            items(cards, key = { it.kind + it.id }) { c ->
                // Clé préfixée par la rangée : une même œuvre peut figurer dans plusieurs rangées.
                val key = "$title/${c.kind}${c.id}"
                val long = onLongPress?.let { { it(c) } }
                if (wide) WideCard(c, client, onClick = { onOpen(c) }, focusKey = key, onLongClick = long)
                else PosterCard(c, client, onClick = { onOpen(c) }, focusKey = key, onLongClick = long)
            }
        }
    }
}

/** Puce sélectionnable (onglets, dossiers, saisons, tris). */
@Composable
fun Chip(label: String, selected: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier, count: Int? = null) {
    Surface(
        onClick = onClick,
        shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(50)),
        colors = ClickableSurfaceDefaults.colors(
            containerColor = if (selected) Aurora.Paper else Color(0x14FFFFFF),
            contentColor = if (selected) Aurora.Ink else Aurora.Mist,
            focusedContainerColor = Aurora.Accent,
            focusedContentColor = Aurora.Ink,
        ),
        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.04f),
        modifier = modifier,
    ) {
        Row(Modifier.padding(horizontal = 18.dp, vertical = 9.dp), verticalAlignment = Alignment.CenterVertically) {
            Text(label, fontSize = 15.sp, fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal)
            if (count != null) Text("  $count", fontSize = 13.sp)
        }
    }
}

/** Bouton d'action (Lecture, Reprendre…). Le principal est plein et clair. */
@Composable
fun ActionButton(label: String, onClick: () -> Unit, modifier: Modifier = Modifier, primary: Boolean = false) {
    Surface(
        onClick = onClick,
        shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(50)),
        colors = ClickableSurfaceDefaults.colors(
            containerColor = if (primary) Aurora.Paper else Color(0x1FFFFFFF),
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
