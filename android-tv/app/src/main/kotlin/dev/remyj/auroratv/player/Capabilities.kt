package dev.remyj.auroratv.player

import android.content.Context
import android.media.MediaCodecInfo
import android.media.MediaCodecList
import android.os.Build
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.exoplayer.audio.AudioCapabilities
import dev.remyj.auroratv.net.Caps

/**
 * Ce que la box sait lire, mesuré et non supposé : décodeurs présents (MediaCodecList) et formats
 * acceptés en sortie numérique (passthrough HDMI/optique vers un ampli). Le serveur AURORA s'en sert
 * pour choisir lecture directe, remux ou conversion.
 *
 * ExoPlayer lit nativement MKV, MP4, MOV, WebM, MPEG-TS et AVI : ces conteneurs ne demandent jamais
 * de remux sur la box (contrairement au navigateur).
 */
object Capabilities {

    private val videoMimes = mapOf(
        "video/avc" to "h264",
        "video/hevc" to "hevc",
        "video/av01" to "av1",
        "video/x-vnd.on2.vp9" to "vp9",
        "video/mpeg2" to "mpeg2video",
    )

    private val audioMimes = mapOf(
        "audio/mp4a-latm" to "aac",
        "audio/mpeg" to "mp3",
        "audio/flac" to "flac",
        "audio/opus" to "opus",
        "audio/vorbis" to "vorbis",
        "audio/ac3" to "ac3",
        "audio/eac3" to "eac3",
        "audio/vnd.dts" to "dts",
        "audio/true-hd" to "truehd",
    )

    @androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
    fun detect(context: Context): Caps {
        val audio = linkedSetOf<String>()
        val hardware = linkedMapOf<String, Int>() // codec → hauteur max décodable
        val software = linkedMapOf<String, Int>()
        for (info in MediaCodecList(MediaCodecList.REGULAR_CODECS).codecInfos) {
            if (info.isEncoder) continue
            for (type in info.supportedTypes) {
                audioMimes[type.lowercase()]?.let { audio += it }
                val codec = videoMimes[type.lowercase()] ?: continue
                // Hauteur du décodeur, pas de l'écran : une box 4K sur une TV 1080p décode la 4K puis réduit.
                val height = runCatching { info.getCapabilitiesForType(type).videoCapabilities?.supportedHeights?.upper }
                    .getOrNull() ?: 0
                val target = if (isHardware(info)) hardware else software
                target[codec] = maxOf(target[codec] ?: 0, height)
            }
        }
        // Décodeurs logiciels exclus : ils annoncent AV1/HEVC jusqu'en 4K mais saccadent sur une box, sans
        // erreur, donc sans repli. Seulement si la box n'a aucun décodeur matériel déclaré (cas improbable),
        // on garde les logiciels plutôt que de tout faire convertir.
        val video = LinkedHashMap(hardware.ifEmpty { software })
        // H.264 est le format de sortie des conversions du NAS : toujours annoncé s'il existe un décodeur.
        if ("h264" !in video) software["h264"]?.let { video["h264"] = it }
        val maxHeight = listOfNotNull(video["h264"], video["hevc"]).maxOrNull() ?: 0
        // Passthrough : l'ampli décode ce que la box ne sait pas décoder elle-même.
        runCatching {
            val caps = AudioCapabilities.getCapabilities(context, AudioAttributes.DEFAULT, null)
            if (caps.supportsEncoding(C.ENCODING_AC3)) audio += "ac3"
            if (caps.supportsEncoding(C.ENCODING_E_AC3) || caps.supportsEncoding(C.ENCODING_E_AC3_JOC)) audio += "eac3"
            if (caps.supportsEncoding(C.ENCODING_DTS) || caps.supportsEncoding(C.ENCODING_DTS_HD)) audio += "dts"
            if (caps.supportsEncoding(C.ENCODING_DOLBY_TRUEHD)) audio += "truehd"
        }
        return Caps(
            containers = listOf("mp4", "mov", "mkv", "webm", "ts", "avi"),
            video = video.keys.toList(),
            audio = audio.toList(),
            hls = "native",
            maxHeight = if (maxHeight >= 2160) 2160 else maxHeight.coerceAtLeast(1080),
        )
    }

    /** Décodeur matériel : API 29+ le déclare ; avant, les décodeurs logiciels d'Android ont un préfixe connu. */
    private fun isHardware(info: MediaCodecInfo): Boolean =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) info.isHardwareAccelerated
        else !info.name.startsWith("OMX.google.") && !info.name.startsWith("c2.android.")
}
