import Observation

struct PlayIntent: Equatable, Hashable, Sendable {
    let itemID: Int64
    let fileID: Int64?
    let startPosition: Double
    let restart: Bool
}

@MainActor
@Observable
final class DetailModel {
    let itemID: Int64
    private let api: any AuroraServing
    private let onChanged: @MainActor () -> Void
    private var seasonGeneration = 0

    private(set) var detail: Detail?
    private(set) var selectedSeason: Int?
    private(set) var episodes: [Card] = []
    private(set) var state: CatalogLoadState = .idle
    private(set) var episodesState: CatalogLoadState = .idle
    var errorMessage: String?

    init(itemID: Int64, api: any AuroraServing, onChanged: @escaping @MainActor () -> Void = {}) {
        self.itemID = itemID
        self.api = api
        self.onChanged = onChanged
    }

    func load() async {
        state = .loading
        do {
            let loaded = try await api.detail(itemID: itemID)
            detail = loaded
            selectedSeason = loaded.seasons?.first?.number
            state = .content
            if loaded.kind == "series", let selectedSeason {
                await selectSeason(selectedSeason)
            }
        } catch {
            state = .error(error.localizedDescription)
        }
    }

    func selectSeason(_ season: Int) async {
        seasonGeneration += 1
        let request = seasonGeneration
        selectedSeason = season
        episodesState = .loading
        do {
            let loaded = try await api.episodes(seriesID: itemID, season: season)
            guard request == seasonGeneration, selectedSeason == season else { return }
            episodes = loaded
            episodesState = episodes.isEmpty ? .empty : .content
        } catch {
            guard request == seasonGeneration, selectedSeason == season else { return }
            episodesState = .error(error.localizedDescription)
        }
    }

    func playIntent(restart: Bool, fileID: Int64?) -> PlayIntent? {
        guard let detail else { return nil }
        let file = fileID.flatMap { requested in detail.files.first { $0.id == requested } }
            ?? detail.state.lastFileId.flatMap { previous in detail.files.first { $0.id == previous } }
            ?? detail.files.first
        guard let file else { return nil }
        let resumable = detail.state.position > 0 && !detail.state.watched && !restart
        return PlayIntent(itemID: detail.id, fileID: file.id, startPosition: resumable ? detail.state.position : 0, restart: restart)
    }

    func toggleFavorite() async {
        await mutate { value in
            value.state.favorite.toggle()
            value.favorite = value.state.favorite
        } request: { value in
            try await api.setState(itemID: itemID, favorite: value.state.favorite, watchlist: nil, watched: nil, season: nil)
        }
    }

    func toggleWatchlist() async {
        await mutate { value in
            value.state.watchlist.toggle()
            value.watchlist = value.state.watchlist
        } request: { value in
            try await api.setState(itemID: itemID, favorite: nil, watchlist: value.state.watchlist, watched: nil, season: nil)
        }
    }

    func setWatched(_ watched: Bool) async {
        await mutate { value in
            value.state.watched = watched
            value.watched = watched
        } request: { _ in
            try await api.setState(itemID: itemID, favorite: nil, watchlist: nil, watched: watched, season: selectedSeason)
        }
    }

    func hideContinue(onHidden: @MainActor (Int64) -> Void) async {
        do {
            try await api.hideContinue(itemID: itemID)
            onHidden(itemID)
            onChanged()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func mutate(
        _ mutation: (inout Detail) -> Void,
        request: (Detail) async throws -> Void
    ) async {
        guard let original = detail else { return }
        var optimistic = original
        mutation(&optimistic)
        detail = optimistic
        errorMessage = nil
        do {
            try await request(optimistic)
            onChanged()
        } catch {
            detail = original
            errorMessage = error.localizedDescription
        }
    }
}
