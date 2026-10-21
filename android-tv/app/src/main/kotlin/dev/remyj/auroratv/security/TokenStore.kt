package dev.remyj.auroratv.security

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.security.GeneralSecurityException
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * Équivalent du Keychain de la version macOS : les jetons de session AURORA ne sont jamais stockés en clair.
 * Chiffrement AES/GCM avec une clé détenue par l'AndroidKeyStore (jamais exportable).
 *
 * Repris de glassplayer / PlexSeries TV (en service sur la box). Un magasin par serveur AURORA.
 */
class TokenStore(context: Context, private val name: String) {
    private val preferences = context.applicationContext.getSharedPreferences(
        "auroratv.session.$name",
        Context.MODE_PRIVATE,
    )

    fun put(token: String) {
        require(token.isNotEmpty())
        require(token.all { it.code in 0..127 }) { "Jeton invalide" }
        val plaintext = ByteArray(token.length) { token[it].code.toByte() }
        try {
            val cipher = Cipher.getInstance(TRANSFORMATION).apply {
                init(Cipher.ENCRYPT_MODE, encryptionKey())
            }
            val iv = requireNotNull(cipher.iv).also {
                require(it.size == IV_SIZE_BYTES) { "Taille d'IV AES/GCM inattendue" }
            }
            val encrypted = cipher.doFinal(plaintext)
            preferences.edit()
                .putString(TOKEN_KEY, Base64.encodeToString(iv + encrypted, Base64.NO_WRAP))
                .apply()
        } finally {
            plaintext.fill(0)
        }
    }

    fun get(): String? {
        val encoded = preferences.getString(TOKEN_KEY, null) ?: return null
        return try {
            val payload = Base64.decode(encoded, Base64.NO_WRAP)
            require(payload.size > IV_SIZE_BYTES) { "Jeton chiffré invalide" }
            val iv = payload.copyOfRange(0, IV_SIZE_BYTES)
            val encrypted = payload.copyOfRange(IV_SIZE_BYTES, payload.size)
            val plaintext = Cipher.getInstance(TRANSFORMATION).run {
                init(Cipher.DECRYPT_MODE, encryptionKey(), GCMParameterSpec(TAG_SIZE_BITS, iv))
                doFinal(encrypted)
            }
            try {
                String(CharArray(plaintext.size) { (plaintext[it].toInt() and 0xff).toChar() })
            } finally {
                plaintext.fill(0)
            }
        } catch (_: GeneralSecurityException) {
            clear()
            null
        } catch (_: IllegalArgumentException) {
            clear()
            null
        }
    }

    fun clear() {
        preferences.edit().remove(TOKEN_KEY).apply()
    }

    private fun encryptionKey(): SecretKey {
        val keyStore = KeyStore.getInstance(KEYSTORE_PROVIDER).apply { load(null) }
        (keyStore.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        return KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE_PROVIDER).run {
            init(
                KeyGenParameterSpec.Builder(
                    KEY_ALIAS,
                    KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
                )
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .setKeySize(256)
                    .build(),
            )
            generateKey()
        }
    }

    private companion object {
        const val TOKEN_KEY = "token"
        const val KEYSTORE_PROVIDER = "AndroidKeyStore"
        const val KEY_ALIAS = "auroratv.session"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val IV_SIZE_BYTES = 12
        const val TAG_SIZE_BITS = 128
    }
}
