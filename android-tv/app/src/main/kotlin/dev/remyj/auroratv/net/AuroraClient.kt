package dev.remyj.auroratv.net

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import java.io.IOException
import java.util.concurrent.ConcurrentHashMap

/** Erreur renvoyée par AURORA, avec son message (déjà en français) ou une erreur réseau. */
class ApiException(val code: Int, message: String) : Exception(message)

val json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
    coerceInputValues = true
}

/**
 * Jetons de session par serveur (« http://ip:port »). L'intercepteur les ajoute aux requêtes
 * d'images (Coil) ; le lecteur reçoit le même en-tête via [cookieHeader].
 */
object Sessions {
    private val tokens = ConcurrentHashMap<String, String>()

    fun set(baseUrl: String, token: String?) {
        val key = origin(baseUrl) ?: return
        if (token == null) tokens.remove(key) else tokens[key] = token
    }

    fun cookieHeader(baseUrl: String): String? = origin(baseUrl)?.let { tokens[it] }?.let { "$COOKIE=$it" }

    fun interceptor() = Interceptor { chain ->
        val req = chain.request()
        val token = tokens["${req.url.scheme}://${req.url.host}:${req.url.port}"]
        if (token == null || req.header("Cookie") != null) chain.proceed(req)
        else chain.proceed(req.newBuilder().header("Cookie", "$COOKIE=$token").build())
    }

    private fun origin(url: String): String? = url.toHttpUrlOrNull()?.let { "${it.scheme}://${it.host}:${it.port}" }

    const val COOKIE = "aurora_session"
}

/** Client d'un serveur AURORA. [baseUrl] : « http://192.168.1.20:8890 ». */
class AuroraClient(private val http: OkHttpClient, val baseUrl: String) {

    /** Appelé (thread réseau) quand le NAS refuse la session : 401 sur une requête authentifiée. */
    @Volatile var onUnauthorized: (() -> Unit)? = null

    fun absolute(path: String?): String? = when {
        path.isNullOrBlank() -> null
        path.startsWith("http") -> path
        else -> baseUrl.trimEnd('/') + path
    }

    private suspend fun call(method: String, path: String, body: String? = null): String = withContext(Dispatchers.IO) {
        val builder = Request.Builder()
            .url(baseUrl.trimEnd('/') + "/api/v1" + path)
            // Protection CSRF d'AURORA : obligatoire sur toute requête modifiante.
            .header("X-Aurora", "1")
        when (method) {
            "GET" -> builder.get()
            "DELETE" -> builder.delete()
            else -> builder.method(method, (body ?: "{}").toRequestBody(JSON_TYPE))
        }
        val response: Response = try {
            http.newCall(builder.build()).execute()
        } catch (e: IOException) {
            throw ApiException(0, "Le serveur AURORA ne répond pas (${e.message ?: "réseau"}). Vérifiez l'adresse et que le NAS est allumé.")
        }
        response.use { resp ->
            // Coupure pendant le corps (NAS en veille, Wi-Fi) : même erreur qu'une requête sans réponse.
            val text = try {
                resp.body.string()
            } catch (e: IOException) {
                throw ApiException(0, "Connexion au NAS interrompue (${e.message ?: "réseau"}).")
            }
            if (!resp.isSuccessful) {
                if (resp.code == 401) onUnauthorized?.invoke()
                val msg = runCatching { json.decodeFromString<ErrorBody>(text).error }.getOrNull()
                throw ApiException(resp.code, msg?.replaceFirstChar { it.uppercase() } ?: "Erreur ${resp.code}")
            }
            text
        }
    }

    private suspend inline fun <reified T> get(path: String): T = decode(call("GET", path))
    private suspend inline fun <reified T, reified B> post(path: String, body: B): T =
        decode(call("POST", path, json.encodeToString(body)))

    /** Réponse illisible (page HTML d'un proxy, autre service sur ce port) : erreur affichable, pas un plantage. */
    private inline fun <reified T> decode(text: String): T = try {
        json.decodeFromString(serializer<T>(), text)
    } catch (e: kotlinx.serialization.SerializationException) {
        throw ApiException(0, "Réponse inattendue : est-ce bien AURORA à l'adresse $baseUrl ?")
    } catch (e: IllegalArgumentException) {
        throw ApiException(0, "Réponse inattendue : est-ce bien AURORA à l'adresse $baseUrl ?")
    }

    suspend fun status(): Status = get("/status")

    /** Connexion : renvoie le jeton de session (cookie HttpOnly côté navigateur). */
    suspend fun login(username: String, password: String): String = withContext(Dispatchers.IO) {
        val req = Request.Builder()
            .url(baseUrl.trimEnd('/') + "/api/v1/auth/login")
            .header("X-Aurora", "1")
            .post(json.encodeToString(Credentials(username, password)).toRequestBody(JSON_TYPE))
            .build()
        val resp = try {
            http.newCall(req).execute()
        } catch (e: IOException) {
            throw ApiException(0, "Le serveur AURORA ne répond pas (${e.message ?: "réseau"}).")
        }
        resp.use {
            if (!it.isSuccessful) {
                val msg = runCatching { json.decodeFromString<ErrorBody>(it.body.string()).error }.getOrNull()
                throw ApiException(it.code, msg?.replaceFirstChar { c -> c.uppercase() } ?: "Erreur ${it.code}")
            }
            sessionCookie(it) ?: throw ApiException(500, "Réponse de connexion inattendue")
        }
    }

    private fun sessionCookie(resp: Response): String? = resp.headers("Set-Cookie").firstNotNullOfOrNull { c ->
        c.substringBefore(';').takeIf { kv -> kv.startsWith(Sessions.COOKIE + "=") }?.substringAfter('=')
    }

    /** Association par code (comme plex.tv/link) : demande un code à afficher sur la TV. */
    suspend fun linkStart(deviceName: String): LinkStart =
        decode(call("POST", "/auth/link/start", json.encodeToString(LinkName(deviceName))))

    /**
     * Suivi de l'association : null tant que personne n'a validé le code, sinon le jeton de session.
     * Code expiré : ApiException 410.
     */
    suspend fun linkPoll(poll: String): String? = withContext(Dispatchers.IO) {
        val req = Request.Builder()
            .url(baseUrl.trimEnd('/') + "/api/v1/auth/link/poll")
            .header("X-Aurora", "1")
            .post(json.encodeToString(LinkPoll(poll)).toRequestBody(JSON_TYPE))
            .build()
        val resp = try {
            http.newCall(req).execute()
        } catch (e: IOException) {
            throw ApiException(0, "Le serveur AURORA ne répond pas (${e.message ?: "réseau"}).")
        }
        resp.use {
            val text = try { it.body.string() } catch (e: IOException) { throw ApiException(0, "Connexion au NAS interrompue.") }
            if (!it.isSuccessful) {
                val msg = runCatching { json.decodeFromString<ErrorBody>(text).error }.getOrNull()
                throw ApiException(it.code, msg?.replaceFirstChar { c -> c.uppercase() } ?: "Erreur ${it.code}")
            }
            if (decode<LinkStatus>(text).status == "ok") sessionCookie(it) ?: throw ApiException(500, "Réponse d'association inattendue")
            else null
        }
    }

    suspend fun me(): Me = get("/auth/me")
    suspend fun logout() { runCatching { call("POST", "/auth/logout") } }
    suspend fun home(): Home = get("/home")
    suspend fun libraries(): List<LibraryRef> = get("/libraries")

    suspend fun list(kind: String, libraryId: Long?, sort: String, desc: Boolean, offset: Int, limit: Int, unwatched: Boolean = false): ListPage {
        val q = buildString {
            append("?sort=").append(sort).append("&order=").append(if (desc) "desc" else "asc")
            append("&offset=").append(offset).append("&limit=").append(limit)
            if (libraryId != null) append("&library=").append(libraryId)
            if (unwatched) append("&watched=unwatched")
        }
        return get("/${if (kind == "movie") "movies" else "series"}$q")
    }

    suspend fun item(id: Long): Detail = get("/items/$id")
    suspend fun episodes(seriesId: Long, season: Int): List<Card> = get("/items/$seriesId/seasons/$season")
    suspend fun next(episodeId: Long): Card? = get<NextResp>("/items/$episodeId/next").next
    suspend fun search(q: String): SearchResults = get("/search?q=" + java.net.URLEncoder.encode(q, "UTF-8") + "&limit=30")

    suspend fun play(req: PlayRequest): PlayResponse = post("/play", req)

    suspend fun progress(itemId: Long, fileId: Long, position: Double, duration: Double) {
        runCatching { call("POST", "/items/$itemId/progress", json.encodeToString(Progress(fileId, position, duration))) }
    }

    suspend fun setState(itemId: Long, favorite: Boolean? = null, watchlist: Boolean? = null, watched: Boolean? = null, season: Int? = null) {
        call("POST", "/items/$itemId/state", json.encodeToString(StateBody(favorite, watchlist, watched, season)))
    }

    /** « Retirer de Continuer à regarder » (fichier et progression intacts ; revient à la prochaine lecture). */
    suspend fun hideContinue(itemId: Long) { call("POST", "/items/$itemId/continue/hide") }

    suspend fun stopSession(id: String) { runCatching { call("DELETE", "/hls/$id") } }

    @Serializable private data class ErrorBody(val error: String)
    @Serializable private data class LinkName(val name: String)
    @Serializable private data class LinkPoll(val poll: String)
    @Serializable private data class LinkStatus(val status: String)
    @Serializable private data class Credentials(val username: String, val password: String)
    @Serializable private data class NextResp(val next: Card? = null)
    @Serializable private data class Progress(val fileId: Long, val position: Double, val duration: Double)
    @Serializable private data class StateBody(val favorite: Boolean?, val watchlist: Boolean?, val watched: Boolean?, val season: Int?)

    private companion object {
        val JSON_TYPE = "application/json".toMediaType()
    }
}

@Serializable
data class PlayRequest(
    val fileId: Long,
    val caps: Caps,
    val start: Double = 0.0,
    val audioIndex: Int? = null,
    val subtitleIndex: Int? = null,
    val forceMode: String? = null,
    /** Demande un lien signé, pour un lecteur externe (VLC). */
    val external: Boolean = false,
)
