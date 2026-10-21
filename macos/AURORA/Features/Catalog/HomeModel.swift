import Observation

@MainActor
@Observable
final class HomeModel {
    private var api: any AuroraServing
    private var origin: String
    private var generation = 0

    private(set) var hero: Card?
    private(set) var sections: [Section] = []
    private(set) var state: CatalogLoadState = .idle

    init(api: any AuroraServing, origin: String) {
        self.api = api
        self.origin = origin
    }

    func replace(api: any AuroraServing, origin: String) {
        generation += 1
        self.api = api
        self.origin = origin
        hero = nil
        sections = []
        state = .idle
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
        } catch {
            guard request == generation, requestOrigin == origin else { return }
            state = .error(error.localizedDescription)
        }
    }
}
