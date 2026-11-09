package dev.remyj.auroratv.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.tv.material3.MaterialTheme
import androidx.tv.material3.darkColorScheme

/**
 * Palette « Cinémathèque », la même que l'app macOS (validée le 24/10/2026) : brun très sombre,
 * crème, or ; titres en serif. L'or est réservé au focus et aux états actifs.
 */
object Aurora {
    val Ink = Color(0xFF15110E)
    val Night = Color(0xFF100D0B)
    val Coal = Color(0xFF211B16)
    val Graphite = Color(0xFF2E261F)
    val Paper = Color(0xFFF4EAD8)
    // Tons sable volontairement clairs (plus que sur Mac) : sur un téléviseur, le réglage « texte à
    // contraste élevé » cerne chaque glyphe de noir ; un ton sombre devient flou (appris sur PlexSeries TV).
    val Mist = Color(0xFFEAD9B9)
    val Ash = Color(0xFFCDB892)
    val Accent = Color(0xFFD9B26F)
    val FocusGold = Color(0xFFFFE3A1)
    val AccentDeep = Color(0xFFB89A68)
    val Cream = Color(0xFFF1E4CC)
    val Line = Color(0x29D9B26F)
    /** Coche « vu » : vert franc, comme Plex et l'app Mac. */
    val Seen = Color(0xFF4CC76B)
    val Ember = Color(0xFFF2B366)
    val Danger = Color(0xFFFF7A6B)
    val Display = FontFamily.Serif
}

@Composable
fun AuroraTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = darkColorScheme(
            primary = Aurora.Accent,
            background = Aurora.Ink,
            surface = Aurora.Coal,
            onPrimary = Aurora.Ink,
            onBackground = Aurora.Paper,
            onSurface = Aurora.Paper,
        ),
        content = content,
    )
}
