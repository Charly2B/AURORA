import Observation

/// Infos affichées dans la carte « À la une » : complétées par la fiche (réalisateur, acteurs, résumé).
struct FeatureInfo: Sendable, Equatable {
    var eyebrow: String
    var title: String
    var credit: String?
    var overview: String?
    var genres: [String]
    var poster: String?
    var backdrop: String?

    init(card: Card) {
        let isEpisode = card.kind == "episode"
        eyebrow = isEpisode ? ["Série", AuroraFormat.episodeCode(card)].compactMap { $0 }.joined(separator: " · ")
            : card.kind == "series" ? "Série" : "Film"
        title = isEpisode ? card.seriesTitle ?? card.title : card.title
        overview = card.overview
        genres = card.genres ?? []
        poster = card.poster
        backdrop = card.backdrop ?? card.thumb
    }

    mutating func merge(_ detail: Detail) {
        credit = Self.credit(kind: detail.kind, people: detail.people)
        overview = overview ?? detail.overview
        if genres.isEmpty { genres = detail.genres ?? [] }
        poster = poster ?? detail.poster
        backdrop = backdrop ?? detail.backdrop
    }

    /// « De Christopher Nolan · avec Matthew McConaughey, Anne Hathaway » ; « Créée par … » pour une série.
    static func credit(kind: String, people: [Person]) -> String? {
        let makers = people.filter { $0.role == "director" || $0.role == "creator" }.prefix(2).map(\.name)
        let actors = people.filter { $0.role == "actor" }.prefix(2).map(\.name)
        var parts: [String] = []
        if !makers.isEmpty {
            parts.append((kind == "series" ? "Créée par " : "De ") + makers.joined(separator: " et "))
        }
        if !actors.isEmpty {
            parts.append((parts.isEmpty ? "Avec " : "avec ") + actors.joined(separator: ", "))
        }
        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }
}

@MainActor
@Observable
final class HomeModel {
    private var api: any AuroraServing
    private var origin: String
    private var generation = 0
    private var didPickRandom = false
    /// Tirage du titre à la une ; remplaçable dans les tests.
    var randomIndex: (Int) -> Int = { Int.random(in: 0..<$0) }

    private(set) var hero: Card?
    private(set) var sections: [Section] = []
    private(set) var state: CatalogLoadState = .idle
    /// Titre à la une : tiré au hasard dans toute la bibliothèque, puis suit le survol des affiches.
    private(set) var featured: Card?
    private(set) var info: [Int64: FeatureInfo] = [:]
    private(set) var movieCount: Int?
    private(set) var seriesCount: Int?

    init(api: any AuroraServing, origin: String) {
        self.api = api
        self.origin = origin
    }

    var continueItems: [Card] { sections.first { $0.id == "continue" }?.items ?? [] }

    func replace(api: any AuroraServing, origin: String) {
        generation += 1
        self.api = api
        self.origin = origin
        hero = nil
        sections = []
        state = .idle
        featured = nil
        didPickRandom = false
        info = [:]
        movieCount = nil
        seriesCount = nil
    }

    func load() async {
        generation += 1
        let request = generation
        let requestOrigin = origin
        state = .loading
        do {
            let home = try await api.home()
            guard request == generation, requestOrigin == origin else { return }
            hero = home.hero
            sections = home.sections
            state = hero == nil && sections.allSatisfy(\.items.isEmpty) ? .empty : .content
        } catch is CancellationError {
            return
        } catch {
            guard request == generation, requestOrigin == origin else { return }
            state = .error(error.localizedDescription)
            return
        }
        if featured == nil, let fallback = hero ?? sections.first(where: { !$0.items.isEmpty })?.items.first {
            feature(fallback)
        }
        await pickRandomFeature(request: request, origin: requestOrigin)
    }

    /// Met un titre à la une (tirage ou survol) et complète ses infos une seule fois.
    func feature(_ card: Card) {
        featured = card
        guard info[card.id] == nil else { return }
        info[card.id] = FeatureInfo(card: card)
        let request = generation
        let detailID = card.kind == "episode" ? card.seriesId ?? card.id : card.id
        Task {
            guard let detail = try? await api.detail(itemID: detailID), request == generation else { return }
            info[card.id]?.merge(detail)
        }
    }

    // Le serveur ne fournit pas de tirage : on prend un rang au hasard parmi tous les films et séries
    // (comptes de /libraries), puis on lit cette seule entrée. En cas d'échec, le héros serveur reste.
    private func pickRandomFeature(request: Int, origin requestOrigin: String) async {
        guard let libraries = try? await api.libraries(), request == generation, requestOrigin == origin else { return }
        let movies = libraries.reduce(0) { $0 + $1.movies }
        let series = libraries.reduce(0) { $0 + $1.series }
        movieCount = movies
        seriesCount = series
        // Un seul tirage par session : les rechargements (retour du lecteur, scan) gardent le titre.
        guard !didPickRandom, movies + series > 0 else { return }
        didPickRandom = true
        let rank = randomIndex(movies + series)
        let kind: MediaKind = rank < movies ? .movie : .series
        let offset = rank < movies ? rank : rank - movies
        guard let page = try? await api.list(kind: kind.rawValue, libraryID: nil, sort: LibrarySort.added.rawValue, descending: true, offset: offset, limit: 1, unwatched: false),
              request == generation, requestOrigin == origin,
              let card = page.items.first else { return }
        feature(card)
    }
}
