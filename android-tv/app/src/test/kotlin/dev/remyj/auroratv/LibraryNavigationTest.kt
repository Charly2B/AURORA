package dev.remyj.auroratv

import dev.remyj.auroratv.net.LibraryRef
import dev.remyj.auroratv.net.ListPage
import dev.remyj.auroratv.ui.LibraryNavigation
import dev.remyj.auroratv.ui.foldersFor
import dev.remyj.auroratv.ui.hasMore
import dev.remyj.auroratv.ui.libraryCount
import org.junit.Assert.*
import org.junit.Test

class LibraryNavigationTest {
    private val libraries = listOf(
        LibraryRef(1, "Mixte", movies = 7, series = 3),
        LibraryRef(2, "Films", movies = 5),
        LibraryRef(3, "Séries", series = 9),
        LibraryRef(4, "Vide"),
    )

    @Test fun foldersAndCountsUseEachCategory() {
        assertEquals(listOf(1L, 2L), foldersFor(libraries, "movie").map { it.id })
        assertEquals(listOf(1L, 3L), foldersFor(libraries, "series").map { it.id })
        assertEquals(7, libraryCount(libraries[0], "movie"))
        assertEquals(3, libraryCount(libraries[0], "series"))
        assertEquals(12, foldersFor(libraries, "movie").sumOf { libraryCount(it, "movie") })
        assertEquals(12, foldersFor(libraries, "series").sumOf { libraryCount(it, "series") })
    }

    @Test fun selectionDefaultsToAllAndIsIndependentForEachCategory() {
        val initial = LibraryNavigation()
        assertNull(initial.selected("movie"))
        assertNull(initial.selected("series"))
        val chosen = initial.select("movie", 2, libraries).select("series", 3, libraries)
        assertEquals(2L, chosen.selected("movie"))
        assertEquals(3L, chosen.selected("series"))
        val allMovies = chosen.select("movie", null, libraries)
        assertNull(allMovies.selected("movie"))
        assertEquals(3L, allMovies.selected("series"))
    }

    @Test fun rejectsFolderWithoutItemsOfRequestedType() {
        val chosen = LibraryNavigation().select("movie", 3, libraries).select("series", 2, libraries)
        assertNull(chosen.selected("movie"))
        assertNull(chosen.selected("series"))
    }

    @Test fun disappearedOrEmptiedFolderFallsBackToAll() {
        val chosen = LibraryNavigation().select("movie", 1, libraries).select("series", 3, libraries)
        val updated = chosen.reconcile(listOf(LibraryRef(1, "Mixte", series = 3)))
        assertNull(updated.selected("movie"))
        assertNull(updated.selected("series"))
        assertNull(chosen.reconcile(emptyList()).selected("movie"))
    }

    @Test fun sameNasKeepsBothSelectionsWhenFoldersAreReloadedInAnotherOrder() {
        val chosen = LibraryNavigation().forServer("nas-a")
            .select("movie", 2, libraries).select("series", 1, libraries)
        val reloaded = chosen.forServer("nas-a").reconcile(libraries.reversed())
        assertEquals(2L, reloaded.selected("movie"))
        assertEquals(1L, reloaded.selected("series"))
    }

    @Test fun changingNasResetsSelectionEvenWhenFolderIdsOverlap() {
        val chosen = LibraryNavigation().forServer("nas-a").select("movie", 1, libraries)
        assertEquals(1L, chosen.forServer("nas-a").selected("movie"))
        val switched = chosen.forServer("nas-b")
        assertNull(switched.selected("movie"))
        assertNull(switched.selected("series"))
    }

    @Test fun pageKnowsWhenMoreItemsRemain() {
        assertTrue(ListPage(items = List(500) { dev.remyj.auroratv.net.Card(it.toLong(), "movie", "F$it") }, total = 812).hasMore())
        assertFalse(ListPage(items = List(812) { dev.remyj.auroratv.net.Card(it.toLong(), "movie", "F$it") }, total = 812).hasMore())
        assertFalse(ListPage(total = 0).hasMore())
    }
}
