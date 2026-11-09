package dev.remyj.auroratv.ui

import dev.remyj.auroratv.net.Person

/**
 * Titre « À la une » de l'accueil, comme sur Mac : un rang tiré parmi tous les films puis toutes les séries
 * (comptes de /libraries), traduit en (type, position) pour une seule requête `list` à `limit = 1`.
 */
fun featuredSlot(movies: Int, series: Int, rank: Int): Pair<String, Int> =
    if (rank < movies) "movie" to rank else "series" to (rank - movies)

/** « De Christopher Nolan · avec Matthew McConaughey, Anne Hathaway » ; « Créée par … » pour une série. */
fun creditLine(kind: String, people: List<Person>): String? {
    val makers = people.filter { it.role == "director" || it.role == "creator" }.take(2).map { it.name }
    val actors = people.filter { it.role == "actor" }.take(2).map { it.name }
    val parts = buildList {
        if (makers.isNotEmpty()) add((if (kind == "series") "Créée par " else "De ") + makers.joinToString(" et "))
        if (actors.isNotEmpty()) add((if (isEmpty()) "Avec " else "avec ") + actors.joinToString(", "))
    }
    return parts.takeIf { it.isNotEmpty() }?.joinToString(" · ")
}
