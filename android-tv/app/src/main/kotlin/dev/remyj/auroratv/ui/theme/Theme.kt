package dev.remyj.auroratv.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.tv.material3.MaterialTheme
import androidx.tv.material3.darkColorScheme

/**
 * Palette AURORA (identique à l'interface web) : salle de projection, un seul accent « aurore »
 * réservé au focus et aux états actifs.
 */
object Aurora {
    val Ink = Color(0xFF08090B)
    val Night = Color(0xFF0E1013)
    val Coal = Color(0xFF15181C)
    val Graphite = Color(0xFF1F2328)
    val Paper = Color(0xFFF2F1EC)
    // Gris volontairement clairs : sur un téléviseur, le réglage « texte à contraste élevé »
    // cerne chaque glyphe de noir ; un gris sombre devient flou (appris sur PlexSeries TV).
    val Mist = Color(0xFFC9CED4)
    val Ash = Color(0xFFA3AAB3)
    val Accent = Color(0xFF7EE8C6)
    val AccentDeep = Color(0xFF2FBF98)
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
