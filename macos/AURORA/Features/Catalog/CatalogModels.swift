import Foundation

enum MediaKind: String, CaseIterable, Codable, Sendable {
    case movie
    case series

    var title: String { self == .movie ? "Films" : "Séries" }
    var emptyMessage: String { self == .movie ? "Aucun film." : "Aucune série." }
}

enum LibrarySort: String, CaseIterable, Codable, Sendable {
    case added
    case title
    case year
    case rating

    var label: String {
        switch self {
        case .added: "Ajout récent"
        case .title: "Titre"
        case .year: "Année"
        case .rating: "Note"
        }
    }

    var descending: Bool { self == .added || self == .year || self == .rating }

    static func available(for kind: MediaKind) -> [LibrarySort] {
        kind == .movie ? [.added, .title, .year, .rating] : [.title, .added, .year]
    }
}

enum CatalogLoadState: Equatable {
    case idle
    case loading
    case content
    case empty
    case error(String)
}

struct FolderContent: Identifiable, Sendable {
    let library: LibraryRef
    let items: [Card]
    var id: Int64 { library.id }
}

struct CatalogLoader: Sendable {
    let api: any AuroraServing
    private let pageSize = 500

    func loadAll(
        kind: MediaKind,
        libraryID: Int64?,
        sort: LibrarySort,
        descending: Bool
    ) async throws -> [Card] {
        var offset = 0
        var total = Int.max
        var items: [Card] = []
        var seen = Set<Int64>()

        while offset < total {
            try Task.checkCancellation()
            let page = try await api.list(
                kind: kind.rawValue,
                libraryID: libraryID,
                sort: sort.rawValue,
                descending: descending,
                offset: offset,
                limit: pageSize,
                unwatched: false
            )
            total = page.total
            guard !page.items.isEmpty else { break }
            for card in page.items where seen.insert(card.id).inserted {
                items.append(card)
            }
            offset += pageSize
        }
        return items
    }
}
