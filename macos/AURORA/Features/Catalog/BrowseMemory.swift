import Foundation

enum BrowseTab: String, Hashable, Sendable {
    case home
    case movies
    case series
    case search
    case profile
}

struct BrowseKey: Hashable, Sendable {
    let origin: String
    let tab: BrowseTab
    let libraryID: Int64?
}

struct BrowseState: Sendable {
    let sort: LibrarySort
    let grouped: Bool
    let selectedLibraryID: Int64?
    let scrollTarget: Int64?
    let openedItemID: Int64?
}

@MainActor
final class BrowseMemory {
    private var values: [BrowseKey: BrowseState] = [:]

    func save(_ state: BrowseState, for key: BrowseKey) { values[key] = state }
    func restore(_ key: BrowseKey) -> BrowseState? { values[key] }
    func leave(_ key: BrowseKey) { values[key] = nil }
    func purge(origin: String) { values = values.filter { $0.key.origin != origin } }
}
