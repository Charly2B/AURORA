package dev.remyj.auroratv.ui.components

import android.graphics.Bitmap
import coil3.size.Size
import coil3.transform.Transformation

/**
 * Flou pour Android 11 (le `Modifier.blur` de Compose exige l'API 31) : moyenne glissante horizontale puis
 * verticale, [passes] fois, sur une image déjà réduite par Coil (≈ 160 × 90) — quelques millisecondes.
 * Un simple agrandissement d'une petite image donnait des pavés, pas un flou.
 */
class BoxBlur(private val radius: Int = 6, private val passes: Int = 3) : Transformation() {
    override val cacheKey = "box-blur-$radius-$passes"

    override suspend fun transform(input: Bitmap, size: Size): Bitmap {
        val w = input.width
        val h = input.height
        val src = IntArray(w * h)
        input.getPixels(src, 0, w, 0, 0, w, h)
        val tmp = IntArray(w * h)
        repeat(passes) {
            blurLine(src, tmp, w, h, horizontal = true)
            blurLine(tmp, src, w, h, horizontal = false)
        }
        return Bitmap.createBitmap(src, w, h, Bitmap.Config.ARGB_8888)
    }

    private fun blurLine(from: IntArray, to: IntArray, w: Int, h: Int, horizontal: Boolean) {
        val lines = if (horizontal) h else w
        val len = if (horizontal) w else h
        val window = radius * 2 + 1
        for (line in 0 until lines) {
            var a = 0; var r = 0; var g = 0; var b = 0
            fun at(i: Int): Int {
                val p = i.coerceIn(0, len - 1)
                return if (horizontal) from[line * w + p] else from[p * w + line]
            }
            for (i in -radius..radius) {
                val c = at(i)
                a += c ushr 24; r += (c shr 16) and 0xFF; g += (c shr 8) and 0xFF; b += c and 0xFF
            }
            for (i in 0 until len) {
                val index = if (horizontal) line * w + i else i * w + line
                to[index] = ((a / window) shl 24) or ((r / window) shl 16) or ((g / window) shl 8) or (b / window)
                val out = at(i - radius)
                val inc = at(i + radius + 1)
                a += (inc ushr 24) - (out ushr 24)
                r += ((inc shr 16) and 0xFF) - ((out shr 16) and 0xFF)
                g += ((inc shr 8) and 0xFF) - ((out shr 8) and 0xFF)
                b += (inc and 0xFF) - (out and 0xFF)
            }
        }
    }
}
