import Foundation

typealias AuroraAPIFactory = @MainActor @Sendable (
    ServerAddress,
    @escaping @Sendable () async -> Void
) -> any AuroraServing

enum AppDependencies {
    @MainActor
    static let liveAPI: AuroraAPIFactory = { server, onUnauthorized in
        AuroraAPI(server: server, onUnauthorized: onUnauthorized)
    }
}
