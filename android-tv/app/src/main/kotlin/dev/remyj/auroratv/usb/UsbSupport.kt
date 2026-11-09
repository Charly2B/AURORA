package dev.remyj.auroratv.usb

/** Extensions de fichiers vidéo reconnues dans le navigateur USB (sans métadonnées AURORA). */
private val VIDEO_EXTENSIONS = setOf(
    "mkv", "mp4", "avi", "ts", "m2ts", "mov", "m4v", "wmv", "flv", "webm",
    "mpg", "mpeg", "3gp", "vob", "iso", "divx", "rmvb", "mts",
)

/** Vrai si le fichier est une vidéo : type MIME vidéo, ou extension connue. */
fun isVideoFile(name: String?, mimeType: String?): Boolean {
    if (mimeType != null && mimeType.startsWith("video/")) return true
    val ext = name?.substringAfterLast('.', "")?.lowercase() ?: return false
    return ext in VIDEO_EXTENSIONS
}

/** Taille lisible : « 8,2 Go », « 1,8 To », « 640 Mo ». */
fun formatBytes(bytes: Long): String {
    if (bytes < 0) return ""
    val tb = bytes / 1_000_000_000_000.0
    val gb = bytes / 1_000_000_000.0
    val mb = bytes / 1_000_000.0
    return when {
        bytes >= 1_000_000_000_000 -> "%.1f To".format(tb)
        bytes >= 1_000_000_000 -> "%.1f Go".format(gb)
        bytes >= 1_000_000 -> "%.0f Mo".format(mb)
        else -> "$bytes o"
    }
}

/** Durée lisible : « 2 h 28 », « 45 min ». */
fun formatDuration(seconds: Long): String {
    if (seconds <= 0) return ""
    val h = seconds / 3600
    val m = (seconds % 3600) / 60
    return when {
        h > 0 -> "${h} h %02d".format(m)
        else -> "$m min"
    }
}
