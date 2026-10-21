protocol SessionStoring: Sendable {
    func token(for origin: String) async throws -> String?
    func setToken(_ token: String?, for origin: String) async throws
}

