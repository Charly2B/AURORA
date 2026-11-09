package dev.remyj.auroratv.ui

import dev.remyj.auroratv.net.Section

/**
 * Rangées de l'accueil : celles que le serveur renvoie non vides, dans son ordre, limitées aux types que
 * la TV sait ouvrir. « Collections » renvoie des cartes `kind = "collection"` dont l'identifiant est celui
 * de la collection, pas d'une œuvre : les garder ouvrait la fiche d'un film au hasard (et « Vu » écrivait
 * sur une œuvre sans rapport).
 *
 * « Continuer à regarder » et « Prochain épisode » sont retirés (demande utilisateur du 29/10/2026) :
 * la reprise passe par les dossiers « En cours » du NAS, comme sur la TV Samsung.
 */
private val HiddenHomeRails = setOf("continue", "nextup")

fun homeRails(sections: List<Section>): List<Section> = sections.mapNotNull { section ->
    if (section.id in HiddenHomeRails) return@mapNotNull null
    val items = section.items.filter { it.kind == "movie" || it.kind == "series" || it.kind == "episode" }
    if (items.isEmpty()) null else section.copy(items = items)
}
