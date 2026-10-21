package dev.remyj.auroratv.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.tv.material3.ClickableSurfaceDefaults
import androidx.tv.material3.Surface
import androidx.tv.material3.Text
import coil3.compose.AsyncImage
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.Card
import dev.remyj.auroratv.net.Detail
import dev.remyj.auroratv.ui.components.ActionButton
import dev.remyj.auroratv.ui.components.Artwork
import dev.remyj.auroratv.ui.components.Chip
import dev.remyj.auroratv.ui.components.ProgressLine
import dev.remyj.auroratv.ui.components.WatchedBadge
import dev.remyj.auroratv.ui.components.clock
import dev.remyj.auroratv.ui.components.episodeCode
import dev.remyj.auroratv.ui.components.runtimeLabel
import dev.remyj.auroratv.ui.theme.Aurora
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun DetailScreen(vm: AppViewModel, screen: Screen.Detail) {
    val client = vm.client ?: return
    var reload by remember { mutableIntStateOf(0) }
    val state by produceState<Load<Detail>>(Load.Loading, screen.itemId, reload, vm.refresh) {
        value = load { client.item(screen.itemId) }
    }
    when (val s = state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> Message(s.message)
        is Load.Ok -> DetailContent(vm, client, s.value, onChanged = { reload++ })
    }
}

@Composable
private fun DetailContent(vm: AppViewModel, client: AuroraClient, d: Detail, onChanged: () -> Unit) {
    val scope = rememberCoroutineScope()
    val firstButton = remember { FocusRequester() }
    LaunchedEffect(d.id) { delay(80); runCatching { firstButton.requestFocus() } }
    val isSeries = d.kind == "series"
    val file = d.files.firstOrNull { it.id == d.state.lastFileId } ?: d.files.firstOrNull()
    val length = file?.duration?.takeIf { it > 0 } ?: ((d.runtime ?: 0) * 60.0)
    val resumable = !isSeries && d.state.position > 0 && !d.state.watched
    var season by remember(d.id) { mutableStateOf(d.nextUp?.season ?: d.seasons?.firstOrNull()?.number) }
    val toggle: (suspend () -> Unit) -> Unit = { action -> scope.launch { runCatching { action() }; onChanged() } }

    Box(Modifier.fillMaxSize().background(Aurora.Ink)) {
        AsyncImage(
            model = client.absolute(d.backdrop ?: d.thumb), contentDescription = null, contentScale = ContentScale.Crop,
            modifier = Modifier.fillMaxSize(),
        )
        Box(Modifier.fillMaxSize().background(Brush.horizontalGradient(listOf(Color(0xF508090B), Color(0xCC08090B), Color(0x6608090B)))))
        Box(Modifier.fillMaxSize().background(Brush.verticalGradient(listOf(Color.Transparent, Aurora.Ink), startY = 300f)))

        LazyColumn(contentPadding = PaddingValues(top = 48.dp, bottom = 64.dp), verticalArrangement = Arrangement.spacedBy(26.dp)) {
            item(key = "head") {
                Row(Modifier.padding(horizontal = 56.dp), horizontalArrangement = Arrangement.spacedBy(40.dp)) {
                    Box(Modifier.width(220.dp).aspectRatio(2f / 3f).clip(RoundedCornerShape(12.dp))) {
                        Artwork(client.absolute(d.poster), client.absolute(d.thumb), d.title, d.year)
                    }
                    Column(Modifier.width(860.dp)) {
                        if (d.kind == "episode") {
                            Text("${d.seriesTitle.orEmpty()} · ${episodeCode(d.season, d.episode, d.episodeEnd)}", color = Aurora.Accent, fontSize = 16.sp)
                        }
                        Text(d.title, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 48.sp, lineHeight = 50.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        d.originalTitle?.takeIf { it.isNotBlank() && it != d.title }?.let { Text(it, color = Aurora.Ash, fontSize = 16.sp) }
                        Text(
                            listOfNotNull(
                                d.year?.toString(), d.contentRating?.takeIf { it.isNotBlank() },
                                d.rating?.let { "%.1f /10 %s".format(it, d.ratingSource.orEmpty()) },
                                if (!isSeries) runtimeLabel(d.runtime, file?.duration).ifEmpty { null } else d.status?.takeIf { it.isNotBlank() },
                                file?.resolution?.takeIf { it.isNotBlank() }, file?.hdr?.takeIf { it.isNotBlank() },
                                d.genres?.take(3)?.joinToString(", "),
                            ).joinToString("   ·   "),
                            color = Aurora.Mist, fontSize = 16.sp, modifier = Modifier.padding(top = 10.dp),
                        )
                        if (isSeries) Text("${d.total ?: 0} épisodes${if ((d.unwatched ?: 0) > 0) " · ${d.unwatched} non vus" else ""}", color = Aurora.Ash, fontSize = 15.sp, modifier = Modifier.padding(top = 4.dp))
                        // FlowRow : jusqu'à 5 boutons (film en reprise) ne tiennent pas sur une ligne à côté de l'affiche.
                        FlowRow(
                            Modifier.padding(top = 22.dp),
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalArrangement = Arrangement.spacedBy(12.dp),
                        ) {
                            when {
                                isSeries && d.nextUp != null -> {
                                    val n = d.nextUp
                                    ActionButton(
                                        (if ((n.position ?: 0.0) > 0) "▶  Reprendre " else "▶  Lecture ") + episodeCode(n.season, n.episode, n.episodeEnd),
                                        primary = true, onClick = { vm.push(Screen.Player(n.id)) }, modifier = Modifier.focusRequester(firstButton),
                                    )
                                }
                                !isSeries && file != null -> {
                                    ActionButton(
                                        if (resumable) "▶  Reprendre à ${clock(d.state.position)}" else "▶  Lecture",
                                        primary = true, onClick = { vm.push(Screen.Player(d.id, file.id)) }, modifier = Modifier.focusRequester(firstButton),
                                    )
                                    if (resumable) ActionButton("Depuis le début", onClick = { vm.push(Screen.Player(d.id, file.id, restart = true)) })
                                }
                                else -> Spacer(Modifier.focusRequester(firstButton))
                            }
                            val watched = if (isSeries) (d.unwatched ?: 1) == 0 else d.state.watched
                            ActionButton(if (watched) "Non vu" else "Vu", onClick = { toggle { client.setState(d.id, watched = !watched) } })
                            ActionButton(if (d.state.watchlist) "✓ Ma liste" else "+ Ma liste", onClick = { toggle { client.setState(d.id, watchlist = !d.state.watchlist) } })
                            ActionButton(if (d.state.favorite) "♥ Favori" else "♡ Favori", onClick = { toggle { client.setState(d.id, favorite = !d.state.favorite) } })
                        }
                        if (resumable && length > 0) ProgressLine((d.state.position / length).toFloat(), Modifier.padding(top = 14.dp).width(360.dp))
                        d.overview?.takeIf { it.isNotBlank() }?.let {
                            Text(it, color = Aurora.Mist, fontSize = 17.sp, lineHeight = 24.sp, maxLines = 5, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 18.dp))
                        }
                        val crew = d.people.filter { it.role != "actor" }.joinToString(", ") { it.name }
                        val cast = d.people.filter { it.role == "actor" }.take(6).joinToString(", ") { it.name }
                        if (crew.isNotEmpty()) Text((if (isSeries) "Création : " else "Réalisation : ") + crew, color = Aurora.Ash, fontSize = 15.sp, modifier = Modifier.padding(top = 12.dp))
                        if (cast.isNotEmpty()) Text("Avec $cast", color = Aurora.Ash, fontSize = 15.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        file?.let { f ->
                            val audio = f.audio.joinToString("  ·  ") { a -> listOf(a.language.uppercase(), a.codec.uppercase(), channels(a.channels)).filter { it.isNotBlank() }.joinToString(" ") }
                            if (!isSeries && audio.isNotEmpty()) Text("Audio : $audio", color = Aurora.Ash, fontSize = 14.sp, modifier = Modifier.padding(top = 10.dp), maxLines = 1, overflow = TextOverflow.Ellipsis)
                        }
                    }
                }
            }
            if (isSeries && !d.seasons.isNullOrEmpty()) {
                item(key = "seasons") {
                    LazyRow(contentPadding = PaddingValues(horizontal = 56.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        items(d.seasons, key = { it.number }) { s ->
                            Chip(s.title, selected = s.number == season, count = s.unwatched.takeIf { it in 1 until s.episodes }, onClick = { season = s.number })
                        }
                    }
                }
                season?.let { sn -> item(key = "eps-$sn") { Episodes(vm, client, d.id, sn) } }
            }
        }
    }
}

private fun channels(n: Int) = when (n) { 1 -> "mono"; 2 -> "stéréo"; 6 -> "5.1"; 8 -> "7.1"; 0 -> ""; else -> "$n can." }

@Composable
private fun Episodes(vm: AppViewModel, client: AuroraClient, seriesId: Long, season: Int) {
    val state by produceState<Load<List<Card>>>(Load.Loading, seriesId, season, vm.refresh) {
        value = Load.Loading
        value = load { client.episodes(seriesId, season) }
    }
    when (val s = state) {
        Load.Loading -> Text("Chargement des épisodes…", color = Aurora.Ash, modifier = Modifier.padding(horizontal = 56.dp))
        is Load.Failed -> Text(s.message, color = Aurora.Danger, modifier = Modifier.padding(horizontal = 56.dp))
        is Load.Ok -> Column(Modifier.padding(horizontal = 56.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            s.value.forEach { ep -> EpisodeRow(ep, client, onClick = { vm.push(Screen.Player(ep.id)) }) }
        }
    }
}

@Composable
private fun EpisodeRow(ep: Card, client: AuroraClient, onClick: () -> Unit) {
    var focused by remember { mutableStateOf(false) }
    Surface(
        onClick = onClick,
        shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(12.dp)),
        colors = ClickableSurfaceDefaults.colors(containerColor = Color(0x10FFFFFF), focusedContainerColor = Color(0x22FFFFFF)),
        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.01f),
        modifier = Modifier.fillMaxWidth().onFocusChanged { focused = it.isFocused },
    ) {
        Row(
            Modifier.padding(10.dp).then(if (focused) Modifier.border(0.dp, Color.Transparent) else Modifier),
            horizontalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            Box(Modifier.width(240.dp).aspectRatio(16f / 9f).clip(RoundedCornerShape(8.dp)).background(Aurora.Coal)) {
                AsyncImage(model = client.absolute(ep.thumb), contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                if (ep.watched) WatchedBadge(Modifier.align(Alignment.TopEnd).padding(6.dp))
                if (ep.progress > 0) ProgressLine(ep.progress, Modifier.align(Alignment.BottomCenter).padding(6.dp))
                if (focused) Box(Modifier.fillMaxSize().border(3.dp, Aurora.Accent, RoundedCornerShape(8.dp)))
            }
            Column(Modifier.padding(top = 4.dp)) {
                Text(
                    episodeCode(ep.season, ep.episode, ep.episodeEnd) + (if (ep.episodeEnd != null) " · épisode double" else ""),
                    color = Aurora.Ash, fontSize = 13.sp, fontWeight = FontWeight.SemiBold,
                )
                Text(ep.title, color = Aurora.Paper, fontSize = 19.sp, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                ep.overview?.takeIf { it.isNotBlank() }?.let {
                    Text(it, color = Aurora.Mist, fontSize = 14.sp, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 4.dp))
                }
                Spacer(Modifier.height(6.dp))
                Text(
                    listOfNotNull(
                        runtimeLabel(ep.runtime, ep.duration).ifEmpty { null },
                        (ep.position ?: 0.0).takeIf { it > 0 && !ep.watched }?.let { "Reprise à ${clock(it)}" },
                    ).joinToString("   ·   "),
                    color = if ((ep.position ?: 0.0) > 0) Aurora.Accent else Aurora.Ash, fontSize = 13.sp,
                )
            }
        }
    }
}
