import Foundation

@MainActor
final class ServerStore {
    private enum Keys {
        static let servers = "aurora.servers"
        static let selected = "aurora.selectedServer"
    }

    private let defaults: UserDefaults
    private let sessions: any SessionStoring
    private let encoder = JSONEncoder()
    private let decoder = JSONDecoder()

    init(defaults: UserDefaults = .standard, sessions: any SessionStoring = KeychainSessionStore()) {
        self.defaults = defaults
        self.sessions = sessions
    }

    func allServers() -> [ServerAddress] {
        guard let data = defaults.data(forKey: Keys.servers) else { return [] }
        return (try? decoder.decode([ServerAddress].self, from: data)) ?? []
    }

    func selectedServer() -> ServerAddress? {
        let servers = allServers()
        guard !servers.isEmpty else { return nil }
        guard let origin = defaults.string(forKey: Keys.selected) else { return servers.first }
        return servers.first(where: { $0.origin == origin }) ?? servers.first
    }

    @discardableResult
    func add(_ input: String) throws -> ServerAddress {
        let server = try ServerAddress.normalize(input)
        var servers = allServers()
        if let existing = servers.first(where: { $0 == server }) { return existing }
        servers.append(server)
        persist(servers)
        if defaults.string(forKey: Keys.selected) == nil {
            defaults.set(server.origin, forKey: Keys.selected)
        }
        return server
    }

    func select(_ server: ServerAddress) {
        guard allServers().contains(server) else { return }
        defaults.set(server.origin, forKey: Keys.selected)
    }

    func remove(_ server: ServerAddress) async throws {
        let remaining = allServers().filter { $0 != server }
        persist(remaining)
        try await sessions.setToken(nil, for: server.origin)
        if defaults.string(forKey: Keys.selected) == server.origin {
            defaults.set(remaining.first?.origin, forKey: Keys.selected)
        }
    }

    func clearSession(for origin: String) async throws {
        try await sessions.setToken(nil, for: origin)
    }

    private func persist(_ servers: [ServerAddress]) {
        if let data = try? encoder.encode(servers) {
            defaults.set(data, forKey: Keys.servers)
        }
    }
}
