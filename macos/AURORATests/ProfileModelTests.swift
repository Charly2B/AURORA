import Testing
@testable import AURORA

@Suite("Profil", .serialized)
@MainActor
struct ProfileModelTests {
    @Test func exposesUserServerVersionAndCapabilities() async throws {
        let server = try ServerAddress.normalize("nas")
        let api = FakeAuroraAPI(server: server, status: [.value(testStatus(user: testUser("alice")))])
        let model = ProfileModel(
            user: testUser("alice"),
            server: server,
            api: api,
            capabilities: PlaybackCapabilities.detect { _ in true }
        )

        await model.load()

        #expect(model.username == "alice")
        #expect(model.serverAddress == "http://nas:8890")
        #expect(model.serverVersion == "0.1.8")
        #expect(model.playbackSummary.contains("2160p"))
    }
}
