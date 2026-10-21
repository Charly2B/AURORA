import Foundation
import Observation

enum AppPhase: Equatable {
    case loading
    case serverList
    case addServer
    case login
    case authenticated
}

@MainActor
@Observable
final class AppSession {
    private let store: ServerStore
    private let apiFactory: AuroraAPIFactory

    @ObservationIgnored private(set) var api: (any AuroraServing)?
    private(set) var servers: [ServerAddress] = []
    private(set) var selectedServer: ServerAddress?
    private(set) var profile: UserProfile?
    var phase: AppPhase = .loading
    var errorMessage: String?
    private(set) var shellIdentity = UUID()
    private var activationGeneration = 0

    init(store: ServerStore = ServerStore(), apiFactory: @escaping AuroraAPIFactory = AppDependencies.liveAPI) {
        self.store = store
        self.apiFactory = apiFactory
    }

    func restore() async {
        servers = store.allServers()
        guard let server = store.selectedServer() else {
            phase = .serverList
            return
        }

        let request = activate(server)
        do {
            let status = try await api?.status()
            guard request == activationGeneration, selectedServer == server else { return }
            if let user = status?.user {
                authenticated(user)
            } else {
                phase = .login
            }
        } catch {
            guard request == activationGeneration, selectedServer == server else { return }
            errorMessage = error.localizedDescription
            phase = .login
        }
    }

    @discardableResult
    func addServer(_ input: String) async throws -> ServerAddress {
        let server = try store.add(input)
        servers = store.allServers()
        await select(server)
        return server
    }

    func select(_ server: ServerAddress) async {
        store.select(server)
        shellIdentity = UUID()
        profile = nil
        phase = .loading
        let request = activate(server)
        do {
            let status = try await api?.status()
            guard request == activationGeneration, selectedServer == server else { return }
            if let user = status?.user { authenticated(user) } else { phase = .login }
        } catch {
            guard request == activationGeneration, selectedServer == server else { return }
            errorMessage = error.localizedDescription
            phase = .login
        }
    }

    func remove(_ server: ServerAddress) async throws {
        try await store.remove(server)
        servers = store.allServers()
        if selectedServer == server {
            selectedServer = nil
            profile = nil
            api = nil
            activationGeneration += 1
            shellIdentity = UUID()
            phase = .serverList
        }
    }

    func authenticated(_ user: UserProfile) {
        profile = user
        errorMessage = nil
        phase = .authenticated
    }

    func service() -> (any AuroraServing)? {
        if api == nil, let server = store.selectedServer() {
            activate(server)
        }
        return api
    }

    func unauthorized(origin: String) async {
        guard selectedServer?.origin == origin else { return }
        try? await store.clearSession(for: origin)
        profile = nil
        shellIdentity = UUID()
        phase = .login
    }

    func showServers() {
        profile = nil
        api = nil
        selectedServer = nil
        activationGeneration += 1
        shellIdentity = UUID()
        phase = .serverList
    }

    func logout() async {
        let origin = selectedServer?.origin
        await api?.logout()
        if let origin { try? await store.clearSession(for: origin) }
        profile = nil
        shellIdentity = UUID()
        phase = .login
    }

    @discardableResult
    private func activate(_ server: ServerAddress) -> Int {
        activationGeneration += 1
        selectedServer = server
        let origin = server.origin
        api = apiFactory(server) { [weak self] in
            await self?.unauthorized(origin: origin)
        }
        return activationGeneration
    }
}
