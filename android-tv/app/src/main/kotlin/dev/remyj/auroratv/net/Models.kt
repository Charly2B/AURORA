package dev.remyj.auroratv.net

import kotlinx.serialization.Serializable

// Modèles de l'API AURORA (/api/v1). Champs facultatifs par défaut : le serveur omet les valeurs vides.

@Serializable
data class Card(
    val id: Long,
    val kind: String,
    val title: String,
    val year: Int? = null,
    val poster: String? = null,
    val backdrop: String? = null,
    val thumb: String? = null,
    val rating: Double? = null,
    val runtime: Int? = null,
    val duration: Double? = null,
    val resolution: String? = null,
    val hdr: String? = null,
    val watched: Boolean = false,
    val position: Double? = null,
    val favorite: Boolean = false,
    val watchlist: Boolean = false,
    val overview: String? = null,
    val genres: List<String>? = null,
    val seriesId: Long? = null,
    val seriesTitle: String? = null,
    val season: Int? = null,
    val episode: Int? = null,
    val episodeEnd: Int? = null,
    val unwatched: Int? = null,
    val total: Int? = null,
    val versions: Int? = null,
) {
    /** Durée de référence en secondes (fichier, sinon métadonnées). */
    val length: Double get() = duration ?: ((runtime ?: 0) * 60.0)
    val progress: Float get() = if (length > 0 && (position ?: 0.0) > 0) ((position ?: 0.0) / length).toFloat() else 0f
}

@Serializable
data class Section(
    val id: String,
    val title: String,
    val libraryId: Long? = null,
    val kind: String? = null,
    val items: List<Card> = emptyList(),
)

@Serializable
data class Home(val hero: Card? = null, val sections: List<Section> = emptyList())

@Serializable
data class LibraryRef(val id: Long, val name: String, val movies: Int = 0, val series: Int = 0)

@Serializable
data class ListPage(val items: List<Card> = emptyList(), val total: Int = 0)

@Serializable
data class Audio(
    val index: Int,
    val stream: Int = -1,
    val codec: String = "",
    val channels: Int = 0,
    val language: String = "",
    val title: String = "",
    val default: Boolean = false,
    val atmos: Boolean = false,
)

@Serializable
data class Subtitle(
    val index: Int,
    val codec: String = "",
    val language: String = "",
    val title: String = "",
    val default: Boolean = false,
    val forced: Boolean = false,
    val text: Boolean = true,
)

@Serializable
data class FileInfo(
    val id: Long,
    val name: String = "",
    val label: String = "",
    val size: Long = 0,
    val container: String = "",
    val duration: Double = 0.0,
    val videoCodec: String = "",
    val videoProfile: String = "",
    val width: Int = 0,
    val height: Int = 0,
    val resolution: String = "",
    val hdr: String = "",
    val audio: List<Audio> = emptyList(),
    val subtitles: List<Subtitle> = emptyList(),
)

@Serializable
data class Person(val name: String, val role: String, val character: String? = null, val photo: String? = null)

@Serializable
data class ItemState(
    val position: Double = 0.0,
    val duration: Double = 0.0,
    val watched: Boolean = false,
    val lastFileId: Long? = null,
    val favorite: Boolean = false,
    val watchlist: Boolean = false,
)

@Serializable
data class Season(val number: Int, val title: String, val episodes: Int = 0, val unwatched: Int = 0, val poster: String? = null)

@Serializable
data class Detail(
    val id: Long,
    val kind: String,
    val title: String,
    val year: Int? = null,
    val poster: String? = null,
    val backdrop: String? = null,
    val thumb: String? = null,
    val rating: Double? = null,
    val ratingSource: String? = null,
    val runtime: Int? = null,
    val duration: Double? = null,
    val resolution: String? = null,
    val hdr: String? = null,
    val overview: String? = null,
    val tagline: String? = null,
    val originalTitle: String? = null,
    val contentRating: String? = null,
    val status: String? = null,
    val genres: List<String>? = null,
    val seriesId: Long? = null,
    val seriesTitle: String? = null,
    val season: Int? = null,
    val episode: Int? = null,
    val episodeEnd: Int? = null,
    val total: Int? = null,
    val unwatched: Int? = null,
    val watched: Boolean = false,
    val favorite: Boolean = false,
    val watchlist: Boolean = false,
    val files: List<FileInfo> = emptyList(),
    val people: List<Person> = emptyList(),
    val state: ItemState = ItemState(),
    val seasons: List<Season>? = null,
    val nextUp: Card? = null,
    val collections: List<Card>? = null,
    val similar: List<Card>? = null,
)

@Serializable
data class PlaySubtitle(
    val index: Int,
    val label: String = "",
    val language: String = "",
    val codec: String = "",
    val forced: Boolean = false,
    val default: Boolean = false,
    val text: Boolean = true,
    val external: Boolean = false,
    val url: String? = null,
)

@Serializable
data class PlayResponse(
    val mode: String,
    val reason: String = "",
    val url: String,
    val sessionId: String? = null,
    val offset: Double = 0.0,
    val start: Double = 0.0,
    val duration: Double = 0.0,
    val itemId: Long = 0,
    val fileId: Long = 0,
    val audio: List<Audio> = emptyList(),
    val audioStream: Int = -1,
    val subtitles: List<PlaySubtitle> = emptyList(),
    val signedUrl: String? = null,
)

@Serializable
data class SearchResults(
    val movies: List<Card> = emptyList(),
    val series: List<Card> = emptyList(),
    val episodes: List<Card> = emptyList(),
    val people: List<String> = emptyList(),
    val fuzzy: Boolean = false,
)

@Serializable
data class Prefs(
    val autoplayNext: Boolean = true,
    val audioLang: String = "fre",
    val subLang: String = "",
    val subMode: String = "auto",
)

@Serializable
data class Me(val id: Long, val username: String, val role: String, val prefs: Prefs = Prefs())

@Serializable
data class Status(val version: String = "", val hasAdmin: Boolean = false, val setupDone: Boolean = false, val user: Me? = null)

@Serializable
data class Caps(
    val containers: List<String>,
    val video: List<String>,
    val audio: List<String>,
    val hls: String,
    val maxHeight: Int,
)

/** Code d'association affiché sur la TV ; [poll] est le secret de suivi, connu de la TV seule. */
@Serializable
data class LinkStart(val code: String, val poll: String, val expiresIn: Int = 600)
