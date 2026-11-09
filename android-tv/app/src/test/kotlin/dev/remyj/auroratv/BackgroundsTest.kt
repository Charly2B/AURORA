package dev.remyj.auroratv

import dev.remyj.auroratv.net.ApiException
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.Sessions
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

class BackgroundsTest {
    private lateinit var server: MockWebServer
    private lateinit var http: OkHttpClient
    private lateinit var client: AuroraClient
    private val id = "a".repeat(64)

    @Before fun setUp() {
        server = MockWebServer().apply { start() }
        http = OkHttpClient.Builder().addInterceptor(Sessions.interceptor()).build()
        client = AuroraClient(http, server.url("/").toString().trimEnd('/'))
        Sessions.set(client.baseUrl, "session-test")
    }

    @After fun tearDown() {
        Sessions.set(client.baseUrl, null)
        server.shutdown()
    }

    @Test fun galleryDecodesRealContractAndAuthenticatesListAndImage() = runBlocking {
        server.enqueue(MockResponse().setBody("""{"images":[{"id":"$id","name":"Montagnes.jpg","url":"/api/v1/backgrounds/$id/image","width":3840,"height":2160,"size":3245678901,"createdAt":4294967296}]}"""))
        val image = client.backgrounds().single()
        assertEquals(id, image.id)
        assertEquals("Montagnes.jpg", image.name)
        assertEquals(3840, image.width)
        assertEquals(2160, image.height)
        assertEquals(3245678901L, image.size)
        assertEquals(4294967296L, image.createdAt)
        val list = server.takeRequest()
        assertEquals("/api/v1/backgrounds", list.path)
        assertEquals("aurora_session=session-test", list.getHeader("Cookie"))
        val url = requireNotNull(client.backgroundUrl(image.id))
        assertEquals(client.absolute(image.url), url)
        assertFalse(url.contains("session-test"))
        server.enqueue(MockResponse().setBody("image bytes"))
        http.newCall(Request.Builder().url(url).build()).execute().use { assertTrue(it.isSuccessful) }
        assertEquals("aurora_session=session-test", server.takeRequest().getHeader("Cookie"))
    }

    @Test fun emptyGalleryIsValid() = runBlocking {
        server.enqueue(MockResponse().setBody("""{"images":[]}"""))
        assertTrue(client.backgrounds().isEmpty())
    }

    @Test fun unavailableGalleryAndInvalidJsonRemainDisplayableErrors() = runBlocking {
        for ((response, code) in listOf(
            MockResponse().setResponseCode(404).setBody("""{"error":"introuvable"}""") to 404,
            MockResponse().setResponseCode(200).setBody("<html>proxy</html>") to 0,
            MockResponse().setResponseCode(200).setBody("""{"images":[{"id":17}]}""") to 0,
        )) {
            server.enqueue(response)
            try {
                client.backgrounds()
                fail("Expected ApiException")
            } catch (e: ApiException) { assertEquals(code, e.code) }
        }
    }

    @Test fun expiredGallerySessionUsesExistingUnauthorizedCallback() = runBlocking {
        var unauthorized = false
        client.onUnauthorized = { unauthorized = true }
        server.enqueue(MockResponse().setResponseCode(401).setBody("""{"error":"session expirée"}"""))
        try { client.backgrounds(); fail("Expected 401") }
        catch (e: ApiException) { assertEquals(401, e.code) }
        assertTrue(unauthorized)
    }

    @Test fun savedIdentifierResolvesOnlyAgainstCurrentNas() {
        val other = AuroraClient(http, "http://other-nas:8890")
        assertEquals("http://other-nas:8890/api/v1/backgrounds/$id/image", other.backgroundUrl(id))
        assertNull(other.backgroundUrl("https://another-host/image"))
    }

    @Test fun offlineNasRemainsDisplayableNetworkError() = runBlocking {
        val offline = AuroraClient(http, "http://127.0.0.1:1")
        try { offline.backgrounds(); fail("Expected network error") }
        catch (e: ApiException) { assertEquals(0, e.code) }
    }
}
