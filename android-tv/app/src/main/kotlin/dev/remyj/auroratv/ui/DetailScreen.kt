package dev.remyj.auroratv.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
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
import dev.remyj.auroratv.ui.components.MoreButton
import dev.remyj.auroratv.ui.components.rememberWatchedCard
import dev.remyj.auroratv.ui.components.WatchedMenu
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
import androidx.compose.ui.text.font.FontStyle
import dev.remyj.auroratv.ui.components.Eyebrow
import dev.remyj.auroratv.ui.components.SegmentBar
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.relocation.BringIntoViewRequester
import androidx.compose.foundation.relocation.bringIntoViewRequester
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.focus.focusProperties
import androidx.tv.material3.Border
import dev.remyj.auroratv.net.Season

@Composable
fun DetailScreen(vm: AppViewModel, screen: Screen.Detail) {
    val client = vm.client ?: return
    var reload by remember { mutableIntStateOf(0) }
    val state by produceState<Load<Detail>>(Load.Loading, screen.itemId, reload, vm.refresh) {
        value = load { client.item(screen.itemId) }
    }
    when (val s = state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> ErrorMessage(s.message) { reload++ }
        is Load.Ok -> DetailContent(vm, client, s.value, onChanged = { reload++ })
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun DetailContent(vm: AppViewModel, client: AuroraClient, d: Detail, onChanged: () -> Unit) {
    val scope = rememberCoroutineScope()
    val firstButton = remember { FocusRequester() }
    val isSeries = d.kind == "series"
    val file = d.files.firstOrNull { it.id == d.state.lastFileId } ?: d.files.firstOrNull()
    val length = file?.duration?.takeIf { it > 0 } ?: ((d.runtime ?: 0) * 60.0)
    val resumable = !isSeries && d.state.position > 0 && !d.state.watched
    var season by rememberSaveable(d.id) { mutableStateOf(d.nextUp?.season ?: d.seasons?.firstOrNull()?.number) }
    // Ligne et épisode sous le focus : la barre collante affiche « Épisode 4 sur 10 », et le retour du
    // lecteur retrouve la ligne qu'on vient de quitter (saveable : l'écran est détruit pendant la lecture).
    var focusedRow by rememberSaveable(d.id) { mutableStateOf<Long?>(null) }
    // Par défaut, la barre annonce l'épisode que l'on va regarder (« Épisode 4 sur 10 »), pas un vide.
    var focusedEpisode by rememberSaveable(d.id) { mutableStateOf(d.nextUp?.episode) }
    // La saison de cet épisode : sans elle, la barre annonçait « Épisode 4 sur 24 » pour la saison 3.
    var focusedSeason by rememberSaveable(d.id) { mutableStateOf(d.nextUp?.season) }
    var rowRestored by remember { mutableStateOf(false) }
    var userMoved by remember { mutableStateOf(false) }
    var jumpTo by remember { mutableStateOf<Long?>(null) }
    val listState = rememberLazyListState()
    LaunchedEffect(d.id) {
        // Au retour du lecteur, c'est la ligne regardée qui reprend le focus, pas la barre de saison.
        if (focusedRow != null) {
            // La ligne n'est peut-être pas encore composée (position de défilement restaurée) : on attend.
            repeat(18) {
                if (rowRestored || userMoved) return@LaunchedEffect
                delay(80)
            }
            // Elle n'est jamais revenue (épisode supprimé, saison changée) : on rend le focus à la fiche,
            // mais seulement si l'utilisateur n'a pas déjà repris la main sur une autre ligne.
            runCatching { firstButton.requestFocus() }
            return@LaunchedEffect
        }
        // La barre de saison peut n'être composée qu'après l'en-tête : on réessaie plutôt que de laisser
        // la télécommande sans focus (runCatching avalait l'échec et il fallait une flèche pour démarrer).
        repeat(10) {
            if (userMoved) return@LaunchedEffect
            delay(80)
            if (runCatching { firstButton.requestFocus() }.getOrDefault(false)) {
                // Le focus sur le chip de saison fait défiler la fiche (~160 dp) : on la rouvre sur son
                // titre une fois le focus posé, le chip restant visible juste en dessous.
                delay(250)
                listState.scrollToItem(0)
                return@LaunchedEffect
            }
        }
    }
    // « Où j'en suis » : amène la fiche sur la ligne du prochain épisode, sans lancer la lecture.
    val jumpToNext: () -> Unit = {
        d.nextUp?.let { next ->
            jumpTo = next.id
            next.season?.let { season = it }
        }
    }
    val toggle: (suspend () -> Unit) -> Unit = { action -> scope.launch { runCatching { action() }; onChanged() } }

    Box(Modifier.fillMaxSize().background(Aurora.Ink)) {
        // Fond du film à droite, fondu vers la gauche et vers le bas (comme sur Mac).
        Box(Modifier.align(Alignment.TopEnd).fillMaxWidth(0.58f).height(330.dp)) {
            AsyncImage(
                model = client.absolute(d.backdrop ?: d.thumb), contentDescription = null, contentScale = ContentScale.Crop,
                alpha = 0.5f, modifier = Modifier.fillMaxSize(),
            )
            Box(Modifier.fillMaxSize().background(Brush.horizontalGradient(0f to Aurora.Ink, 0.45f to Color.Transparent)))
            Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0.55f to Color.Transparent, 1f to Aurora.Ink)))
        }

        LazyColumn(state = listState, contentPadding = PaddingValues(top = 30.dp, bottom = 48.dp), verticalArrangement = Arrangement.spacedBy(22.dp)) {
            item(key = "head") {
                Row(Modifier.padding(horizontal = 40.dp), horizontalArrangement = Arrangement.spacedBy(28.dp)) {
                    Box(
                        Modifier.width(170.dp).aspectRatio(2f / 3f).clip(RoundedCornerShape(6.dp))
                            .border(1.dp, Color(0x1FF4EAD8), RoundedCornerShape(6.dp)),
                    ) {
                        Artwork(client.absolute(d.poster), client.absolute(d.thumb), d.title, d.year)
                    }
                    Column(Modifier.width(620.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Eyebrow(
                            when (d.kind) {
                                "series" -> listOfNotNull("Série", d.year?.toString()).joinToString(" · ")
                                "episode" -> "${d.seriesTitle.orEmpty()} · ${episodeCode(d.season, d.episode, d.episodeEnd)}"
                                else -> "Film"
                            },
                        )
                        Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            Text(d.title, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 40.sp, lineHeight = 42.sp, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f, fill = false))
                            d.originalTitle?.takeIf { it.isNotBlank() && it != d.title }?.let {
                                Text(it, color = Aurora.Ash, fontFamily = Aurora.Display, fontSize = 18.sp, maxLines = 1, modifier = Modifier.padding(bottom = 6.dp))
                            }
                        }
                        creditLine(d.kind, d.people)?.let {
                            Text(it, color = Aurora.Mist, fontFamily = Aurora.Display, fontStyle = FontStyle.Italic, fontSize = 15.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        }
                        Row(horizontalArrangement = Arrangement.spacedBy(7.dp), verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                listOfNotNull(
                                    if (isSeries) d.seasons?.size?.let { if (it > 1) "$it saisons" else "1 saison" } else d.year?.toString(),
                                    if (!isSeries) runtimeLabel(d.runtime, file?.duration).ifEmpty { null } else null,
                                    d.genres?.firstOrNull(),
                                    d.rating?.let { "%.1f/10".format(it) },
                                ).joinToString(" · "),
                                color = Aurora.Mist, fontSize = 14.sp,
                            )
                            listOfNotNull(d.contentRating, file?.resolution, file?.hdr).filter { it.isNotBlank() }.forEach { Badge(it) }
                        }
                        d.overview?.takeIf { it.isNotBlank() }?.let {
                            Text(it, color = Aurora.Mist, fontSize = 15.sp, lineHeight = 21.sp, maxLines = 4, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 4.dp))
                        }
                        // Série : aucun bouton ici (lecture, « vu », Ma liste, Favori sont sur l'épisode).
                        if (!isSeries) FlowRow(
                            Modifier.padding(top = 10.dp),
                            horizontalArrangement = Arrangement.spacedBy(10.dp),
                            verticalArrangement = Arrangement.spacedBy(10.dp),
                        ) {
                            if (file != null) {
                                ActionButton(
                                    if (resumable) "▶  Reprendre à ${clock(d.state.position)}" else "▶  Lecture",
                                    primary = true, onClick = { vm.push(Screen.Player(d.id, file.id)) }, modifier = Modifier.focusRequester(firstButton),
                                )
                                if (resumable) ActionButton("Depuis le début", onClick = { vm.push(Screen.Player(d.id, file.id, restart = true)) })
                            }
                            ActionButton(
                                if (d.state.watched) "Non vu" else "✓ Vu",
                                // Sans fichier, « Vu » est la première action disponible : elle prend le focus.
                                onClick = { toggle { client.setState(d.id, watched = !d.state.watched) } },
                                modifier = if (file == null) Modifier.focusRequester(firstButton) else Modifier,
                            )
                            if (d.kind == "movie") ActionButton(if (d.state.watchlist) "✓ Ma liste" else "+ Ma liste", onClick = { toggle { client.setState(d.id, watchlist = !d.state.watchlist) } })
                            if (d.kind == "movie") ActionButton(if (d.state.favorite) "♥ Favori" else "♡ Favori", onClick = { toggle { client.setState(d.id, favorite = !d.state.favorite) } })
                        }
                        if (resumable && length > 0) ProgressLine((d.state.position / length).toFloat(), Modifier.padding(top = 8.dp).width(300.dp))
                        if (isSeries) {
                            val total = d.total ?: 0
                            val seen = (total - (d.unwatched ?: 0)).coerceAtLeast(0)
                            if (total > 0) Row(Modifier.padding(top = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                                ProgressLine(seen.toFloat() / total, Modifier.width(150.dp))
                                Text("  $seen épisode${if (seen > 1) "s" else ""} vu${if (seen > 1) "s" else ""} sur $total", color = Aurora.Ash, fontSize = 13.sp)
                            }
                        }
                        file?.let { f ->
                            val audio = f.audio.joinToString("  ·  ") { a -> listOf(a.language.uppercase(), a.codec.uppercase(), channels(a.channels)).filter { it.isNotBlank() }.joinToString(" ") }
                            if (!isSeries && audio.isNotEmpty()) Text("Audio : $audio", color = Aurora.Ash, fontSize = 13.sp, modifier = Modifier.padding(top = 6.dp), maxLines = 1, overflow = TextOverflow.Ellipsis)
                        }
                    }
                }
            }
            if (isSeries && !d.seasons.isNullOrEmpty()) {
                // Barre de saison collante (maquette du 29/10/2026) : elle reste en haut pendant le défilement
                // des épisodes et affiche « Épisode 4 sur 10 » — sinon on ne sait plus où l'on est dans la saison.
                stickyHeader(key = "seasons") {
                    SeasonBar(
                        title = d.title,
                        showTitle = { listState.firstVisibleItemIndex > 0 },
                        seasons = d.seasons,
                        selected = season,
                        episode = focusedEpisode.takeIf { focusedSeason == season },
                        nextUp = d.nextUp,
                        onJump = jumpToNext,
                        onSelect = { season = it },
                        first = firstButton,
                    )
                }
                season?.let { sn ->
                    item(key = "eps-$sn") {
                        Episodes(
                            vm, client, d.id, sn, current = d.nextUp?.id, onChanged = onChanged,
                            restoreId = focusedRow,
                            jumpTo = jumpTo,
                            onJumped = { jumpTo = null },
                            onRestored = { rowRestored = true },
                            onEpisodeFocus = { id, number ->
                                // Une ligne qui perd le focus n'efface la position que si c'est encore la sienne :
                                // en remontant la liste, la ligne quittée s'exécute après celle qui arrive.
                                if (number != null) {
                                    userMoved = true
                                    focusedRow = id; focusedEpisode = number; focusedSeason = sn
                                } else if (focusedRow == id) {
                                    focusedRow = null
                                    focusedEpisode = d.nextUp?.episode
                                    focusedSeason = d.nextUp?.season
                                }
                            },
                        )
                    }
                }
            } else if (isSeries) {
                // Série sans saison indexée : sans cette cible, l'écran s'ouvrait sans aucun focus.
                item(key = "no-episodes") {
                    Column(Modifier.padding(horizontal = 40.dp, vertical = 16.dp)) {
                        Text("Aucun épisode indexé pour cette série.", color = Aurora.Mist, fontSize = 16.sp)
                        ActionButton(
                            "Retour", onClick = { vm.back() },
                            modifier = Modifier.padding(top = 12.dp).focusRequester(firstButton),
                        )
                    }
                }
            }
            val cast = d.people.filter { it.role == "actor" }.take(8)
            if (cast.isNotEmpty()) item(key = "cast") {
                Column(Modifier.padding(horizontal = 40.dp)) {
                    Box(Modifier.fillMaxWidth().height(1.dp).background(Aurora.Line))
                    Text("Distribution", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 20.sp, modifier = Modifier.padding(top = 12.dp, bottom = 6.dp))
                    // Texte façon générique : « Hiroyuki Sanada — Toranaga ».
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(24.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        cast.forEach { p ->
                            Row {
                                Text(p.name, color = Aurora.Mist, fontFamily = Aurora.Display, fontSize = 15.sp)
                                p.character?.takeIf { it.isNotBlank() }?.let {
                                    Text(" — $it", color = Aurora.Ash, fontFamily = Aurora.Display, fontStyle = FontStyle.Italic, fontSize = 15.sp)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun channels(n: Int) = when (n) { 1 -> "mono"; 2 -> "stéréo"; 6 -> "5.1"; 8 -> "7.1"; 0 -> ""; else -> "$n can." }

/**
 * Barre de saison collante : la saison choisie et l'épisode sous le focus restent visibles pendant le
 * défilement (maquette du 29/10/2026). Sans elle, une longue saison faisait perdre le repère.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun SeasonBar(
    title: String,
    showTitle: () -> Boolean,
    seasons: List<Season>,
    selected: Int?,
    episode: Int?,
    nextUp: Card?,
    onJump: () -> Unit,
    onSelect: (Int) -> Unit,
    first: FocusRequester,
) {
    val info = seasons.firstOrNull { it.number == selected }
    val total = info?.episodes ?: 0
    val seen = (total - (info?.unwatched ?: 0)).coerceAtLeast(0)
    val vu = "vu${if (seen > 1) "s" else ""}"
    val where = when {
        total <= 0 -> ""
        episode != null -> "Épisode $episode sur $total · $seen $vu"
        else -> "$total épisodes · $seen $vu"
    }
    // Le titre apparaît en réduisant la place : la saison choisie doit rester visible même avec 12 saisons.
    // (Rien au premier affichage : le focus arrive déjà sur le chip et la fiche doit s'ouvrir sur son titre.)
    val bring = remember { BringIntoViewRequester() }
    var lastSelected by remember { mutableStateOf(selected) }
    LaunchedEffect(selected) {
        if (selected == lastSelected) return@LaunchedEffect
        lastSelected = selected
        delay(60)
        runCatching { bring.bringIntoView() }
    }
    Column(Modifier.fillMaxWidth().background(Aurora.Night.copy(alpha = 0.97f))) {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 32.dp, vertical = 7.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Le titre n'apparaît qu'une fois l'en-tête défilé : sinon il doublerait le grand titre.
            if (showTitle()) {
                Text(
                    title, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 16.sp,
                    maxLines = 1, overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.padding(end = 14.dp).widthIn(max = 200.dp),
                )
            }
            LazyRow(Modifier.weight(1f)) {
                // Repère « où j'en suis » : amène sur la ligne du prochain épisode sans lancer la lecture.
                // Il défile avec les saisons pour ne pas leur voler de la place sur une série à 12 saisons.
                nextUp?.takeIf { episodeCode(it.season, it.episode).isNotBlank() }?.let { next ->
                    item(key = "on-deck") {
                        Chip(
                            "Où j'en suis · ${episodeCode(next.season, next.episode)}",
                            selected = false,
                            onClick = onJump,
                            modifier = Modifier.padding(end = 10.dp),
                        )
                    }
                }
                item(key = "chips") {
                    SegmentBar {
                        seasons.forEach { s ->
                            Chip(
                                s.title, selected = s.number == selected,
                                count = s.unwatched.takeIf { it in 1 until s.episodes },
                                onClick = { onSelect(s.number) },
                                modifier = if (s.number == selected) {
                                    Modifier.focusRequester(first).bringIntoViewRequester(bring)
                                } else Modifier,
                            )
                        }
                    }
                }
            }
            if (where.isNotEmpty()) {
                Text(where, color = Aurora.Ash, fontSize = 12.sp, maxLines = 1, modifier = Modifier.padding(start = 14.dp))
            }
        }
        Box(Modifier.fillMaxWidth().height(1.dp).background(Aurora.Line))
    }
}

@Composable
private fun Episodes(
    vm: AppViewModel, client: AuroraClient, seriesId: Long, season: Int, current: Long?,
    onChanged: () -> Unit, restoreId: Long?, jumpTo: Long?,
    onJumped: () -> Unit, onRestored: () -> Unit,
    onEpisodeFocus: (Long, Int?) -> Unit,
) {
    // Nouvelle tentative sans passer par `vm.retry` : ce drapeau de focus est réservé aux grilles,
    // une fiche ne doit pas voler le focus de la carte restaurée au retour sur l'accueil.
    var attempt by remember { mutableStateOf(0) }
    val state by produceState<Load<List<Card>>>(Load.Loading, seriesId, season, vm.refresh, attempt) {
        value = Load.Loading
        value = load { client.episodes(seriesId, season) }
    }
    when (val s = state) {
        Load.Loading -> Text("Chargement des épisodes…", color = Aurora.Ash, modifier = Modifier.padding(horizontal = 40.dp))
        is Load.Failed -> Column(Modifier.padding(horizontal = 40.dp)) {
            Text(s.message, color = Aurora.Danger, modifier = Modifier.padding(bottom = 10.dp))
            ActionButton("Réessayer", onClick = { attempt++ })
        }
        // Liste verticale (comme Mac et Samsung) : vignette à gauche, résumé à droite.
        is Load.Ok -> Column(Modifier.padding(horizontal = 32.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            s.value.forEach { ep ->
                EpisodeRow(
                    ep, client, current = ep.id == current,
                    onPlay = { vm.push(Screen.Player(ep.id)) },
                    onChanged = onChanged,
                    restore = ep.id == restoreId,
                    jump = ep.id == jumpTo,
                    onJumped = onJumped,
                    onRestored = onRestored,
                    onEpisodeFocus = onEpisodeFocus,
                )
            }
        }
    }
}

private val EpisodeThumbWidth = 200.dp
private val EpisodeThumbHeight = 112.dp
private val EpisodeRowPadding = 8.dp
private val EpisodeRowShape = RoundedCornerShape(10.dp)

/**
 * Ligne d'épisode, à la manière de Plex : rien sur la vignette s'il n'est pas vu, coche verte s'il l'est.
 * Le « ⋯ » est hors de la vignette, à droite de la ligne, dans sa propre colonne : sur la photo il se
 * confondait avec l'image et le texte passait dessous (maquette du 29/10/2026). ▶ l'atteint, OK ouvre
 * Vu / Pas vu, ◀ revient ; OK sur la ligne lance la lecture.
 * Le focus reprend le repère des affiches — cadre crème-or épais, halo, titre en or — sinon on ne voit pas
 * sur quelle ligne on est à trois mètres.
 */
@Composable
private fun EpisodeRow(
    source: Card, client: AuroraClient, current: Boolean,
    onPlay: () -> Unit, onChanged: () -> Unit, restore: Boolean,
    jump: Boolean, onJumped: () -> Unit, onRestored: () -> Unit,
    onEpisodeFocus: (Long, Int?) -> Unit,
) {
    var ep by rememberWatchedCard(source)
    var rowFocused by remember { mutableStateOf(false) }
    var moreFocused by remember { mutableStateOf(false) }
    val active = rowFocused || moreFocused
    val rowRequester = remember { FocusRequester() }
    val moreRequester = remember { FocusRequester() }
    var menu by remember { mutableStateOf(false) }
    var restoreMore by remember { mutableStateOf(false) }
    LaunchedEffect(menu) {
        if (!menu && restoreMore) { delay(80); runCatching { moreRequester.requestFocus() }; restoreMore = false }
    }
    // Retour du lecteur : on retrouve la ligne qu'on vient de regarder.
    LaunchedEffect(Unit) {
        if (restore) repeat(5) {
            delay(80)
            if (runCatching { rowRequester.requestFocus() }.getOrDefault(false)) { onRestored(); return@LaunchedEffect }
        }
    }
    // « Où j'en suis » : la ligne du prochain épisode peut n'être composée qu'après le chargement de saison.
    LaunchedEffect(jump) {
        if (!jump) return@LaunchedEffect
        repeat(15) {
            if (runCatching { rowRequester.requestFocus() }.getOrDefault(false)) { onJumped(); return@LaunchedEffect }
            delay(80)
        }
        onJumped()
    }
    // Position affichée dans la barre collante : une ligne qui perd le focus n'efface que la sienne.
    LaunchedEffect(active) { onEpisodeFocus(ep.id, if (active) ep.episode else null) }
    if (menu) WatchedMenu(ep, client, onDismiss = { menu = false; restoreMore = true }, onChanged = { ep = it; onChanged() })
    Surface(
        onClick = onPlay,
        // Appui long = Vu / Pas vu, en plus du « ⋯ » (qui reste atteignable par ▶).
        onLongClick = { menu = true },
        shape = ClickableSurfaceDefaults.shape(EpisodeRowShape),
        colors = ClickableSurfaceDefaults.colors(containerColor = Color.Transparent, focusedContainerColor = Color(0x1FF4EAD8)),
        border = ClickableSurfaceDefaults.border(focusedBorder = Border(BorderStroke(3.dp, Aurora.FocusGold), shape = EpisodeRowShape)),
        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.012f),
        modifier = Modifier.fillMaxWidth().focusRequester(rowRequester)
            .focusProperties { right = moreRequester }
            .onFocusChanged { rowFocused = it.isFocused }
            .shadow(if (active) 16.dp else 0.dp, EpisodeRowShape, clip = false,
                ambientColor = Aurora.Accent, spotColor = Aurora.Accent),
    ) {
        Row(Modifier.padding(EpisodeRowPadding), verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.width(EpisodeThumbWidth).height(EpisodeThumbHeight).clip(RoundedCornerShape(5.dp)).background(Aurora.Coal)
                    .then(
                        when {
                            active -> Modifier.border(2.dp, Aurora.FocusGold, RoundedCornerShape(5.dp))
                            current -> Modifier.border(2.dp, Aurora.Accent, RoundedCornerShape(5.dp))
                            else -> Modifier
                        },
                    ),
            ) {
                AsyncImage(model = client.absolute(ep.thumb), contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                if (ep.watched) WatchedBadge(Modifier.align(Alignment.TopEnd).padding(6.dp))
                // Toute la largeur de la vignette : les « ⋯ » ne sont plus dessus.
                if (ep.progress > 0 && !ep.watched) ProgressLine(ep.progress, Modifier.align(Alignment.BottomStart).padding(start = 7.dp, end = 7.dp, bottom = 7.dp))
            }
            Column(Modifier.weight(1f).padding(start = 18.dp, end = 14.dp, top = 4.dp)) {
                Text(
                    "ÉPISODE ${ep.episode ?: 0}" + (ep.episodeEnd?.let { "-$it" } ?: ""),
                    color = if (active) Aurora.Accent else Aurora.Ash,
                    fontSize = 11.sp, fontWeight = FontWeight.SemiBold, letterSpacing = 2.sp,
                )
                Text(
                    ep.title, color = if (active) Aurora.FocusGold else Aurora.Mist,
                    fontFamily = Aurora.Display, fontSize = 18.sp,
                    fontWeight = if (active) FontWeight.SemiBold else FontWeight.Normal,
                    maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 2.dp),
                )
                ep.overview?.takeIf { it.isNotBlank() }?.let {
                    Text(it, color = Aurora.Mist, fontSize = 13.sp, lineHeight = 18.sp, maxLines = 3, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 4.dp))
                }
                Text(
                    listOfNotNull(
                        runtimeLabel(ep.runtime, ep.duration).ifEmpty { null },
                        (ep.position ?: 0.0).takeIf { it > 0 && !ep.watched }?.let { "reprise à ${clock(it)}" },
                    ).joinToString(" · "),
                    color = if ((ep.position ?: 0.0) > 0 && !ep.watched) Aurora.Accent else Aurora.Ash, fontSize = 12.sp, modifier = Modifier.padding(top = 4.dp),
                )
            }
            // Colonne d'action, à l'extérieur du résumé : le « ⋯ » se voit et se vise sans toucher la vignette.
            Box(Modifier.width(1.dp).height(EpisodeThumbHeight).background(Aurora.Line))
            MoreButton(onClick = { menu = true }, modifier = Modifier.padding(horizontal = 12.dp)
                .focusRequester(moreRequester)
                .focusProperties { left = rowRequester }
                .onFocusChanged { moreFocused = it.isFocused })
        }
    }
}
