package dev.remyj.auroratv.usb

import android.content.Context
import android.os.Build
import android.os.Environment
import android.os.StatFs
import android.os.storage.StorageManager
import android.os.storage.StorageVolume

/**
 * Volumes USB amovibles (disque branché sur le Formovie), lus **directement par le système de
 * fichiers** (montage sous `/storage/…`), comme le fait VLC. Le Formovie n'embarque pas de
 * sélecteur de documents (DocumentsUI) : le Storage Access Framework y plante, il est donc écarté
 * au profit de l'accès direct, qui exige l'autorisation « Tous les fichiers »
 * (`MANAGE_EXTERNAL_STORAGE`).
 */
class UsbStore(private val context: Context) {
    private val sm: StorageManager = context.getSystemService(StorageManager::class.java)!!

    /** Volumes USB amovibles montés — chaque partition apparaît comme un volume. */
    fun mountedVolumes(): List<StorageVolume> = sm.storageVolumes
        .filter { it.isRemovable && it.state == Environment.MEDIA_MOUNTED }

    fun label(volume: StorageVolume): String =
        volume.getDescription(context).takeIf { it.isNotBlank() } ?: "Disque USB"

    /** Chemin du point de montage (`/storage/…`), ou null si indisponible (API 30+). */
    fun path(volume: StorageVolume): String? =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) volume.directory?.path else null

    /** Capacité totale du volume, en clair (« 1,8 To »), ou null si indisponible. */
    fun capacityText(volume: StorageVolume): String? {
        val dir = path(volume) ?: return null
        return runCatching { formatBytes(StatFs(dir).totalBytes) }.getOrNull()
    }

    /** Le processus a-t-il l'autorisation « Tous les fichiers » (MANAGE_EXTERNAL_STORAGE) ? */
    fun canAccessAllFiles(): Boolean =
        Build.VERSION.SDK_INT < Build.VERSION_CODES.R || Environment.isExternalStorageManager()
}
