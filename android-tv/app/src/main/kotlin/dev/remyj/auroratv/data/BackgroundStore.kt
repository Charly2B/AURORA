package dev.remyj.auroratv.data

import android.content.Context
import android.content.SharedPreferences

/** Apparence locale, indépendante du compte et de la session, une préférence par NAS enregistré. */
class BackgroundStore(private val prefs: SharedPreferences) {
    constructor(context: Context) : this(context.getSharedPreferences("auroratv.backgrounds", Context.MODE_PRIVATE))

    fun get(serverId: String): String? = prefs.getString(serverId, null)

    fun set(serverId: String, imageId: String?) {
        if (imageId == null) remove(serverId)
        else prefs.edit().putString(serverId, imageId).apply()
    }

    fun remove(serverId: String) { prefs.edit().remove(serverId).apply() }
}
