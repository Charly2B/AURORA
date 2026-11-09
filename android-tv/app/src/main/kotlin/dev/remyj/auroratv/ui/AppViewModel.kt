package dev.remyj.auroratv.ui

import android.app.Application
import android.os.Build
import android.os.storage.StorageManager
import android.os.storage.StorageVolume
import androidx.core.content.ContextCompat
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dev.remyj.auroratv.AppContainer
import dev.remyj.auroratv.data.Server
import dev.remyj.auroratv.net.ApiException
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.Card
import dev.remyj.auroratv.net.Caps
import dev.remyj.auroratv.net.LinkStart
import dev.remyj.auroratv.net.LibraryRef
import dev.remyj.auroratv.net.Me
import dev.remyj.auroratv.net.Sessions
import dev.remyj.auroratv.player.Capabilities
import dev.remyj.auroratv.ui.components.CardFocusMemory
import kotlinx.coroutines.launch
import kotlinx.coroutines.ensureActive
import kotlin.coroutines.coroutineContext
import java.util.UUID
import kotlin.random.Random

/** Écrans de l'application (pile de navigation). */
sealed interface Screen {
    data object Loading : Screen
    data object Servers : Screen
    data object AddServer : Screen
    data class Login(val server: Server) : Screen
    data object Main : Screen
    data class Detail(val itemId: Long) : Screen
    data class Player(val itemId: Long, val fileId: Long? = null, val restart: Boolean = false) : Screen
    data class UsbPlayer(val uri: String, val title: String) : Screen
}

enum class Tab(val label: String) { HOME("Accueil"), MOVIES("Films"), SERIES("Séries"), USB("Disque USB"), SEARCH("Recherche"), PROFILE("Profil") }

class AppViewModel(private val application: Application, private val container: AppContainer) : ViewModel() {

    private val stack = mutableStateListOf<Screen>(Screen.Loading)
    val screen: Screen get() = stack.last()
    val canGoBack: Boolean get() = stack.size > 1
    /** Écrans de la pile (le plus ancien d'abord). */
    val screens: List<Screen> get() = stack.toList()

    /** Dernière carte focalisée de l'accueil/bibliothèques : le focus y revient au retour d'une fiche. */
    val cardFocus = CardFocusMemory()

    var tab by mutableStateOf(Tab.HOME)
    var server by mutableStateOf<Server?>(null)
        private set
    var client by mutableStateOf<AuroraClient?>(null)
        private set
    var me by mutableStateOf<Me?>(null)
        private set
    var backgroundId by mutableStateOf<String?>(null)
        private set

    var librariesState by mutableStateOf<Load<List<LibraryRef>>>(Load.Loading)
        private set
    var libraryNavigation by mutableStateOf(LibraryNavigation())
        private set

    suspend fun loadLibraries(c: AuroraClient) {
        val result = load { c.libraries() }
        coroutineContext.ensureActive()
        if (client !== c) return
        librariesState = result
        if (result is Load.Ok) libraryNavigation = libraryNavigation.reconcile(result.value)
    }

    fun selectLibrary(kind: String, id: Long?) {
        val libraries = (librariesState as? Load.Ok)?.value.orEmpty()
        val selection = libraryNavigation.select(kind, id, libraries)
        val targetTab = if (kind == "movie") Tab.MOVIES else Tab.SERIES
        if (tab == targetTab && selection == libraryNavigation) return
        libraryNavigation = selection
        tab = targetTab
        cardFocus.last = null
        cardFocus.restore = null
    }

    /** Un ancien écran/client ne peut écrire l'apparence d'un nouveau NAS. */
    fun chooseBackground(c: AuroraClient, id: String?) {
        if (client !== c) return
        val target = server ?: return
        if (id != null && c.backgroundUrl(id) == null) return
        container.backgrounds.set(target.id, id)
        backgroundId = id
        tab = Tab.HOME
    }
    /** Incrémenté après une lecture : les écrans rechargent progression et « vu ». */
    var refresh by mutableStateOf(0)
        private set

    /** « Réessayer » des écrans en échec : relance accueil, dossiers, fiche et bibliothèques. */
    fun retry() {
        // Le bouton « Réessayer » disparaît quand le chargement réussit : le contenu reprend le focus.
        focusContent = true
        refresh++
    }

    /**
     * Le prochain chargement réussi doit poser le focus sur son premier élément. Demandé quand le
     * bouton qui avait le focus disparaît (Réessayer, « Voir les N films ») : sans cela la télécommande
     * restait sans cible.
     */
    var focusContent by mutableStateOf(false)
        private set

    /** Vrai une seule fois : le premier écran chargé consomme la demande. */
    fun takeFocusContent(): Boolean = focusContent.also { focusContent = false }

    /** L'utilisateur reprend la main (colonne de gauche) : une demande de focus en attente est périmée. */
    fun clearContentFocus() { focusContent = false }

    /** Demande le focus sur le contenu au prochain chargement (ex. « Voir les N films »). */
    fun requestContentFocus() { focusContent = true }

    /**
     * Entrée dans Recherche depuis la colonne : le champ prend le focus. Un compteur (et non un booléen)
     * rejoue la demande si l'on reclique sur « Rechercher » alors que l'onglet est déjà ouvert, sans
     * rejouer au retour d'une fiche.
     */
    var searchFocusRequest by mutableStateOf(0)
        private set
    private var searchFocusConsumed = 0

    fun requestSearchFocus() { searchFocusRequest++ }

    fun takeSearchFocus(): Boolean {
        val fresh = searchFocusRequest != searchFocusConsumed
        searchFocusConsumed = searchFocusRequest
        return fresh
    }

    /** Film du fond de l'accueil : tiré une fois par NAS connecté, gardé aux rechargements. */
    var featured by mutableStateOf<Card?>(null)
        private set
    /** Nombre de films / séries de toute la médiathèque (colonne de gauche). */
    val movieCount: Int? get() = (librariesState as? Load.Ok)?.value?.sumOf { it.movies }
    val seriesCount: Int? get() = (librariesState as? Load.Ok)?.value?.sumOf { it.series }
    private var featuredFor: AuroraClient? = null

    /** Tire le titre à la une dans toute la médiathèque ; en cas d'échec, le héros du serveur reste. */
    fun loadFeatured(c: AuroraClient, fallback: Card?) {
        if (client !== c) return
        if (featured == null) featured = fallback
        val libs = (librariesState as? Load.Ok)?.value ?: return
        if (featuredFor === c) return
        featuredFor = c
        featured = fallback ?: featured
        viewModelScope.launch {
            val picked = runCatching {
                val movies = libs.sumOf { it.movies }
                val series = libs.sumOf { it.series }
                if (client !== c) return@launch
                if (movies + series == 0) null
                else featuredSlot(movies, series, Random.nextInt(movies + series)).let { (kind, offset) ->
                    c.list(kind, null, "added", true, offset, 1).items.firstOrNull()
                }
            }.getOrNull() ?: fallback ?: featured
            if (client !== c) return@launch
            featured = picked
        }
    }

    /** Message affiché sur la liste des NAS (déclaré avant `init`, qui peut l'écrire immédiatement). */
    var serverError by mutableStateOf<String?>(null)

    /** Capacités de lecture de la box, mesurées une fois. */
    val caps: Caps = Capabilities.detect(application)

    /** Volumes USB amovibles montés (chaque partition = un volume). Détectés à chaud. */
    var usbVolumes by mutableStateOf<List<StorageVolume>>(emptyList())
        private set
    /** Volume USB sélectionné pour navigation, ou null. */
    var usbSelected by mutableStateOf<StorageVolume?>(null)
        private set
    /** Chemin de montage du volume sélectionné, ou null. */
    val usbPath: String? get() = usbSelected?.let { container.usb.path(it) }
    val usbLabel: String? get() = usbSelected?.let { container.usb.label(it) }
    val usbCapacity: String? get() = usbSelected?.let { container.usb.capacityText(it) }
    /** Vrai si l'autorisation « Tous les fichiers » est accordée (lecture directe possible). */
    val usbCanBrowse: Boolean get() = container.usb.canAccessAllFiles()

    fun usbVolumeLabel(volume: StorageVolume): String = container.usb.label(volume)

    /** Ouvre le navigateur sur ce volume. */
    fun selectUsbVolume(volume: StorageVolume) {
        usbSelected = volume
        tab = Tab.USB
    }

    /** Ouvre la source USB ; sélectionne la première partition si aucune n'est encore choisie. */
    fun openUsb() {
        tab = Tab.USB
        if (usbSelected == null) usbSelected = usbVolumes.firstOrNull()
    }

    /**
     * NAS enregistrés. État Compose (et non simple lecture des préférences) : sans cela, « Retirer »
     * laissait la ligne affichée jusqu'au prochain passage sur l'écran.
     */
    var servers by mutableStateOf(container.servers.servers())
        private set

    private fun reloadServers() { servers = container.servers.servers() }

    /** Client HTTP partagé (cookie de session) : sert aussi à l'association par code avant connexion. */
    val http get() = container.http

    // Demande d'association en cours par NAS : réutilisée tant qu'elle est valide. Sans cela, chaque passage par
    // l'écran de connexion en créait une nouvelle, et la 5e (limite du NAS par adresse) était refusée 10 min.
    private val links = mutableMapOf<String, Pair<LinkStart, Long>>()

    fun cachedLink(serverId: String): LinkStart? = links[serverId]
        ?.takeIf { (link, at) -> System.currentTimeMillis() - at < (link.expiresIn - 30) * 1000L }?.first

    fun rememberLink(serverId: String, link: LinkStart) { links[serverId] = link to System.currentTimeMillis() }
    fun forgetLink(serverId: String) { links.remove(serverId) }

    // `lazy` : StorageVolumeCallback n'existe qu'à partir de l'API 30 ; il ne doit pas être chargé sur
    // les box plus anciennes (les deux box ciblées sont en Android 11, mais minSdk 24 exige le garde).
    // Déclaré AVANT `init` : les initialiseurs et blocs `init` s'exécutent dans l'ordre de déclaration,
    // donc `observeUsb()` (appelé depuis `init`) doit trouver `usbCallback` déjà initialisé.
    private val usbCallback by lazy {
        object : StorageManager.StorageVolumeCallback() {
            override fun onStateChanged(volume: StorageVolume) { refreshUsb() }
        }
    }

    init {
        observeUsb()
        viewModelScope.launch { start() }
    }

    /** Détecte le disque USB branché/débranché (StorageManager) et maintient l'état Compose à jour. */
    private fun observeUsb() {
        refreshUsb()
        val sm = application.getSystemService(StorageManager::class.java) ?: return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            sm.registerStorageVolumeCallback(ContextCompat.getMainExecutor(application), usbCallback)
        }
    }

    private fun refreshUsb() {
        val vols = container.usb.mountedVolumes()
        usbVolumes = vols
        // Le volume sélectionné a disparu (débranché) : on l'oublie.
        if (usbSelected != null && vols.none { it.uuid != null && it.uuid == usbSelected?.uuid }) usbSelected = null
        if (vols.isEmpty()) usbSelected = null
    }

    override fun onCleared() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            runCatching {
                application.getSystemService(StorageManager::class.java)?.unregisterStorageVolumeCallback(usbCallback)
            }
        }
    }

    private suspend fun start() {
        val all = servers
        val last = all.firstOrNull { it.id == container.servers.lastServerId() } ?: all.firstOrNull()
        if (last == null) {
            reset(Screen.AddServer)
            return
        }
        if (!connect(last)) {
            reset(Screen.Servers)
            serverError = unreachable(last)
        }
    }

    private fun unreachable(target: Server) =
        "« ${target.name} » ne répond pas. Vérifiez que le NAS est allumé et que l'adresse ${target.baseUrl} est correcte."

    /** Ouvre un serveur avec son jeton enregistré. Renvoie false s'il faut se (re)connecter. */
    suspend fun connect(target: Server): Boolean {
        val token = container.servers.tokens(target.id).get() ?: run {
            reset(Screen.Servers)
            push(Screen.Login(target))
            return true
        }
        Sessions.set(target.baseUrl, token)
        val c = AuroraClient(container.http, target.baseUrl)
        serverError = null
        return try {
            val profile = c.me()
            me = profile
            // Nom du profil connu seulement après connexion (association par code) : affiché dans la liste des NAS.
            val known = if (target.username == profile.username) target
            else target.copy(username = profile.username).also { container.servers.save(it); reloadServers() }
            // Session révoquée plus tard (mot de passe changé, compte désactivé) : retour à la connexion.
            c.onUnauthorized = { viewModelScope.launch { if (client === c) sessionExpired(target) } }
            server = known
            client = c
            librariesState = Load.Loading
            libraryNavigation = libraryNavigation.forServer(target.id)
            featured = null
            featuredFor = null
            backgroundId = container.backgrounds.get(target.id)
            focusContent = false
            searchFocusConsumed = searchFocusRequest
            container.servers.setLastServer(target.id)
            tab = Tab.HOME
            cardFocus.last = null
            reset(Screen.Main)
            true
        } catch (e: ApiException) {
            if (e.code == 401) {
                container.servers.tokens(target.id).clear()
                reset(Screen.Servers)
                push(Screen.Login(target))
                true
            } else {
                false
            }
        }
    }

    fun openServer(target: Server) = viewModelScope.launch {
        if (!connect(target)) {
            reset(Screen.Servers)
            serverError = unreachable(target)
        }
    }

    private fun sessionExpired(target: Server) {
        container.servers.tokens(target.id).clear()
        Sessions.set(target.baseUrl, null)
        client = null
        me = null
        server = null
        reset(Screen.Servers)
        push(Screen.Login(target))
    }


    /** Ajout d'un NAS : vérifie qu'AURORA répond à cette adresse, puis passe à la connexion. */
    suspend fun addServer(baseUrl: String): String? {
        val c = AuroraClient(container.http, baseUrl)
        val status = try {
            c.status()
        } catch (e: ApiException) {
            return e.message ?: "Serveur injoignable"
        }
        if (!status.hasAdmin) return "AURORA est installé à cette adresse mais pas encore configuré : terminez l'assistant depuis un navigateur."
        // Même adresse déjà enregistrée : on la réutilise plutôt que de créer un doublon.
        val s = container.servers.servers().firstOrNull { it.baseUrl == baseUrl } ?: run {
            val host = baseUrl.substringAfter("://").substringBefore(':')
            Server(id = UUID.randomUUID().toString(), name = "Synology $host", baseUrl = baseUrl)
                .also { container.servers.save(it); reloadServers() }
        }
        // Retour depuis la connexion → liste des NAS (et non le formulaire d'ajout).
        reset(Screen.Servers)
        push(Screen.Login(s))
        return null
    }

    suspend fun login(target: Server, username: String, password: String): String? {
        val c = AuroraClient(container.http, target.baseUrl)
        return try {
            loginWithToken(target, c.login(username, password))
        } catch (e: ApiException) {
            e.message
        }
    }

    /** Jeton obtenu (mot de passe ou code d'association) : on l'enregistre et on ouvre le NAS. */
    suspend fun loginWithToken(target: Server, token: String): String? {
        container.servers.tokens(target.id).put(token)
        return if (connect(target)) null else unreachable(target)
    }

    fun logout() = viewModelScope.launch {
        client?.onUnauthorized = null
        client?.logout()
        server?.let {
            container.servers.tokens(it.id).clear()
            Sessions.set(it.baseUrl, null)
        }
        client = null
        me = null
        server = null
        reset(Screen.Servers)
    }

    fun removeServer(target: Server) {
        container.servers.remove(target.id)
        Sessions.set(target.baseUrl, null)
        reloadServers()
        if (container.servers.servers().isEmpty()) reset(Screen.AddServer) else reset(Screen.Servers)
    }

    fun switchServer() = reset(Screen.Servers)

    fun push(s: Screen) { stack.add(s) }

    fun back(): Boolean {
        if (stack.size <= 1) return false
        val leaving = stack.removeAt(stack.lastIndex)
        if (leaving is Screen.Player) refresh++
        return true
    }

    /** Remplace l'écran courant (ex. épisode suivant) sans empiler. */
    fun replace(s: Screen) {
        stack[stack.lastIndex] = s
    }

    private fun reset(s: Screen) {
        stack.clear()
        stack.add(s)
    }
}
