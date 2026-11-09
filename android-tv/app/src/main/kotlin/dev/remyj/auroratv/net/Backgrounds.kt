package dev.remyj.auroratv.net

import kotlinx.serialization.Serializable

/** Métadonnées de l'image stockée sur le NAS ; identifiant SHA-256 stable. */
@Serializable
data class BackgroundImage(
    val id: String,
    val name: String,
    val url: String,
    val width: Int,
    val height: Int,
    val size: Long,
    val createdAt: Long,
)

@Serializable
internal data class BackgroundsResponse(val images: List<BackgroundImage>)
