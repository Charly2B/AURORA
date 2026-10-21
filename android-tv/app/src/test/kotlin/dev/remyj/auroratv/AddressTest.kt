package dev.remyj.auroratv

import dev.remyj.auroratv.data.normalizeAddress
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class AddressTest {
    @Test
    fun normalise() {
        assertEquals("http://192.168.1.20:8890", normalizeAddress("192.168.1.20"))
        assertEquals("http://192.168.1.20:8890", normalizeAddress(" 192.168.1.20/ "))
        assertEquals("http://192.168.1.20:5050", normalizeAddress("192.168.1.20:5050"))
        assertEquals("https://nas.example:443", normalizeAddress("https://nas.example:443"))
        assertEquals("http://nas.local:8890", normalizeAddress("nas.local"))
        assertEquals("http://[fe80::1]:8890", normalizeAddress("http://[fe80::1]"))
        assertNull(normalizeAddress(""))
        assertNull(normalizeAddress("192.168 .1.2"))
    }
}
