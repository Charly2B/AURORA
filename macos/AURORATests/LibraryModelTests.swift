import Testing
@testable import AURORA

@Suite("Bibliothèques", .serialized)
@MainActor
struct LibraryModelTests {
    @Test func loadsPagesAtFixedOffsetsAndKeepsFirstDuplicate() async throws {
        let server = try ServerAddress.normalize("nas")
        let api = FakeAuroraAPI(server: server, lists: [
            "movie:7:0": .value(page([card(1, "Premier"), card(2, "Original")], total: 1001)),
            "movie:7:500": .value(page([card(2, "Doublon"), card(3, "Troisième")], total: 1001)),
            "movie:7:1000": .value(page([card(4, "Quatrième")], total: 1001))
        ])

        let cards = try await CatalogLoader(api: api).loadAll(kind: .movie, libraryID: 7, sort: .added, descending: true)

        #expect(cards.map(\.id) == [1, 2, 3, 4])
        #expect(cards[1].title == "Original")
        #expect(await api.snapshot().listOffsets == [0, 500, 1000])
    }

    @Test func groupedFoldersAreNeverMerged() async throws {
        let server = try ServerAddress.normalize("nas")
        let libraries: [LibraryRef] = decode(#"[{"id":1,"name":"Films A","movies":1},{"id":2,"name":"Films B","movies":1}]"#)
        let api = FakeAuroraAPI(server: server, libraries: .value(libraries), lists: [
            "movie:1:0": .value(page([card(10, "A")], total: 1)),
            "movie:2:0": .value(page([card(20, "B")], total: 1))
        ])
        let model = LibraryModel(api: api, origin: server.origin, kind: .movie)

        await model.loadGrouped()

        #expect(model.folders.map(\.library.id) == [1, 2])
        #expect(model.folders.map { $0.items.map(\.id) } == [[10], [20]])
    }
}
