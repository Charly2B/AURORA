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

    @Test func featuresRandomTitleFromWholeLibraryOncePerSession() async throws {
        let server = try ServerAddress.normalize("nas")
        let home: Home = decode(#"{"hero":{"id":7,"kind":"movie","title":"Héros"},"sections":[]}"#)
        let libraries: [LibraryRef] = decode(#"[{"id":1,"name":"Films","movies":3,"series":0},{"id":2,"name":"Séries","movies":0,"series":2}]"#)
        let page: ListPage = decode(#"{"items":[{"id":42,"kind":"series","title":"Tirée"}],"total":2}"#)
        // Rang 4 sur 5 : au-delà des 3 films, donc 2e série (offset 1).
        let api = FakeAuroraAPI(server: server, home: .value(home), libraries: .value(libraries), lists: ["series:-1:1": .value(page)])
        let model = HomeModel(api: api, origin: server.origin)
        var draws: [Int] = []
        model.randomIndex = { draws.append($0); return 4 }

        await model.load()
        #expect(model.featured?.id == 42)
        #expect(model.movieCount == 3 && model.seriesCount == 2)
        #expect(draws == [5])

        await model.load()
        #expect(model.featured?.id == 42)
        #expect(draws == [5])
    }

    @Test func creditLineNamesMakersThenActors() {
        let people: [Person] = decode(#"[{"name":"Christopher Nolan","role":"director"},{"name":"Matthew McConaughey","role":"actor"},{"name":"Anne Hathaway","role":"actor"},{"name":"Jessica Chastain","role":"actor"}]"#)
        #expect(FeatureInfo.credit(kind: "movie", people: people) == "De Christopher Nolan · avec Matthew McConaughey, Anne Hathaway")
        let creators: [Person] = decode(#"[{"name":"Rachel Kondo","role":"creator"},{"name":"Justin Marks","role":"creator"}]"#)
        #expect(FeatureInfo.credit(kind: "series", people: creators) == "Créée par Rachel Kondo et Justin Marks")
        #expect(FeatureInfo.credit(kind: "movie", people: []) == nil)
    }
}
