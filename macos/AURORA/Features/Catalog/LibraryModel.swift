import Observation

@MainActor
@Observable
final class LibraryModel {
    private var api: any AuroraServing
    private var origin: String
    private var generation = 0
    private var grouped = false

    let kind: MediaKind
    var selectedLibraryID: Int64?
    var sort: LibrarySort
    private(set) var libraries: [LibraryRef] = []
    private(set) var items: [Card] = []
    private(set) var folders: [FolderContent] = []
    private(set) var state: CatalogLoadState = .idle

    init(api: any AuroraServing, origin: String, kind: MediaKind) {
        self.api = api
        self.origin = origin
        self.kind = kind
        sort = kind == .movie ? .added : .title
    }

    func replace(api: any AuroraServing, origin: String) {
        generation += 1
        self.api = api
        self.origin = origin
        libraries = []
        items = []
        folders = []
        state = .idle
    }

    func loadSelected() async {
        grouped = false
        generation += 1
        let request = generation
        let requestOrigin = origin
        state = .loading
        do {
            if libraries.isEmpty { libraries = try await filteredLibraries() }
            let loaded = try await CatalogLoader(api: api).loadAll(
                kind: kind,
                libraryID: selectedLibraryID ?? libraries.first?.id,
                sort: sort,
                descending: sort.descending
            )
            guard request == generation, requestOrigin == origin else { return }
            items = loaded
            folders = []
            state = loaded.isEmpty ? .empty : .content
        } catch is CancellationError {
        } catch {
            guard request == generation, requestOrigin == origin else { return }
            state = .error(error.localizedDescription)
        }
    }

    func loadGrouped() async {
        grouped = true
        generation += 1
        let request = generation
        let requestOrigin = origin
        state = .loading
        do {
            let refs = try await filteredLibraries()
            var loaded: [FolderContent] = []
            for library in refs {
                let cards = try await CatalogLoader(api: api).loadAll(
                    kind: kind,
                    libraryID: library.id,
                    sort: sort,
                    descending: sort.descending
                )
                loaded.append(FolderContent(library: library, items: cards))
            }
            guard request == generation, requestOrigin == origin else { return }
            libraries = refs
            folders = loaded
            items = []
            state = loaded.allSatisfy(\.items.isEmpty) ? .empty : .content
        } catch is CancellationError {
        } catch {
            guard request == generation, requestOrigin == origin else { return }
            state = .error(error.localizedDescription)
        }
    }

    func reload() async {
        if grouped { await loadGrouped() } else { await loadSelected() }
    }

    private func filteredLibraries() async throws -> [LibraryRef] {
        try await api.libraries().filter { kind == .movie ? $0.movies > 0 : $0.series > 0 }
    }
}
