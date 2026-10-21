package dev.remyj.auroratv.ui

import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import kotlinx.coroutines.launch
import androidx.compose.ui.window.Dialog
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.tv.material3.Text
import coil3.compose.AsyncImage
import dev.remyj.auroratv.net.ApiException
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.Card
import dev.remyj.auroratv.net.Home
import dev.remyj.auroratv.net.LibraryRef
import dev.remyj.auroratv.net.ListPage
import dev.remyj.auroratv.net.SearchResults
import dev.remyj.auroratv.ui.components.ActionButton
import dev.remyj.auroratv.ui.components.Chip
import dev.remyj.auroratv.ui.components.PosterCard
import dev.remyj.auroratv.ui.components.ProgressLine
import dev.remyj.auroratv.ui.components.Rail
import dev.remyj.auroratv.ui.components.runtimeLabel
import dev.remyj.auroratv.ui.theme.Aurora
import kotlinx.coroutines.delay

/** Résultat de chargement d'un écran. */
sealed interface Load<out T> {
    data object Loading : Load<Nothing>
    data class Ok<T>(val value: T) : Load<T>
    data class Failed(val message: String) : Load<Nothing>
}

suspend fun <T> load(block: suspend () -> T): Load<T> = try {
    Load.Ok(block())
} catch (e: ApiException) {
    Load.Failed(e.message ?: "Erreur")
}

@Composable
fun MainScreen(vm: AppViewModel) {
    val client = vm.client ?: return
    val open: (Card) -> Unit = { c ->
        when (c.kind) {
            // Un épisode ou un film en cours se lance directement (comme une carte « Reprendre »).
            "episode" -> vm.push(Screen.Player(c.id))
            else -> vm.push(Screen.Detail(c.id))
        }
    }
    // Retour d'une fiche : le focus revient sur la carte ouverte (sinon il repartait sur l'onglet).
    val restoring = remember { vm.cardFocus.last?.also { vm.cardFocus.restore = it } != null }
    Column(Modifier.fillMaxSize().background(Aurora.Ink)) {
        TopBar(vm, restoring)
        Box(Modifier.fillMaxSize()) {
            when (vm.tab) {
                Tab.HOME -> HomeTab(vm, client, open)
                Tab.MOVIES -> LibraryTab(vm, client, "movie", open)
                Tab.SERIES -> LibraryTab(vm, client, "series", open)
                Tab.SEARCH -> SearchTab(client, open)
                Tab.PROFILE -> ProfileTab(vm)
            }
        }
    }
}

@Composable
private fun TopBar(vm: AppViewModel, restoring: Boolean) {
    val first = remember { FocusRequester() }
    LaunchedEffect(Unit) {
        if (restoring) {
            // La carte n'est réapparue qu'après rechargement ; si elle n'existe plus, focus sur l'onglet.
            delay(1_500)
            if (vm.cardFocus.restore == null) return@LaunchedEffect
            vm.cardFocus.restore = null
        } else {
            delay(100)
        }
        runCatching { first.requestFocus() }
    }
    Row(
        Modifier.fillMaxWidth().padding(start = 48.dp, end = 48.dp, top = 24.dp, bottom = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text("AURORA", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 22.sp, letterSpacing = 5.sp)
        Spacer(Modifier.width(36.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Tab.entries.forEach { t ->
                Chip(
                    t.label, selected = vm.tab == t, onClick = { vm.tab = t },
                    modifier = if (t == vm.tab) Modifier.focusRequester(first) else Modifier,
                )
            }
        }
        Spacer(Modifier.weight(1f))
        Text("${vm.me?.username.orEmpty()} · ${vm.server?.name.orEmpty()}", color = Aurora.Ash, fontSize = 14.sp)
    }
}

// --- Accueil --------------------------------------------------------------------------------------

@Composable
private fun HomeTab(vm: AppViewModel, client: AuroraClient, open: (Card) -> Unit) {
    val state by produceState<Load<Home>>(Load.Loading, client, vm.refresh) { value = load { client.home() } }
    // Appui long sur une carte « Continuer » / « Prochain épisode » : la retirer (fichier jamais touché).
    var removing by remember { mutableStateOf<Card?>(null) }
    removing?.let { card ->
        RemoveContinueDialog(card, onDismiss = { removing = null }, onConfirm = {
            try {
                client.hideContinue(card.id)
                removing = null
                vm.reload()
                null
            } catch (e: ApiException) {
                e.message ?: "Retrait impossible"
            }
        })
    }
    when (val s = state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> Message(s.message)
        is Load.Ok -> {
            val home = s.value
            if (home.hero == null && home.sections.isEmpty()) {
                Message("Aucune vidéo pour l'instant. Ajoutez un dossier depuis l'administration web d'AURORA.")
                return
            }
            LazyColumn(contentPadding = PaddingValues(bottom = 48.dp), verticalArrangement = Arrangement.spacedBy(28.dp)) {
                home.hero?.let { hero -> item(key = "hero") { Hero(hero, client, vm) } }
                items(home.sections, key = { it.id }) { section ->
                    val continueRow = section.id == "continue" || section.id == "nextup"
                    Rail(
                        section.title, section.items, client, wide = continueRow, onOpen = open,
                        onLongPress = if (continueRow) { c -> removing = c } else null,
                    )
                }
            }
        }
    }
}

@Composable
private fun Hero(card: Card, client: AuroraClient, vm: AppViewModel) {
    Box(Modifier.fillMaxWidth().height(380.dp)) {
        AsyncImage(
            model = client.absolute(card.backdrop ?: card.thumb), contentDescription = null,
            contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize(),
        )
        Box(
            Modifier.fillMaxSize().background(
                Brush.horizontalGradient(listOf(Color(0xF208090B), Color(0x9908090B), Color.Transparent)),
            ),
        )
        Box(Modifier.fillMaxSize().background(Brush.verticalGradient(listOf(Color.Transparent, Aurora.Ink))))
        Column(Modifier.align(Alignment.BottomStart).padding(start = 48.dp, bottom = 24.dp).width(760.dp)) {
            Text(if (card.progress > 0) "REPRENDRE" else "CE SOIR À L'AFFICHE", color = Aurora.Ash, fontSize = 13.sp, letterSpacing = 3.sp)
            Text(card.title, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 52.sp, lineHeight = 54.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(
                listOfNotNull(card.year?.toString(), runtimeLabel(card.runtime, card.duration).ifEmpty { null }, card.resolution, card.hdr).joinToString("  ·  "),
                color = Aurora.Mist, fontSize = 16.sp, modifier = Modifier.padding(top = 6.dp),
            )
            card.overview?.takeIf { it.isNotBlank() }?.let {
                Text(it, color = Aurora.Mist, fontSize = 16.sp, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 10.dp))
            }
            Row(Modifier.padding(top = 18.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                ActionButton(if (card.progress > 0) "▶  Reprendre" else "▶  Lecture", primary = true, onClick = { vm.push(Screen.Player(card.id)) })
                ActionButton("Détails", onClick = { vm.push(Screen.Detail(card.id)) })
            }
            if (card.progress > 0) ProgressLine(card.progress, Modifier.padding(top = 14.dp).width(320.dp))
        }
    }
}

// --- Films / Séries : un onglet par dossier, « Tous » = un bloc par dossier -------------------------

private const val ALL = -1L

@Composable
private fun LibraryTab(vm: AppViewModel, client: AuroraClient, kind: String, open: (Card) -> Unit) {
    val libsState by produceState<Load<List<LibraryRef>>>(Load.Loading, client) { value = load { client.libraries() } }
    val libs = (libsState as? Load.Ok)?.value?.filter { if (kind == "movie") it.movies > 0 else it.series > 0 } ?: emptyList()
    var selected by rememberSaveable(kind) { mutableStateOf<Long?>(null) }
    var sort by rememberSaveable(kind) { mutableStateOf(if (kind == "movie") "added" else "title") }
    val active = when {
        libs.size <= 1 -> libs.firstOrNull()?.id
        selected == ALL -> ALL
        libs.any { it.id == selected } -> selected
        else -> libs.first().id
    }
    Column(Modifier.fillMaxSize()) {
        LazyRow(
            contentPadding = PaddingValues(horizontal = 48.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            if (libs.size > 1) {
                items(libs, key = { it.id }) { l ->
                    Chip(l.name, selected = active == l.id, count = if (kind == "movie") l.movies else l.series, onClick = { selected = l.id })
                }
                item { Chip("Tous les dossiers", selected = active == ALL, onClick = { selected = ALL }) }
                item { Spacer(Modifier.width(24.dp)) }
            }
            val sorts = if (kind == "movie") listOf("added" to "Ajout récent", "title" to "Titre", "year" to "Année", "rating" to "Note")
            else listOf("title" to "Titre", "added" to "Ajout récent", "year" to "Année")
            items(sorts, key = { it.first }) { (id, label) -> Chip(label, selected = sort == id, onClick = { sort = id }) }
        }
        when {
            libsState is Load.Loading -> Message("Chargement…")
            libsState is Load.Failed -> Message((libsState as Load.Failed).message)
            libs.isEmpty() -> Message(if (kind == "movie") "Aucun film." else "Aucune série.")
            active == ALL -> GroupedGrid(vm, client, kind, libs, sort, open)
            else -> FolderGrid(vm, client, kind, active, sort, open)
        }
    }
}

private val sortDesc = setOf("added", "year", "rating")

/** Tout un dossier : le NAS plafonne à 500 œuvres par page, on enchaîne les pages jusqu'au total. */
private suspend fun listAll(client: AuroraClient, kind: String, libraryId: Long?, sort: String): ListPage {
    val first = client.list(kind, libraryId, sort, sort in sortDesc, 0, PAGE)
    val items = first.items.toMutableList()
    while (items.size < first.total) {
        val page = client.list(kind, libraryId, sort, sort in sortDesc, items.size, PAGE)
        if (page.items.isEmpty()) break
        items += page.items
    }
    // L'ordre peut glisser entre deux pages (ajout, enrichissement TMDB) : une œuvre en double ferait
    // planter la grille (clé de LazyVerticalGrid dupliquée).
    return first.copy(items = items.distinctBy { it.id })
}

private const val PAGE = 500

@Composable
private fun FolderGrid(vm: AppViewModel, client: AuroraClient, kind: String, libraryId: Long?, sort: String, open: (Card) -> Unit) {
    val state by produceState<Load<ListPage>>(Load.Loading, libraryId, sort, vm.refresh) {
        value = Load.Loading
        value = load { listAll(client, kind, libraryId, sort) }
    }
    when (val s = state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> Message(s.message)
        is Load.Ok -> PosterGrid(client, s.value.items, open)
    }
}

/** « Tous les dossiers » : titre du dossier, puis ses œuvres ; jamais mélangés. */
@Composable
private fun GroupedGrid(vm: AppViewModel, client: AuroraClient, kind: String, libs: List<LibraryRef>, sort: String, open: (Card) -> Unit) {
    val state by produceState<Load<List<Pair<LibraryRef, ListPage>>>>(Load.Loading, libs, sort, vm.refresh) {
        value = Load.Loading
        value = load { libs.map { it to listAll(client, kind, it.id, sort) } }
    }
    when (val s = state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> Message(s.message)
        // Un panneau façon fenêtre macOS par dossier : barre de titre (pastilles, nom, nombre) puis ses œuvres
        // par rangées de 7. Le panneau est découpé en éléments de liste (barre, rangées, bas arrondi) pour rester
        // paresseux sur une grosse bibliothèque.
        is Load.Ok -> LazyColumn(
            contentPadding = PaddingValues(start = 48.dp, end = 48.dp, top = 12.dp, bottom = 48.dp),
        ) {
            s.value.filter { it.second.items.isNotEmpty() }.forEach { (lib, page) ->
                item(key = "h${lib.id}") { FolderTitleBar(lib.name, page.total, kind) }
                val rows = page.items.chunked(7)
                rows.forEachIndexed { i, row ->
                    item(key = "r${lib.id}-$i") {
                        val last = i == rows.lastIndex
                        Row(
                            Modifier
                                .fillMaxWidth()
                                .background(PanelFill, if (last) PanelBottom else RectangleShape)
                                .padding(start = 22.dp, end = 22.dp, top = if (i == 0) 20.dp else 10.dp, bottom = if (last) 24.dp else 10.dp),
                            horizontalArrangement = Arrangement.spacedBy(20.dp),
                        ) {
                            // Parts égales : 7 colonnes alignées d'une rangée à l'autre, dernière rangée complétée à vide.
                            row.forEach { c -> PosterCard(c, client, onClick = { open(c) }, modifier = Modifier.weight(1f)) }
                            repeat(7 - row.size) { Spacer(Modifier.weight(1f)) }
                        }
                    }
                }
                item(key = "s${lib.id}") { Spacer(Modifier.height(28.dp)) }
            }
        }
    }
}

private val PanelFill = Color(0x17FFFFFF) // ~9 % de blanc : le panneau se détache du fond noir
private val PanelBar = Color(0x24FFFFFF)
private val PanelTop = RoundedCornerShape(topStart = 18.dp, topEnd = 18.dp)
private val PanelBottom = RoundedCornerShape(bottomStart = 18.dp, bottomEnd = 18.dp)

/** Barre de titre du panneau d'un dossier : trois pastilles macOS, nom du dossier, nombre d'œuvres. */
@Composable
private fun FolderTitleBar(name: String, total: Int, kind: String) {
    Row(
        Modifier.fillMaxWidth().background(PanelBar, PanelTop).padding(horizontal = 22.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        listOf(0xFFFF5F57, 0xFFFEBC2E, 0xFF28C840).forEach { c ->
            Box(Modifier.padding(end = 8.dp).size(12.dp).background(Color(c).copy(alpha = 0.85f), CircleShape))
        }
        Text(name, color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 28.sp, modifier = Modifier.padding(start = 10.dp))
        Text(
            "   $total ${if (kind == "movie") "film" else "série"}${if (total > 1) "s" else ""}",
            color = Aurora.Ash, fontSize = 15.sp,
        )
    }
}

@Composable
private fun PosterGrid(client: AuroraClient, items: List<Card>, open: (Card) -> Unit) {
    if (items.isEmpty()) {
        Message("Rien ici pour l'instant.")
        return
    }
    LazyVerticalGrid(
        columns = GridCells.Fixed(7),
        contentPadding = PaddingValues(start = 48.dp, end = 48.dp, top = 12.dp, bottom = 48.dp),
        horizontalArrangement = Arrangement.spacedBy(20.dp),
        verticalArrangement = Arrangement.spacedBy(22.dp),
    ) {
        items(items, key = { it.id }) { c -> PosterCard(c, client, onClick = { open(c) }, width = 200.dp) }
    }
}

// --- Recherche : champ seul dans son panneau (jamais dans une rangée) -----------------------------

@Composable
private fun SearchTab(client: AuroraClient, open: (Card) -> Unit) {
    var query by rememberSaveable { mutableStateOf("") }
    val results by produceState<Load<SearchResults>?>(null, query) {
        if (query.trim().length < 2) { value = null; return@produceState }
        delay(300) // anti-rebond : on attend la fin de la saisie
        value = load { client.search(query.trim()) }
    }
    Column(Modifier.fillMaxSize().padding(top = 8.dp)) {
        Box(Modifier.padding(horizontal = 48.dp).width(640.dp)) {
            Field(query, { query = it }, "Un film, une série, un acteur…")
        }
        when (val r = results) {
            null -> Message("Tapez au moins deux lettres.")
            Load.Loading -> Message("Recherche…")
            is Load.Failed -> Message(r.message)
            is Load.Ok -> {
                val all = r.value.movies + r.value.series + r.value.episodes
                if (all.isEmpty()) Message("Aucun résultat pour « ${query.trim()} ».")
                else LazyColumn(contentPadding = PaddingValues(top = 20.dp, bottom = 48.dp), verticalArrangement = Arrangement.spacedBy(24.dp)) {
                    if (r.value.fuzzy) item { Text("Résultats approchants :", color = Aurora.Ash, fontSize = 15.sp, modifier = Modifier.padding(start = 48.dp)) }
                    if (r.value.movies.isNotEmpty()) item { Rail("Films", r.value.movies, client, wide = false, onOpen = open) }
                    if (r.value.series.isNotEmpty()) item { Rail("Séries", r.value.series, client, wide = false, onOpen = open) }
                    if (r.value.episodes.isNotEmpty()) item { Rail("Épisodes", r.value.episodes, client, wide = true, onOpen = open) }
                }
            }
        }
    }
}

// --- Profil ---------------------------------------------------------------------------------------

@Composable
private fun ProfileTab(vm: AppViewModel) {
    Column(Modifier.padding(horizontal = 48.dp, vertical = 24.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text(vm.me?.username.orEmpty(), color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 40.sp)
        Text("${vm.server?.name.orEmpty()} — ${vm.server?.baseUrl.orEmpty()}", color = Aurora.Ash, fontSize = 17.sp)
        Text(
            "Lecture : ${vm.caps.video.joinToString(", ").uppercase()} · audio ${vm.caps.audio.joinToString(", ").uppercase()} · jusqu'à ${vm.caps.maxHeight}p",
            color = Aurora.Ash, fontSize = 14.sp,
        )
        Spacer(Modifier.height(10.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            ActionButton("Changer de Synology", onClick = { vm.switchServer() })
            ActionButton("Ajouter un Synology", onClick = { vm.push(Screen.AddServer) })
            ActionButton("Se déconnecter", onClick = { vm.logout() })
        }
        Text(
            "Les préférences (langue audio, sous-titres, enchaînement des épisodes) se règlent dans « Mon profil » de l'interface web d'AURORA ; la box les applique.",
            color = Aurora.Ash, fontSize = 14.sp, modifier = Modifier.padding(top = 18.dp).width(900.dp),
        )
    }
}

@Composable
fun Message(text: String) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Text(text, color = Aurora.Mist, fontSize = 18.sp, modifier = Modifier.width(820.dp))
    }
}

/** Confirmation du retrait d'une œuvre de « Continuer à regarder » (fenêtre modale : le focus y reste piégé). */
@Composable
private fun RemoveContinueDialog(card: Card, onDismiss: () -> Unit, onConfirm: suspend () -> String?) {
    val title = if (card.kind == "episode") card.seriesTitle ?: card.title else card.title
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    // Focus initial sur « Annuler » : l'appui long qui ouvre la fenêtre peut se terminer dessus (OK relâché).
    val first = remember { FocusRequester() }
    LaunchedEffect(Unit) { delay(80); runCatching { first.requestFocus() } }
    Dialog(onDismissRequest = onDismiss) {
        Column(
            Modifier.width(560.dp).background(Aurora.Coal, RoundedCornerShape(18.dp)).padding(28.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Text("Retirer de Continuer à regarder ?", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 26.sp)
            Text(
                "« $title » ne sera plus proposé ici. Le fichier n'est pas supprimé et la progression est gardée ; " +
                    "il reviendra si vous le relancez.",
                color = Aurora.Mist, fontSize = 16.sp, lineHeight = 22.sp,
            )
            error?.let { Text(it, color = Aurora.Danger, fontSize = 15.sp) }
            Row(Modifier.padding(top = 8.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                ActionButton(if (busy) "Retrait…" else "Retirer", primary = true, onClick = {
                    if (busy) return@ActionButton
                    busy = true
                    scope.launch {
                        error = onConfirm()
                        busy = false
                    }
                })
                ActionButton("Annuler", onClick = onDismiss, modifier = Modifier.focusRequester(first))
            }
        }
    }
}
