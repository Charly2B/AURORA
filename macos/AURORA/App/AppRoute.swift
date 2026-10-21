import Foundation

enum AppRoute: Hashable {
    case home
    case detail(itemID: Int64)
    case player(PlayIntent, title: String)

    var identity: String {
        switch self {
        case .home: "home"
        case .detail(let itemID): "detail:\(itemID)"
        case .player(let intent, _): "player:\(intent.itemID):\(intent.fileID.map(String.init) ?? "auto"):\(intent.startPosition):\(intent.restart)"
        }
    }

    @MainActor
    static func initial(for session: AppSession) -> AppRoute? {
        session.phase == .authenticated ? .home : nil
    }
}

enum MainTab: String, CaseIterable, Identifiable {
    case home
    case movies
    case series
    case search
    case profile

    var id: String { rawValue }
    var label: String {
        switch self {
        case .home: "Accueil"
        case .movies: "Films"
        case .series: "Séries"
        case .search: "Recherche"
        case .profile: "Profil"
        }
    }
    var symbol: String {
        switch self {
        case .home: "house"
        case .movies: "film"
        case .series: "tv"
        case .search: "magnifyingglass"
        case .profile: "person.crop.circle"
        }
    }
}
