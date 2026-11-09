package dev.remyj.auroratv

import dev.remyj.auroratv.net.Person
import dev.remyj.auroratv.ui.creditLine
import dev.remyj.auroratv.ui.featuredSlot
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class FeaturedTest {
    @Test fun rankPicksMoviesThenSeries() {
        assertEquals("movie" to 0, featuredSlot(movies = 3, series = 2, rank = 0))
        assertEquals("movie" to 2, featuredSlot(movies = 3, series = 2, rank = 2))
        assertEquals("series" to 1, featuredSlot(movies = 3, series = 2, rank = 4))
        assertEquals("series" to 0, featuredSlot(movies = 0, series = 2, rank = 0))
    }

    @Test fun creditNamesMakersThenActors() {
        val film = listOf(
            Person("Christopher Nolan", "director"),
            Person("Matthew McConaughey", "actor"),
            Person("Anne Hathaway", "actor"),
            Person("Jessica Chastain", "actor"),
        )
        assertEquals("De Christopher Nolan · avec Matthew McConaughey, Anne Hathaway", creditLine("movie", film))
        val series = listOf(Person("Rachel Kondo", "creator"), Person("Justin Marks", "creator"))
        assertEquals("Créée par Rachel Kondo et Justin Marks", creditLine("series", series))
        assertEquals("Avec Ewan McGregor", creditLine("series", listOf(Person("Ewan McGregor", "actor"))))
        assertNull(creditLine("movie", emptyList()))
    }
}
