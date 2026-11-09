package dev.remyj.auroratv

import android.content.SharedPreferences
import dev.remyj.auroratv.data.BackgroundStore
import java.lang.reflect.Proxy
import org.junit.Assert.*
import org.junit.Test

class BackgroundStoreTest {
    @Test fun choicesPersistAcrossStoreInstancesAndStayIsolatedByNas() {
        val prefs = preferences()
        val first = BackgroundStore(prefs)
        first.set("nas-a", "a".repeat(64))
        first.set("nas-b", "b".repeat(64))
        val restored = BackgroundStore(prefs)
        assertEquals("a".repeat(64), restored.get("nas-a"))
        assertEquals("b".repeat(64), restored.get("nas-b"))
        restored.set("nas-a", null)
        assertNull(BackgroundStore(prefs).get("nas-a"))
        assertEquals("b".repeat(64), BackgroundStore(prefs).get("nas-b"))
    }

    @Test fun removalClearsOnlyThatNas() {
        val store = BackgroundStore(preferences())
        store.set("nas-a", "a".repeat(64))
        store.set("nas-b", "b".repeat(64))
        store.remove("nas-a")
        assertNull(store.get("nas-a"))
        assertEquals("b".repeat(64), store.get("nas-b"))
    }

    /** In-memory preferences backend; the production store owns keys and persistence operations. */
    private fun preferences(): SharedPreferences {
        val values = mutableMapOf<String, String>()
        val editor = Proxy.newProxyInstance(javaClass.classLoader, arrayOf(SharedPreferences.Editor::class.java)) { proxy, method, args ->
            when (method.name) {
                "putString" -> { values[args!![0] as String] = args[1] as String; proxy }
                "remove" -> { values.remove(args!![0] as String); proxy }
                "apply" -> null
                else -> error("Unexpected editor operation: ${method.name}")
            }
        } as SharedPreferences.Editor
        return Proxy.newProxyInstance(javaClass.classLoader, arrayOf(SharedPreferences::class.java)) { _, method, args ->
            when (method.name) {
                "getString" -> values[args!![0] as String] ?: args[1]
                "edit" -> editor
                else -> error("Unexpected preferences operation: ${method.name}")
            }
        } as SharedPreferences
    }
}
