package dev.remyj.auroratv.ui

import dev.remyj.auroratv.net.LibraryRef
import dev.remyj.auroratv.net.ListPage

fun libraryCount(library: LibraryRef, kind: String): Int =
    if (kind == "movie") library.movies else library.series

fun foldersFor(libraries: List<LibraryRef>, kind: String): List<LibraryRef> =
    libraries.filter { libraryCount(it, kind) > 0 }

/** D'autres œuvres existent-elles au-delà de cette page ? (le NAS plafonne à 500 par page) */
fun ListPage.hasMore(): Boolean = total > items.size

/** Une sélection indépendante par catégorie ; null signifie « Tous les dossiers ». */
data class LibraryNavigation(
    val serverId: String? = null,
    private val movieId: Long? = null,
    private val seriesId: Long? = null,
) {
    fun selected(kind: String): Long? = if (kind == "movie") movieId else seriesId

    fun select(kind: String, id: Long?, libraries: List<LibraryRef>): LibraryNavigation {
        val valid = id?.takeIf { selectedId -> foldersFor(libraries, kind).any { it.id == selectedId } }
        return if (kind == "movie") copy(movieId = valid) else copy(seriesId = valid)
    }

    fun reconcile(libraries: List<LibraryRef>): LibraryNavigation =
        select("movie", movieId, libraries).select("series", seriesId, libraries)

    fun forServer(id: String): LibraryNavigation = if (id == serverId) this else LibraryNavigation(serverId = id)
}
