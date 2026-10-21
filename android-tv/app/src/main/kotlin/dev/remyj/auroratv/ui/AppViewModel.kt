package dev.remyj.auroratv.ui

import android.app.Application
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
import dev.remyj.auroratv.net.Caps
import dev.remyj.auroratv.net.LinkStart
import dev.remyj.auroratv.net.Me
import dev.remyj.auroratv.net.Sessions
import dev.remyj.auroratv.player.Capabilities
import dev.remyj.auroratv.ui.components.CardFocusMemory
import kotlinx.coroutines.launch
import java.util.UUID

/** Écrans de l'application (pile de navigation). */
sealed interface Screen {
    data object Loading : Screen
    data object Servers : Screen
    data object AddServer : Screen
    data class Login(val server: Server) : Screen
    data object Main : Screen
    data class Detail(val itemId: Long) : Screen
    data class Player(val itemId: Long, val fileId: Long? = null, val restart: Boolean = false) : Screen
}

enum class Tab(val label: String) { HOME("Accueil"), MOVIES("Films"), SERIES("Séries"), SEARCH("Recherche"), PROFILE("Profil") }

class AppViewModel(application: Application, private val container: AppContainer) : ViewModel() {

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
    /** Incrémenté après une lecture : les écrans rechargent progression et « vu ». */
    var refresh by mutableStateOf(0)
        private set

    /** Recharge les écrans (accueil, fiches) après une action qui change la progression ou les rangées. */
    fun reload() { refresh++ }

    /** Message affiché sur la liste des NAS (déclaré avant `init`, qui peut l'écrire immédiatement). */
    var serverError by mutableStateOf<String?>(null)

    /** Capacités de lecture de la box, mesurées une fois. */
    val caps: Caps = Capabilities.detect(application)

    val servers get() = container.servers.servers()
    /** Client HTTP partagé (cookie de session) : sert aussi à l'association par code avant connexion. */
    val http get() = container.http

    // Demande d'association en cours par NAS : réutilisée tant qu'elle est valide. Sans cela, chaque passage par
    // l'écran de connexion en créait une nouvelle, et la 5e (limite du NAS par adresse) était refusée 10 min.
    private val links = mutableMapOf<String, Pair<LinkStart, Long>>()

    fun cachedLink(serverId: String): LinkStart? = links[serverId]
        ?.takeIf { (link, at) -> System.currentTimeMillis() - at < (link.expiresIn - 30) * 1000L }?.first

    fun rememberLink(serverId: String, link: LinkStart) { links[serverId] = link to System.currentTimeMillis() }
    fun forgetLink(serverId: String) { links.remove(serverId) }

    init {
        viewModelScope.launch { start() }
    }

    private suspend fun start() {
        val all = container.servers.servers()
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
            else target.copy(username = profile.username).also { container.servers.save(it) }
            // Session révoquée plus tard (mot de passe changé, compte désactivé) : retour à la connexion.
            c.onUnauthorized = { viewModelScope.launch { if (client === c) sessionExpired(target) } }
            server = known
            client = c
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
                .also { container.servers.save(it) }
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
