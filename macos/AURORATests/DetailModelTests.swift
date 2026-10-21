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

    @Test func episodeWatchedToggleTargetsEpisodeIDAndClearsResume() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeSeriesDetail()
        let episodes = [episodeCard(7, watched: false, position: 640), episodeCard(8)]
        let api = FakeAuroraAPI(server: server, details: [42: .value(detail)], episodes: ["42:2": .value(episodes)])
        let model = DetailModel(itemID: 42, api: api)
        await model.load()
        await model.selectSeason(2)

        await model.setEpisodeWatched(model.episodes[0], watched: true)

        #expect(model.episodes[0].watched == true)
        #expect(model.episodes[0].position == 0)
        #expect(model.episodes[1].watched == false)
        let snapshot = await api.snapshot()
        #expect(snapshot.stateCalls == 1)
        #expect(snapshot.stateItemIDs == [7]) // La série entière ne doit pas être marquée vue.
    }

    @Test func episodeWatchedRollsBackWhenServerRejects() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeSeriesDetail()
        let episodes = [episodeCard(7, watched: false, position: 640)]
        let api = FakeAuroraAPI(
            server: server,
            details: [42: .value(detail)],
            episodes: ["42:2": .value(episodes)],
            setState: .failure(APIError(code: 500, message: "Refus"))
        )
        let model = DetailModel(itemID: 42, api: api)
        await model.load()
        await model.selectSeason(2)

        await model.setEpisodeWatched(model.episodes[0], watched: true)

        #expect(model.episodes[0].watched == false)
        #expect(model.episodes[0].position == 640)
        #expect(model.errorMessage == "Refus")
    }

    @Test func artworkApplyFailsShowsServerMessageAndKeepsChoices() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 0, watched: false, lastFileID: nil)
        let choices: ArtworkChoices = decode(#"{"posters":[{"path":"/abc123.jpg","preview":"https://image.tmdb.org/t/p/w342/abc123.jpg","language":"fr","votes":7.5,"width":800,"height":1200}],"backdrops":[]}"#)
        let api = FakeAuroraAPI(
            server: server,
            details: [42: .value(detail)],
            artwork: [42: .value(choices)],
            artworkSet: .failure(APIError(code: 400, message: "associez d'abord la bonne fiche TMDB"))
        )
        let model = DetailModel(itemID: 42, api: api, canEditArtwork: true)
        await model.load()
        await model.loadArtwork()

        await model.applyArtwork(type: "poster", path: "/abc123.jpg")

        #expect(model.errorMessage == "associez d'abord la bonne fiche TMDB")
        #expect(model.artwork?.posters.first?.path == "/abc123.jpg") // les choix restent affichés
        #expect(await api.snapshot().artworkSetCalls == ["42:poster:/abc123.jpg"])
    }

    @Test func artworkApplyRequestsServerAndResetsChoices() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 0, watched: false, lastFileID: nil)
        let choices: ArtworkChoices = decode(#"{"posters":[{"path":"/ok.jpg","preview":"https://img/ok.jpg"}],"backdrops":[]}"#)
        let api = FakeAuroraAPI(
            server: server,
            details: [42: .value(detail)],
            artwork: [42: .value(choices)]
        )
        let model = DetailModel(itemID: 42, api: api, canEditArtwork: true)
        await model.load()
        await model.loadArtwork()

        await model.applyArtwork(type: "poster", path: "/ok.jpg")

        #expect(await api.snapshot().artworkSetCalls == ["42:poster:/ok.jpg"])
        #expect(model.artwork == nil)
        #expect(model.detail != nil) // la fiche a été rechargée
        #expect(model.artworkState == .idle)
    }

    @Test func artworkHiddenForNonAdmin() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 0, watched: false, lastFileID: nil)
        let api = FakeAuroraAPI(server: server, details: [42: .value(detail)])
        let model = DetailModel(itemID: 42, api: api, canEditArtwork: false)
        await model.load()

        await model.loadArtwork()

        #expect(model.artwork == nil)
        #expect(model.artworkState == .idle)
        #expect(await api.snapshot().artworkSetCalls == [])
    }

    @Test func identificationSearchLoadsCandidatesThenMatchReloadsDetail() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 0, watched: false, lastFileID: nil)
        let january: [TMDbCandidate] = decode(#"[{"id":11,"title":"Star Wars","year":1977,"overview":"Un film.","popularity":80.0}]"#)
        let retour: [TMDbCandidate] = decode(#"[{"id":1891,"title":"Retour vers le futur","originalTitle":"Back to the Future","year":1985,"poster":"https://image.tmdb.org/t/p/w185/btf.jpg"}]"#)
        let api = FakeAuroraAPI(
            server: server,
            details: [42: .value(detail)],
            candidates: ["42|": .value(january), "42|futur": .value(retour)]
        )
        let model = DetailModel(itemID: 42, api: api, canEditArtwork: true)
        await model.load()

        await model.loadCandidates(query: "")
        #expect(model.candidates.first?.id == 11)
        #expect(model.candidatesState == .content)

        await model.loadCandidates(query: "futur")
        #expect(model.candidates.first?.title == "Retour vers le futur")

        await model.applyMatch(1891)

        #expect(await api.snapshot().matchCalls == ["42->1891"])
        #expect(model.candidates.isEmpty)
        #expect(model.candidatesState == .idle)
        #expect(model.detail != nil) // la fiche est rechargée après association
    }

    @Test func identificationMatchFailureKeepsMessageAndCandidates() async throws {
        let server = try ServerAddress.normalize("nas")
        let detail = makeDetail(position: 0, watched: false, lastFileID: nil)
        let january: [TMDbCandidate] = decode(#"[{"id":11,"title":"Star Wars"}]"#)
        let api = FakeAuroraAPI(
            server: server,
            details: [42: .value(detail)],
            candidates: ["42|": .value(january)],
            match: .failure(APIError(code: 502, message: "l'association a échoué"))
        )
        let model = DetailModel(itemID: 42, api: api, canEditArtwork: true)
        await model.load()
        await model.loadCandidates(query: "")

        await model.applyMatch(11)

        #expect(model.errorMessage == "l'association a échoué")
        #expect(model.candidates.first?.id == 11) // la liste reste ouverte pour un autre essai
        #expect(model.candidatesState == .content)
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

    private func makeSeriesDetail() -> Detail {
        decode(#"{"id":42,"kind":"series","title":"Série","files":[],"state":{},"seasons":[{"number":1,"title":"Saison 1"},{"number":2,"title":"Saison 2"}]}"#)
    }

    private func episodeCard(_ id: Int64, watched: Bool = false, position: Double = 0) -> Card {
        decode(#"{"id":\#(id),"kind":"episode","title":"Épisode \#(id)","season":2,"episode":\#(id),"duration":1800,"position":\#(position),"watched":\#(watched),"overview":""}"#)
    }
}
