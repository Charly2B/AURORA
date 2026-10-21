package dev.remyj.auroratv.data

import android.content.Context
import dev.remyj.auroratv.net.json
import dev.remyj.auroratv.security.TokenStore
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

/** Un NAS Synology où tourne AURORA. */
@Serializable
data class Server(val id: String, val name: String, val baseUrl: String, val username: String = "")

/**
 * Serveurs enregistrés (préférences) et jetons de session (chiffrés, un magasin par serveur).
 * Rien n'est envoyé ailleurs qu'au NAS concerné.
 */
class ServerStore(private val context: Context) {
    private val prefs = context.getSharedPreferences("auroratv.servers", Context.MODE_PRIVATE)
    private val serializer = ListSerializer(Server.serializer())

    fun servers(): List<Server> =
        prefs.getString(KEY_SERVERS, null)?.let { runCatching { json.decodeFromString(serializer, it) }.getOrNull() } ?: emptyList()

    fun save(server: Server) {
        val list = servers().filter { it.id != server.id } + server
        prefs.edit().putString(KEY_SERVERS, json.encodeToString(serializer, list)).apply()
    }

    fun remove(id: String) {
        prefs.edit().putString(KEY_SERVERS, json.encodeToString(serializer, servers().filter { it.id != id })).apply()
        tokens(id).clear()
        if (lastServerId() == id) prefs.edit().remove(KEY_LAST).apply()
    }

    fun lastServerId(): String? = prefs.getString(KEY_LAST, null)
    fun setLastServer(id: String) = prefs.edit().putString(KEY_LAST, id).apply()

    fun tokens(serverId: String) = TokenStore(context, serverId)

    private companion object {
        const val KEY_SERVERS = "servers"
        const val KEY_LAST = "last"
    }
}

/**
 * Normalise une saisie : « 192.168.1.20 », « 192.168.1.20:8890 », « nas.local » ou une URL complète.
 * Port AURORA par défaut : 8890. Renvoie « http://hôte:port », ou null si la saisie est invalide.
 */
fun normalizeAddress(input: String): String? {
    var s = input.trim().trimEnd('/')
    if (s.isEmpty() || s.contains(' ')) return null
    val explicitPort = Regex("^(https?://)?(\\[[^]]+]|[^/:\\[]+):\\d+").containsMatchIn(s)
    if (!s.contains("://")) s = "http://$s"
    val url = s.toHttpUrlOrNull() ?: return null
    val host = if (url.host.contains(':')) "[${url.host}]" else url.host
    return "${url.scheme}://$host:${if (explicitPort) url.port else 8890}"
}
