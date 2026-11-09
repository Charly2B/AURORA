package dev.remyj.auroratv

import dev.remyj.auroratv.usb.formatBytes
import dev.remyj.auroratv.usb.formatDuration
import dev.remyj.auroratv.usb.isVideoFile
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class UsbSupportTest {
    @Test
    fun detecteLesVideos() {
        assertTrue(isVideoFile("Inception (2010).mkv", null))
        assertTrue(isVideoFile("Dune.2021.MP4", null))
        assertTrue(isVideoFile("clip", "video/mp4"))
        assertFalse(isVideoFile("cover.jpg", null))
        assertFalse(isVideoFile("readme.txt", null))
        assertFalse(isVideoFile("sans-extension", null))
    }

    @Test
    fun formatDesTailles() {
        assertEquals("640 Mo", formatBytes(640_000_000))
        assertEquals("8,2 Go", formatBytes(8_200_000_000))
        assertEquals("1,8 To", formatBytes(1_800_000_000_000))
        assertEquals("", formatBytes(-1))
    }

    @Test
    fun formatDesDurees() {
        assertEquals("45 min", formatDuration(45 * 60))
        assertEquals("2 h 28", formatDuration(2 * 3600 + 28 * 60))
        assertEquals("", formatDuration(0))
    }
}
