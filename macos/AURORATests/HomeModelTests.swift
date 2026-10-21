import Testing
@testable import AURORA

@Suite("Accueil", .serialized)
@MainActor
struct HomeModelTests {
    @Test func preservesAndroidSectionOrderWithoutMerging() async throws {
        let server = try ServerAddress.normalize("nas")
        let home: Home = decode(#"{"sections":[{"id":"continue","title":"Continuer","items":[{"id":1,"kind":"movie","title":"A"}]},{"id":"nextup","title":"Prochain épisode","items":[{"id":2,"kind":"episode","title":"B"}]},{"id":"recent","title":"Ajouts récents","libraryId":9,"items":[{"id":3,"kind":"movie","title":"C"}]}]}"#)
        let api = FakeAuroraAPI(server: server, home: .value(home))
        let model = HomeModel(api: api, origin: server.origin)

        await model.load()

        #expect(model.sections.map(\.id) == ["continue", "nextup", "recent"])
        #expect(model.sections.map(\.items).map { $0.map(\.id) } == [[1], [2], [3]])
    }
}
