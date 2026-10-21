import Testing
@testable import AURORA

@Suite("Fiche média", .serialized)
@MainActor
struct DetailModelTests {
    @Test func playUsesLastFileThenResumeUnlessRestartedOrWatched() async throws {
        let detail = makeDetail(position: 125, watched: false, lastFileID: 20)
        let model = try await makeModel(detail: detail)

        let resume = model.playIntent(restart: false, fileID: nil)
        let restart = model.playIntent(restart: true, fileID: nil)

        #expect(resume?.fileID == 20)
        #expect(resume?.startPosition == 125)
        #expect(restart?.fileID == 20)
        #expect(restart?.startPosition == 0)

        let watched = try await makeModel(detail: makeDetail(position: 125, watched: true, lastFileID: 20))
        #expect(watched.playIntent(restart: false, fileID: nil)?.startPosition == 0)
    }

    @Test func fallsBackToFirstFileAndReloadsSelectedSeason() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 0, watched: false, lastFileID: 999)
        let episodes = [card(7, "Épisode 1")]
        let api = FakeAuroraAPI(server: server, details: [42: .value(detail)], episodes: ["42:2": .value(episodes)])
        let model = DetailModel(itemID: 42, api: api)
        await model.load()

        #expect(model.playIntent(restart: false, fileID: nil)?.fileID == 10)
        await model.selectSeason(2)
        #expect(model.selectedSeason == 2)
        #expect(model.episodes.map(\.id) == [7])
    }

    @Test func optimisticFavoriteRollsBackWhenServerRejects() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 0, watched: false, lastFileID: nil)
        let api = FakeAuroraAPI(server: server, details: [42: .value(detail)], setState: .failure(APIError(code: 500, message: "Refus")))
        let model = DetailModel(itemID: 42, api: api)
        await model.load()

        await model.toggleFavorite()

        #expect(model.detail?.state.favorite == false)
        #expect(model.errorMessage == "Refus")
        #expect(await api.snapshot().stateCalls == 1)
    }

    @Test func hideContinuePreservesProgressAndRemovesRowAfterSuccess() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 89, watched: false, lastFileID: 10)
        let api = FakeAuroraAPI(server: server, details: [42: .value(detail)])
        let model = DetailModel(itemID: 42, api: api)
        await model.load()
        var removed: Int64?

        await model.hideContinue { removed = $0 }

        #expect(model.detail?.state.position == 89)
        #expect(removed == 42)
        #expect(await api.snapshot().hiddenItemIDs == [42])
    }

    @Test func slowOldSeasonCannotReplaceNewSeason() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail: Detail = decode(#"{"id":42,"kind":"series","title":"Série","files":[],"state":{},"seasons":[{"number":1,"title":"Saison 1"},{"number":2,"title":"Saison 2"}]}"#)
        let api = FakeAuroraAPI(
            server: server,
            details: [42: .value(detail)],
            episodes: ["42:1": .value([card(1, "Ancien")]), "42:2": .value([card(2, "Nouveau")])],
            episodeDelays: ["42:1": .milliseconds(80), "42:2": .milliseconds(10)]
        )
        let model = DetailModel(itemID: 42, api: api)
        await model.load()

        async let old: Void = model.selectSeason(1)
        try await Task.sleep(for: .milliseconds(5))
        async let new: Void = model.selectSeason(2)
        _ = await (old, new)

        #expect(model.selectedSeason == 2)
        #expect(model.episodes.map(\.id) == [2])
    }

    private func makeModel(detail: Detail) async throws -> DetailModel {
        let server = try ServerAddress.normalize("nas")
        let api = FakeAuroraAPI(server: server, details: [42: .value(detail)])
        let model = DetailModel(itemID: 42, api: api)
        await model.load()
        return model
    }

    private func makeDetail(position: Double, watched: Bool, lastFileID: Int64?) -> Detail {
        let last = lastFileID.map(String.init) ?? "null"
        return decode(#"{"id":42,"kind":"movie","title":"Film","files":[{"id":10,"name":"A.mkv","duration":7200},{"id":20,"name":"B.mkv","duration":7200}],"state":{"position":\#(position),"duration":7200,"watched":\#(watched),"lastFileId":\#(last)},"seasons":[{"number":1,"title":"Saison 1"},{"number":2,"title":"Saison 2"}]}"#)
    }
}
