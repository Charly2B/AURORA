package dev.remyj.auroratv.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
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
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.saveable.rememberSaveableStateHolder
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.ui.focus.onFocusChanged
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
import dev.remyj.auroratv.ui.components.Rail
import dev.remyj.auroratv.ui.theme.Aurora
import kotlinx.coroutines.delay
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.tv.material3.ClickableSurfaceDefaults
import androidx.tv.material3.Surface
import dev.remyj.auroratv.net.Detail
import dev.remyj.auroratv.ui.components.Artwork
import dev.remyj.auroratv.ui.components.Eyebrow
import dev.remyj.auroratv.ui.components.SegmentBar
import androidx.compose.foundation.layout.heightIn
import coil3.request.ImageRequest
import coil3.compose.LocalPlatformContext
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.withStyle
import dev.remyj.auroratv.ui.components.BoxBlur
import coil3.request.transformations
import androidx.compose.ui.graphics.FilterQuality
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect

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
    // Tri, requête et défilement propres à chaque onglet : sans ceci, passer par Recherche puis revenir
    // sur Films remettait le tri par défaut et perdait la position.
    val tabs = rememberSaveableStateHolder()
    // Accueil chargé une fois ici pour le fond et le panneau de bienvenue.
    val homeState by produceState<Load<Home>>(Load.Loading, client, vm.refresh) { value = load { client.home() } }
    LaunchedEffect(client, vm.refresh) { vm.loadLibraries(client) }
    LaunchedEffect(client, homeState, vm.librariesState) {
        val home = (homeState as? Load.Ok)?.value
        vm.loadFeatured(client, home?.hero ?: home?.sections?.firstOrNull()?.items?.firstOrNull())
    }
    Box(Modifier.fillMaxSize().background(Aurora.Ink)) {
        if (vm.tab == Tab.HOME && homeState is Load.Ok) HomeBackground(vm, client)
        Row(Modifier.fillMaxSize()) {
            Sidebar(vm, restoring)
            Box(Modifier.fillMaxSize()) {
                // Un état « saveable » par onglet : tri, recherche et défilement y survivent.
                tabs.SaveableStateProvider(vm.tab.name) {
                    when (vm.tab) {
                        Tab.HOME -> HomeTab(homeState, client, open,
                            onRetry = vm::retry, consumeFocus = vm::takeFocusContent, clearFocus = vm::clearContentFocus)
                        Tab.MOVIES -> LibraryTab(vm, client, "movie", open)
                        Tab.SERIES -> LibraryTab(vm, client, "series", open)
                        Tab.USB -> UsbScreen(vm)
                        Tab.SEARCH -> SearchTab(vm, client, open)
                        Tab.PROFILE -> key(client) { ProfileTab(vm, client) }
                    }
                }
            }
        }
    }
}

/**
 * Colonne de gauche « Cinémathèque », comme sur Mac : marque, Recherche, Films/Séries (avec les nombres de films et
 * de séries), Profil en pied. ◀ depuis le contenu y revient, ▶ en ressort.
 */
@Composable
private fun Sidebar(vm: AppViewModel, restoring: Boolean) {
    val first = remember { FocusRequester() }
    var sidebarFocused by remember { mutableStateOf(false) }
    val listState = rememberLazyListState()
    val scope = rememberCoroutineScope()
    var preview by remember { mutableStateOf<Job?>(null) }
    val navigation = vm.libraryNavigation
    val libraries = (vm.librariesState as? Load.Ok)?.value.orEmpty()
    val movies = foldersFor(libraries, "movie")
    val series = foldersFor(libraries, "series")
    val activeKind = when (vm.tab) { Tab.MOVIES -> "movie"; Tab.SERIES -> "series"; else -> null }
    val activeFolders = if (activeKind == "movie") movies else series
    val activeId = activeKind?.let(navigation::selected)
    val folderOffset = activeFolders.indexOfFirst { it.id == activeId }.takeIf { it >= 0 } ?: activeFolders.size
    val usbHeaderIndex = 5 + movies.size + series.size
    val activeIndex = when (vm.tab) {
        Tab.HOME -> 0
        Tab.MOVIES -> 2 + folderOffset
        Tab.SERIES -> 4 + movies.size + folderOffset
        Tab.USB -> {
            val selectedIdx = vm.usbVolumes.indexOfFirst { it.uuid != null && it.uuid == vm.usbSelected?.uuid }
            usbHeaderIndex + (if (selectedIdx >= 0) selectedIdx + 1 else 0)
        }
        else -> null
    }
    // Sélection au focus conservée, mais différée de 200 ms : traverser dix dossiers pour atteindre
    // Profil ne lance plus dix chargements de bibliothèque, et le dernier survolé gagne.
    // Quitter la colonne (▶ vers le contenu) applique tout de suite le dossier survolé, sinon on entrait
    // dans le contenu du dossier précédent.
    var pending by remember { mutableStateOf<Pair<String, Long?>?>(null) }
    val previewLibrary: (String, Long?) -> Unit = { kind, id ->
        preview?.cancel()
        pending = kind to id
        preview = scope.launch {
            delay(200)
            pending = null
            vm.selectLibrary(kind, id)
        }
    }
    // L'en-tête d'une catégorie garde le dossier déjà choisi ; avant, le simple passage le remettait
    // sur « Tous les dossiers » et relançait un chargement complet.
    val cancelPreview: () -> Unit = { preview?.cancel(); pending = null }
    val openCategory: (String) -> Unit = { kind ->
        cancelPreview()
        vm.selectLibrary(kind, navigation.selected(kind))
    }
    LaunchedEffect(sidebarFocused) {
        if (!sidebarFocused) {
            preview?.cancel()
            pending?.let { (kind, id) -> pending = null; vm.selectLibrary(kind, id) }
        } else {
            // La colonne a le focus : une demande de focus différée sur le contenu n'a plus lieu d'être.
            vm.clearContentFocus()
        }
    }
    LaunchedEffect(vm.tab) {
        // Le clic de la télécommande garde sa ligne ; seuls l'entrée et le retour sans carte déplacent le focus.
        if (sidebarFocused) return@LaunchedEffect
        if (restoring && vm.cardFocus.restore != null) {
            delay(1_500)
            if (vm.cardFocus.restore == null) return@LaunchedEffect
            vm.cardFocus.restore = null
        } else {
            delay(100)
        }
        activeIndex?.let { listState.scrollToItem(it) }
        // La ligne visée peut n'être composée qu'après le défilement : on réessaie plutôt que
        // d'avaler l'échec et de laisser la télécommande sans focus.
        repeat(10) {
            if (runCatching { first.requestFocus() }.getOrDefault(false)) return@LaunchedEffect
            delay(60)
        }
    }
    val focusIf: (Tab) -> Modifier = { t -> if (t == vm.tab) Modifier.focusRequester(first) else Modifier }
    Column(
        Modifier.width(212.dp).fillMaxHeight()
            .background(Brush.horizontalGradient(if (vm.tab == Tab.HOME) {
                listOf(Aurora.Night.copy(alpha = 0.74f), Aurora.Night.copy(alpha = 0.55f))
            } else listOf(Aurora.Night, Aurora.Night)))
            .padding(horizontal = 14.dp, vertical = 18.dp)
            .onFocusChanged { sidebarFocused = it.hasFocus }.focusRestorer().focusGroup(),
    ) {
        Text("AURORA", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 22.sp, letterSpacing = 3.sp, modifier = Modifier.padding(start = 8.dp))
        Box(Modifier.padding(start = 8.dp, top = 6.dp, bottom = 5.dp).width(28.dp).height(1.dp).background(Aurora.Accent))
        Text("CINÉMATHÈQUE PRIVÉE", color = Aurora.Ash, fontSize = 10.sp, letterSpacing = 1.5.sp, modifier = Modifier.padding(start = 8.dp))
        Spacer(Modifier.height(16.dp))
        NavItem(
            "Rechercher", vm.tab == Tab.SEARCH,
            // Le champ prend le focus à l'entrée (et si l'on reclique sur l'onglet), pas au retour d'une fiche.
            onClick = { cancelPreview(); vm.requestSearchFocus(); vm.tab = Tab.SEARCH },
            modifier = focusIf(Tab.SEARCH), boxed = true, onFocused = cancelPreview,
        )
        Spacer(Modifier.height(14.dp))
        LazyColumn(state = listState, modifier = Modifier.weight(1f).fillMaxWidth(), contentPadding = PaddingValues(bottom = 6.dp)) {
            item(key = "home") {
                NavItem("Accueil", vm.tab == Tab.HOME,
                    onClick = { cancelPreview(); vm.tab = Tab.HOME },
                    modifier = focusIf(Tab.HOME), onFocused = cancelPreview)
            }
            listOf("movie" to Tab.MOVIES, "series" to Tab.SERIES).forEach { (kind, tab) ->
                val active = vm.tab == tab
                val folders = if (kind == "movie") movies else series
                item(key = "$kind-header") {
                    Box(Modifier.sidebarGroupFrame(active, first = true)
                        .background(Aurora.Paper.copy(alpha = 0.08f), RoundedCornerShape(topStart = 8.dp, topEnd = 8.dp))
                        .padding(horizontal = 2.dp)) {
                        NavItem(
                            // Jamais « sélectionné » : le dossier (ou « Tous les dossiers ») porte le repère.
                            tab.label, false, onClick = { openCategory(kind) },
                            onFocused = { openCategory(kind) },
                            count = if (kind == "movie") vm.movieCount else vm.seriesCount,
                            windowTitle = true,
                        )
                    }
                }
                items(folders, key = { "$kind-${it.id}" }) { library ->
                    val selected = active && navigation.selected(kind) == library.id
                    Box(Modifier.sidebarGroupFrame(active).padding(horizontal = 2.dp)) {
                        NavItem(library.name, selected, onClick = { cancelPreview(); vm.selectLibrary(kind, library.id) },
                            onFocused = { previewLibrary(kind, library.id) },
                            modifier = if (selected) Modifier.focusRequester(first) else Modifier,
                            count = libraryCount(library, kind), folder = true)
                    }
                }
                item(key = "$kind-all") {
                    val selected = active && navigation.selected(kind) == null
                    Box(Modifier.padding(bottom = 10.dp).sidebarGroupFrame(active, last = true)
                        .padding(start = 2.dp, end = 2.dp, bottom = 5.dp)) {
                        NavItem("Tous les dossiers", selected, onClick = { cancelPreview(); vm.selectLibrary(kind, null) },
                            onFocused = { previewLibrary(kind, null) },
                            modifier = if (selected) Modifier.focusRequester(first) else Modifier, folder = true)
                    }
                }
            }

            // Disque USB : source locale (partitions du disque branché sur le Formovie), sous les dossiers du NAS.
            item(key = "usb-header") {
                Box(Modifier.sidebarGroupFrame(vm.tab == Tab.USB, first = true)
                    .background(Aurora.Paper.copy(alpha = 0.08f), RoundedCornerShape(topStart = 8.dp, topEnd = 8.dp))
                    .padding(horizontal = 2.dp)) {
                    NavItem("Disque USB", false, onClick = { cancelPreview(); vm.openUsb() },
                        onFocused = { cancelPreview(); vm.openUsb() },
                        modifier = if (vm.usbVolumes.isEmpty()) focusIf(Tab.USB) else Modifier,
                        windowTitle = true)
                }
            }
            if (vm.usbVolumes.isEmpty()) {
                item(key = "usb-empty") {
                    Box(Modifier.padding(bottom = 10.dp).sidebarGroupFrame(vm.tab == Tab.USB, last = true)
                        .padding(start = 2.dp, end = 2.dp, bottom = 5.dp)) {
                        Text("Aucun disque branché", color = Aurora.Ash.copy(alpha = 0.6f), fontSize = 11.sp,
                            modifier = Modifier.padding(start = 25.dp, top = 7.dp, bottom = 8.dp))
                    }
                }
            } else {
                items(vm.usbVolumes, key = { "usb-${it.uuid}" }) { volume ->
                    val selected = vm.tab == Tab.USB && volume.uuid != null && volume.uuid == vm.usbSelected?.uuid
                    val last = volume == vm.usbVolumes.last()
                    Box(Modifier.sidebarGroupFrame(vm.tab == Tab.USB, last = last)
                        .padding(start = 2.dp, end = 2.dp, bottom = if (last) 5.dp else 0.dp)) {
                        NavItem(vm.usbVolumeLabel(volume), selected, onClick = { cancelPreview(); vm.selectUsbVolume(volume) },
                            onFocused = { cancelPreview(); vm.selectUsbVolume(volume) },
                            modifier = if (selected) Modifier.focusRequester(first) else Modifier, folder = true, led = true)
                    }
                }
            }

        }
        Box(Modifier.fillMaxWidth().height(1.dp).background(Aurora.Line))
        NavItem(
            "Profil", vm.tab == Tab.PROFILE,
            onClick = { cancelPreview(); vm.tab = Tab.PROFILE },
            modifier = focusIf(Tab.PROFILE).padding(top = 8.dp),
            subtitle = vm.server?.name,
            onFocused = cancelPreview,
        )
    }
}

/** Bord continu découpé par ligne : même un groupe plus haut que l'écran reste paresseux et navigable. */
private fun Modifier.sidebarGroupFrame(active: Boolean, first: Boolean = false, last: Boolean = false): Modifier = drawBehind {
    val radius = 8.dp.toPx()
    val stroke = 1.dp.toPx()
    val top = if (first) stroke / 2 else -radius
    val bottom = if (last) size.height - stroke / 2 else size.height + radius
    clipRect {
        val origin = Offset(stroke / 2, top)
        val bounds = Size(size.width - stroke, bottom - top)
        drawRoundRect(Aurora.Paper.copy(alpha = 0.035f), origin, bounds, CornerRadius(radius))
        drawRoundRect(if (active) Aurora.Accent else Aurora.Paper.copy(alpha = 0.65f),
            origin, bounds, CornerRadius(radius), style = Stroke(stroke))
    }
}

@Composable
private fun NavItem(
    label: String, selected: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier,
    count: Int? = null, subtitle: String? = null, boxed: Boolean = false, folder: Boolean = false, windowTitle: Boolean = false,
    led: Boolean = false,
    onFocused: (() -> Unit)? = null,
) {
    Surface(
        onClick = onClick,
        shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(6.dp)),
        colors = ClickableSurfaceDefaults.colors(
            containerColor = when {
                selected -> Aurora.Accent.copy(alpha = 0.17f)
                boxed -> Color(0x0FF4EAD8)
                else -> Color.Transparent
            },
            contentColor = if (selected) Aurora.Paper else Aurora.Mist,
            focusedContainerColor = Aurora.Accent,
            focusedContentColor = Aurora.Ink,
        ),
        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.03f),
        modifier = modifier.fillMaxWidth().padding(vertical = 1.dp)
            .onFocusChanged { if (it.isFocused) onFocused?.invoke() },
    ) {
        Row(Modifier.height(IntrinsicSize.Min), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.width(3.dp).fillMaxHeight().background(if (selected) Aurora.Accent else Color.Transparent))
            Column(Modifier.weight(1f).padding(start = if (folder) 25.dp else 9.dp, end = 10.dp, top = 7.dp, bottom = 7.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    if (windowTitle) {
                        Row(horizontalArrangement = Arrangement.spacedBy(3.dp), modifier = Modifier.padding(end = 7.dp)) {
                            listOf(Aurora.Danger, Aurora.Ember, Aurora.Seen).forEach { color ->
                                Box(Modifier.size(5.dp).background(color, CircleShape))
                            }
                        }
                    }
                    if (led) {
                        Box(Modifier.padding(end = 7.dp).size(6.dp).background(Aurora.Seen, CircleShape))
                    }
                    Text(label, fontFamily = if (folder) null else Aurora.Display, fontSize = if (folder) 12.sp else 17.sp,
                        maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f))
                    if (count != null) Text("$count", fontSize = if (folder) 11.sp else 13.sp, modifier = Modifier.padding(start = 4.dp))
                }
                if (subtitle != null) Text(subtitle, fontSize = 11.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
    }
}

// --- Accueil --------------------------------------------------------------------------------------

/** Une seule composition plein écran : le fond reste continu sous la colonne et le contenu. */
@Composable
private fun HomeBackground(vm: AppViewModel, client: AuroraClient) {
    Box(Modifier.fillMaxSize()) {
        vm.featured?.let { card ->
            AsyncImage(
                model = ImageRequest.Builder(LocalPlatformContext.current).data(client.absolute(card.backdrop ?: card.thumb))
                    .size(160, 90).transformations(BoxBlur()).build(),
                contentDescription = null, contentScale = ContentScale.Crop, alpha = 0.9f, filterQuality = FilterQuality.High,
                modifier = Modifier.fillMaxSize(),
            )
        }
        // L'image automatique reste dessous : erreur ou chargement du fond personnel = repli immédiat.
        val backgroundId = vm.backgroundId
        var backgroundFailed by remember(client, backgroundId, vm.refresh) { mutableStateOf(false) }
        if (backgroundId != null && !backgroundFailed) {
            AsyncImage(
                model = ImageRequest.Builder(LocalPlatformContext.current).data(client.backgroundUrl(backgroundId))
                    .size(1920, 1080).build(),
                contentDescription = null, contentScale = ContentScale.Crop,
                onError = { backgroundFailed = true; client.backgroundImageFailed(it.result.throwable) },
                modifier = Modifier.fillMaxSize(),
            )
        }
        Box(Modifier.fillMaxSize().background(Brush.radialGradient(0f to Aurora.Ink.copy(alpha = 0.1f), 1f to Aurora.Ink.copy(alpha = 0.6f), radius = 1300f)))
    }
}

/**
 * Accueil : le panneau « Bienvenue » puis les rangées du serveur (Continuer à regarder, prochain épisode,
 * ajouts récents, favoris…), dans l'ordre des préférences du profil. Avant, l'onglet ne contenait qu'un
 * panneau non focusable : ▶ ne menait nulle part et il fallait passer par Films → dossier → affiche.
 */
@Composable
private fun HomeTab(
    state: Load<Home>, client: AuroraClient, open: (Card) -> Unit,
    onRetry: () -> Unit, consumeFocus: () -> Boolean, clearFocus: () -> Unit,
) {
    var autoFocus by remember { mutableStateOf(false) }
    // Après un « Réessayer », la première affiche reprend le focus (le bouton a disparu) ; en cas d'échec,
    // la demande est abandonnée pour ne pas resurgir plus tard sur un autre écran.
    LaunchedEffect(state) {
        when (state) {
            is Load.Ok -> autoFocus = consumeFocus()
            is Load.Failed -> clearFocus()
            else -> Unit
        }
    }
    when (state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> ErrorMessage(state.message, onRetry)
        is Load.Ok -> {
            val sections = homeRails(state.value.sections)
            LazyColumn(
                Modifier.fillMaxSize(),
                contentPadding = PaddingValues(top = 18.dp, bottom = 48.dp),
                verticalArrangement = Arrangement.spacedBy(18.dp),
            ) {
                item(key = "welcome") { WelcomePanel() }
                items(sections, key = { it.id }) { s ->
                    Rail(
                        s.title, s.items, client,
                        wide = false,
                        onOpen = open,
                        autoFocusFirst = autoFocus && s == sections.first(),
                        onAutoFocused = { autoFocus = false },
                    )
                }
            }
        }
    }
}

/** Panneau verre « Bienvenue sur AURORA » : compact, pour laisser la place aux rangées. */
@Composable
private fun WelcomePanel() {
    Box(
        Modifier.padding(start = 32.dp).width(430.dp).height(126.dp)
            // Pas d'ombre : elle se verrait à travers le verre translucide (rectangle sombre).
            .background(Brush.linearGradient(listOf(Color(0x33FFFFFF), Color(0x12FFFFFF), Color(0x0AFFFFFF))), GlassShape)
            .border(1.dp, Color(0x47FFFFFF), GlassShape),
        contentAlignment = Alignment.Center,
    ) {
        // Reflet sur la moitié haute du verre.
        Box(Modifier.align(Alignment.TopCenter).fillMaxWidth().height(63.dp).clip(GlassShape).background(Brush.verticalGradient(listOf(Color(0x1AFFFFFF), Color.Transparent))))
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                buildAnnotatedString {
                    append("Bienvenue sur ")
                    withStyle(SpanStyle(fontWeight = FontWeight.SemiBold, letterSpacing = 4.sp)) { append("AURORA") }
                },
                color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 30.sp,
            )
            Box(Modifier.padding(top = 11.dp).width(32.dp).height(1.dp).background(Aurora.Accent))
        }
    }
}

private val GlassShape = RoundedCornerShape(18.dp)

@Composable
fun Badge(text: String) {
    Text(
        text.uppercase(), color = Aurora.Accent, fontSize = 11.sp, fontWeight = FontWeight.Bold,
        modifier = Modifier.border(1.dp, Aurora.Accent.copy(alpha = 0.55f), RoundedCornerShape(3.dp)).padding(horizontal = 5.dp, vertical = 1.dp),
    )
}

// --- Films / Séries : sélection partagée avec la colonne, tris conservés ---------------------------

@Composable
private fun LibraryTab(vm: AppViewModel, client: AuroraClient, kind: String, open: (Card) -> Unit) {
    val libsState = vm.librariesState
    val libs = foldersFor((libsState as? Load.Ok)?.value.orEmpty(), kind)
    val selected = vm.libraryNavigation.selected(kind)
    var sort by rememberSaveable(kind) { mutableStateOf(if (kind == "movie") "added" else "title") }
    Column(Modifier.fillMaxSize()) {
        Text(if (kind == "movie") "Films" else "Séries", color = Aurora.Paper, fontFamily = Aurora.Display,
            fontSize = 30.sp, modifier = Modifier.padding(start = 32.dp, top = 20.dp))
        val sorts = if (kind == "movie") listOf("added" to "Ajout récent", "title" to "Titre", "year" to "Année", "rating" to "Note")
        else listOf("title" to "Titre", "added" to "Ajout récent", "year" to "Année")
        LazyRow(
            contentPadding = PaddingValues(start = 32.dp, end = 32.dp, top = 12.dp, bottom = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            item(key = "sorts") {
                SegmentBar { sorts.forEach { (id, label) -> Chip(label, selected = sort == id, onClick = { sort = id }) } }
            }
        }
        Text(libs.firstOrNull { it.id == selected }?.name ?: "Tous les dossiers", color = Aurora.Ash,
            fontFamily = Aurora.Display, fontSize = 19.sp, maxLines = 1, overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(start = 32.dp, end = 32.dp, top = 4.dp, bottom = 6.dp))
        // Un autre NAS possède son propre état ; changer de dossier garde l'ancienne grille affichée
        // pendant le chargement (plus de « Chargement… » à chaque survol de la colonne).
        key(client, kind) {
            when {
                libsState is Load.Loading -> Message("Chargement…")
                libsState is Load.Failed -> ErrorMessage(libsState.message, vm::retry)
                libs.isEmpty() -> Message(if (kind == "movie") "Aucun film." else "Aucune série.")
                selected == null -> GroupedGrid(vm, client, kind, libs, sort, open)
                else -> FolderGrid(vm, client, kind, selected, sort, open)
            }
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
    val state by produceState<Load<ListPage>>(Load.Loading, client, kind, libraryId, sort, vm.refresh) {
        // Pas de retour à « Chargement… » : la grille précédente reste visible jusqu'à la nouvelle page.
        value = load { listAll(client, kind, libraryId, sort) }
    }
    var autoFocus by remember { mutableStateOf(false) }
    // Une grille neuve par dossier : ouvrir un dossier repart du haut. La donnée déjà chargée reste
    // affichée pendant la requête (pas de « Chargement… » entre deux dossiers).
    LaunchedEffect(state) {
        when {
            state is Load.Ok -> autoFocus = vm.takeFocusContent()
            state is Load.Failed -> vm.clearContentFocus()
        }
    }
    when (val s = state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> ErrorMessage(s.message, vm::retry)
        is Load.Ok -> key(libraryId) {
            PosterGrid(client, s.value.items, open, autoFocusFirst = autoFocus, onAutoFocused = { autoFocus = false })
        }
    }
}

/** « Tous les dossiers » : titre du dossier, puis ses œuvres ; jamais mélangés. */
@Composable
private fun GroupedGrid(vm: AppViewModel, client: AuroraClient, kind: String, libs: List<LibraryRef>, sort: String, open: (Card) -> Unit) {
    val state by produceState<Load<List<Pair<LibraryRef, ListPage>>>>(Load.Loading, client, kind, libs, sort, vm.refresh) {
        // Une page par dossier : « Tous les dossiers » n'a plus besoin de rapatrier toute la bibliothèque
        // pour s'afficher. La suite d'un dossier s'obtient en l'ouvrant (toutes ses pages).
        value = load { libs.map { lib -> lib to client.list(kind, lib.id, sort, sort in sortDesc, 0, PAGE) } }
    }
    var autoFocus by remember { mutableStateOf(false) }
    LaunchedEffect(state) {
        when {
            state is Load.Ok -> autoFocus = vm.takeFocusContent()
            state is Load.Failed -> vm.clearContentFocus()
        }
    }
    when (val s = state) {
        Load.Loading -> Message("Chargement…")
        is Load.Failed -> ErrorMessage(s.message, vm::retry)
        // Un panneau façon fenêtre macOS par dossier : barre de titre (pastilles, nom, nombre) puis ses œuvres
        // par rangées de 5. Le panneau est découpé en éléments de liste (barre, rangées, bas arrondi) pour rester
        // paresseux sur une grosse bibliothèque.
        is Load.Ok -> {
            val firstCard = s.value.firstOrNull { it.second.items.isNotEmpty() }
                ?.let { (lib, page) -> lib.id to page.items.firstOrNull()?.id }
            LazyColumn(
                contentPadding = PaddingValues(start = 32.dp, end = 32.dp, top = 12.dp, bottom = 48.dp),
            ) {
                s.value.filter { it.second.items.isNotEmpty() }.forEach { (lib, page) ->
                    item(key = "h${lib.id}") { FolderTitleBar(lib.name, page.total, kind) }
                    val rows = page.items.chunked(5)
                    rows.forEachIndexed { i, row ->
                        item(key = "r${lib.id}-$i") {
                            val last = i == rows.lastIndex
                            val more = last && page.hasMore()
                            Column(
                                Modifier
                                    .fillMaxWidth()
                                    .background(PanelFill, if (last) PanelBottom else RectangleShape),
                            ) {
                                Row(
                                    Modifier
                                        .fillMaxWidth()
                                        .padding(start = 22.dp, end = 22.dp, top = if (i == 0) 20.dp else 10.dp,
                                            bottom = if (more) 12.dp else if (last) 24.dp else 10.dp),
                                    horizontalArrangement = Arrangement.spacedBy(22.dp),
                                ) {
                                    // Parts égales : 5 colonnes alignées d'une rangée à l'autre, dernière rangée complétée à vide.
                                    row.forEach { c ->
                                        PosterCard(
                                            c, client, onClick = { open(c) }, modifier = Modifier.weight(1f),
                                            focusKey = "folder/${lib.id}/${c.kind}${c.id}",
                                            autoFocus = autoFocus && lib.id == firstCard?.first && c.id == firstCard.second,
                                            onAutoFocused = { autoFocus = false },
                                        )
                                    }
                                    repeat(5 - row.size) { Spacer(Modifier.weight(1f)) }
                                }
                                if (more) {
                                    // Au-delà d'une page, on ouvre le dossier : la grille du dossier enchaîne les pages.
                                    Box(Modifier.padding(start = 22.dp, end = 22.dp, bottom = 20.dp)) {
                                        ActionButton(
                                            "Voir les ${page.total} ${if (kind == "movie") "films" else "séries"} de « ${lib.name} »",
                                            // Le bouton disparaît : la première affiche du dossier reprend le focus.
                                            onClick = { vm.requestContentFocus(); vm.selectLibrary(kind, lib.id) },
                                        )
                                    }
                                }
                            }
                        }
                    }
                    item(key = "s${lib.id}") { Spacer(Modifier.height(28.dp)) }
                }
            }
        }
    }
}

private val PanelFill = Color(0x14F4EAD8) // ~8 % de crème : le panneau se détache du fond brun
private val PanelBar = Color(0x21F4EAD8)
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
private fun PosterGrid(
    client: AuroraClient, items: List<Card>, open: (Card) -> Unit,
    autoFocusFirst: Boolean = false, onAutoFocused: () -> Unit = {},
) {
    if (items.isEmpty()) {
        Message("Rien ici pour l'instant.")
        return
    }
    LazyVerticalGrid(
        // Cinq affiches plus grandes, avec la même largeur sur la dernière rangée.
        columns = GridCells.Fixed(5),
        contentPadding = PaddingValues(start = 32.dp, end = 32.dp, top = 12.dp, bottom = 48.dp),
        horizontalArrangement = Arrangement.spacedBy(22.dp),
        verticalArrangement = Arrangement.spacedBy(28.dp),
    ) {
        items(items, key = { it.id }) { c ->
            PosterCard(
                c, client, onClick = { open(c) }, width = 200.dp,
                autoFocus = autoFocusFirst && c == items.first(), onAutoFocused = onAutoFocused,
            )
        }
    }
}

// --- Recherche : champ seul dans son panneau (jamais dans une rangée) -----------------------------

@Composable
private fun SearchTab(vm: AppViewModel, client: AuroraClient, open: (Card) -> Unit) {
    var query by rememberSaveable { mutableStateOf("") }
    var attempt by rememberSaveable { mutableStateOf(0) }
    val field = remember { FocusRequester() }
    // Entrer dans Recherche (depuis la colonne) place le curseur dans le champ ; au retour d'une fiche,
    // c'est la carte ouverte qui reprend le focus, pas le champ.
    LaunchedEffect(vm.searchFocusRequest) {
        if (!vm.takeSearchFocus()) return@LaunchedEffect
        delay(120)
        if (vm.tab != Tab.SEARCH) return@LaunchedEffect
        runCatching { field.requestFocus() }
    }
    val results by produceState<Load<SearchResults>?>(null, query, attempt) {
        if (query.trim().length < 2) { value = null; return@produceState }
        delay(300) // anti-rebond : on attend la fin de la saisie
        value = load { client.search(query.trim()) }
    }
    Column(Modifier.fillMaxSize().padding(top = 22.dp)) {
        Box(Modifier.padding(horizontal = 32.dp).width(640.dp)) {
            Field(query, { query = it }, "Un film, une série, un acteur…", Modifier.focusRequester(field))
        }
        when (val r = results) {
            null -> Message("Tapez au moins deux lettres.")
            Load.Loading -> Message("Recherche…")
            is Load.Failed -> ErrorMessage(r.message) { attempt++ }
            is Load.Ok -> {
                val all = r.value.movies + r.value.series + r.value.episodes
                if (all.isEmpty()) Message("Aucun résultat pour « ${query.trim()} ».")
                else LazyColumn(contentPadding = PaddingValues(top = 20.dp, bottom = 48.dp), verticalArrangement = Arrangement.spacedBy(24.dp)) {
                    if (r.value.fuzzy) item { Text("Résultats approchants :", color = Aurora.Ash, fontSize = 15.sp, modifier = Modifier.padding(start = 32.dp)) }
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
private fun ProfileTab(vm: AppViewModel, client: AuroraClient) {
    var pickerOpen by rememberSaveable(vm.server?.id) { mutableStateOf(false) }
    var restoreFocus by remember { mutableStateOf(false) }
    val backgroundButton = remember { FocusRequester() }
    BackHandler(enabled = pickerOpen) { pickerOpen = false; restoreFocus = true }
    if (pickerOpen) {
        BackgroundPicker(client, vm.backgroundId, onApply = { id ->
            vm.chooseBackground(client, id)
            pickerOpen = false
        })
        return
    }
    LaunchedEffect(restoreFocus) {
        if (restoreFocus) {
            delay(80)
            runCatching { backgroundButton.requestFocus() }
            restoreFocus = false
        }
    }
    Column(Modifier.padding(horizontal = 32.dp, vertical = 24.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text(vm.me?.username.orEmpty(), color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 40.sp)
        Text("${vm.server?.name.orEmpty()} — ${vm.server?.baseUrl.orEmpty()}", color = Aurora.Ash, fontSize = 17.sp)
        Text(
            "Lecture : ${vm.caps.video.joinToString(", ").uppercase()} · audio ${vm.caps.audio.joinToString(", ").uppercase()} · jusqu'à ${vm.caps.maxHeight}p",
            color = Aurora.Ash, fontSize = 14.sp,
        )
        Spacer(Modifier.height(10.dp))
        ActionButton("Fond d’écran", onClick = { pickerOpen = true }, modifier = Modifier.focusRequester(backgroundButton))
        FlowRow(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            ActionButton("Changer de Synology", onClick = { vm.switchServer() })
            ActionButton("Ajouter un Synology", onClick = { vm.push(Screen.AddServer) })
            ActionButton("Se déconnecter", onClick = { vm.logout() })
        }
        Text(
            "Les préférences (langue audio, sous-titres, enchaînement des épisodes) se règlent dans « Mon profil » de l'interface web d'AURORA ; la box les applique.",
            color = Aurora.Ash, fontSize = 14.sp, modifier = Modifier.padding(top = 18.dp).width(680.dp),
        )
    }
}

@Composable
fun Message(text: String) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Text(text, color = Aurora.Mist, fontSize = 18.sp, modifier = Modifier.width(820.dp))
    }
}

/**
 * Échec réseau ou serveur. Le texte seul laissait la télécommande sans cible (▶/OK sans effet) :
 * un bouton « Réessayer » prend le focus et relance le chargement de l'écran.
 */
@Composable
fun ErrorMessage(text: String, onRetry: () -> Unit) {
    val focus = remember { FocusRequester() }
    LaunchedEffect(Unit) { delay(120); runCatching { focus.requestFocus() } }
    Column(
        Modifier.fillMaxSize().padding(horizontal = 48.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(text, color = Aurora.Mist, fontSize = 18.sp, modifier = Modifier.width(820.dp))
        ActionButton("Réessayer", onClick = onRetry, primary = true, modifier = Modifier.padding(top = 18.dp).focusRequester(focus))
    }
}
