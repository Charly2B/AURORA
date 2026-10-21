import Observation

@MainActor
@Observable
final class SearchModel {
    private var api: any AuroraServing
    private var origin: String
    private var generation = 0

    var query = ""
    private(set) var results: SearchResults?
    private(set) var state: CatalogLoadState = .idle
    private(set) var prompt = "Tapez au moins deux lettres."

    var fuzzyLabel: String? { results?.fuzzy == true ? "Résultats approchants :" : nil }

    init(api: any AuroraServing, origin: String) {
        self.api = api
        self.origin = origin
    }

    func replace(api: any AuroraServing, origin: String) {
        generation += 1
        self.api = api
        self.origin = origin
        results = nil
        state = .idle
    }

    func search(_ rawQuery: String) async {
        query = rawQuery
        generation += 1
        let request = generation
        let requestOrigin = origin
        let trimmed = rawQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 2 else {
            results = nil
            prompt = "Tapez au moins deux lettres."
            state = .idle
            return
        }

        state = .loading
        do {
            let response = try await api.search(trimmed)
            guard request == generation, requestOrigin == origin else { return }
            results = response
            let isEmpty = response.movies.isEmpty && response.series.isEmpty && response.episodes.isEmpty
            prompt = isEmpty ? "Aucun résultat pour « \(trimmed) »." : ""
            state = isEmpty ? .empty : .content
        } catch is CancellationError {
        } catch {
            guard request == generation, requestOrigin == origin else { return }
            state = .error(error.localizedDescription)
        }
    }
}
