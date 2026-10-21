import Testing
@testable import AURORA

@Suite("Recherche", .serialized)
@MainActor
struct SearchModelTests {
    @Test func emptyQueryDoesNotCallServerAndFuzzyResultsHaveAndroidLabel() async throws {
        let server = try ServerAddress.normalize("nas")
        let fuzzy: SearchResults = decode(#"{"movies":[{"id":1,"kind":"movie","title":"Alien"}],"fuzzy":true}"#)
        let api = FakeAuroraAPI(server: server, searches: ["alien": .value(fuzzy)])
        let model = SearchModel(api: api, origin: server.origin)

        await model.search("a")
        #expect(model.prompt == "Tapez au moins deux lettres.")
        await model.search("alien")
        #expect(model.fuzzyLabel == "Résultats approchants :")
        #expect(model.results?.movies.map(\.id) == [1])
    }

    @Test func responseFromPreviousServerIsIgnored() async throws {
        let oldServer = try ServerAddress.normalize("old-nas")
        let newServer = try ServerAddress.normalize("new-nas")
        let oldResults: SearchResults = decode(#"{"movies":[{"id":1,"kind":"movie","title":"Ancien"}]}"#)
        let newResults: SearchResults = decode(#"{"movies":[{"id":2,"kind":"movie","title":"Nouveau"}]}"#)
        let slow = FakeAuroraAPI(server: oldServer, searches: ["film": .value(oldResults)], searchDelay: .milliseconds(80))
        let fast = FakeAuroraAPI(server: newServer, searches: ["film": .value(newResults)])
        let model = SearchModel(api: slow, origin: oldServer.origin)

        let oldTask = Task { await model.search("film") }
        try await Task.sleep(for: .milliseconds(10))
        model.replace(api: fast, origin: newServer.origin)
        await model.search("film")
        await oldTask.value

        #expect(model.results?.movies.map(\.id) == [2])
    }
}
