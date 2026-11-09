package dev.remyj.auroratv

import dev.remyj.auroratv.net.Card
import dev.remyj.auroratv.net.Section
import dev.remyj.auroratv.ui.homeRails
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class HomeRailsTest {
    private fun card(id: Long) = Card(id, "movie", "F$id")

    @Test fun keepsServerOrderAndDropsEmptyRails() {
        val sections = listOf(
            Section(id = "recent_movies_1", title = "Films", items = listOf(card(1))),
            Section(id = "watchlist", title = "Ma liste"),
            Section(id = "recent_movies_2", title = "Films 4K", items = listOf(card(2), card(3))),
        )
        assertEquals(listOf("recent_movies_1", "recent_movies_2"), homeRails(sections).map { it.id })
    }

    @Test fun emptyHomeHasNoRails() {
        assertTrue(homeRails(emptyList()).isEmpty())
        assertTrue(homeRails(listOf(Section("watchlist", "Ma liste"))).isEmpty())
    }

    @Test fun continueAndNextUpAreHiddenOnTheHome() {
        // Retirés le 29/10/2026 : la reprise passe par les dossiers « En cours » du NAS.
        val sections = listOf(
            Section("continue", "Continuer à regarder", items = listOf(card(1))),
            Section("nextup", "Prochain épisode", items = listOf(Card(2, "episode", "E2"))),
            Section("recent_movies_1", "Films", items = listOf(card(3))),
        )
        assertEquals(listOf("recent_movies_1"), homeRails(sections).map { it.id })
    }

    @Test fun collectionsAreDroppedBecauseTheirIdsAreNotItems() {
        val sections = listOf(
            Section(id = "collections", title = "Collections", items = listOf(Card(42, "collection", "Saga"))),
            Section(id = "recent_movies_1", title = "Films", items = listOf(card(7))),
        )
        val rails = homeRails(sections)
        assertEquals(listOf("recent_movies_1"), rails.map { it.id })
        assertTrue(rails.single().items.all { it.kind in setOf("movie", "series", "episode") })
    }

    @Test fun unsupportedCardsAreFilteredOutOfAnOtherwiseValidRail() {
        val sections = listOf(
            Section("recent_movies_1", "Films", items = listOf(card(1), Card(9, "collection", "Saga"))),
        )
        assertEquals(listOf(1L), homeRails(sections).single().items.map { it.id })
    }
}
