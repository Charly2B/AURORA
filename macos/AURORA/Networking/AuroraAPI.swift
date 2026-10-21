import Foundation

protocol AuroraServing: Sendable {
    func status() async throws -> Status
    func login(username: String, password: String) async throws -> UserProfile
    func linkStart(deviceName: String) async throws -> LinkStart
    func linkPoll(secret: String) async throws -> UserProfile?
    func me() async throws -> UserProfile
    func logout() async
    func home() async throws -> Home
    func libraries() async throws -> [LibraryRef]
    func list(kind: String, libraryID: Int64?, sort: String, descending: Bool, offset: Int, limit: Int, unwatched: Bool) async throws -> ListPage
    func detail(itemID: Int64) async throws -> Detail
    func episodes(seriesID: Int64, season: Int) async throws -> [Card]
    func next(episodeID: Int64) async throws -> Card?
    func search(_ query: String) async throws -> SearchResults
    func play(_ request: PlayRequest) async throws -> PlayResponse
    func progress(itemID: Int64, fileID: Int64, position: Double, duration: Double) async
    func setState(itemID: Int64, favorite: Bool?, watchlist: Bool?, watched: Bool?, season: Int?) async throws
    func hideContinue(itemID: Int64) async throws
    func stopSession(id: String) async
    func imageData(path: String) async throws -> Data
    nonisolated func absoluteURL(_ path: String?) -> URL?
}

actor AuroraAPI: AuroraServing {
    private struct ErrorBody: Decodable { let error: String }
    private struct Credentials: Encodable { let username: String; let password: String }
    private struct LinkName: Encodable { let name: String }
    private struct LinkPoll: Encodable { let poll: String }
    private struct LinkStatus: Decodable { let status: String }
    private struct NextResponse: Decodable { let next: Card? }
    private struct ProgressBody: Encodable { let fileId: Int64; let position: Double; let duration: Double }
    private struct StateBody: Encodable { let favorite: Bool?; let watchlist: Bool?; let watched: Bool?; let season: Int? }

    nonisolated let server: ServerAddress
    private let transport: HTTPTransporting
    private let sessions: SessionStoring
    private let onUnauthorized: @Sendable () async -> Void
    private let decoder = JSONDecoder()
    private let encoder = JSONEncoder()

    init(
        server: ServerAddress,
        transport: HTTPTransporting = HTTPTransport(),
        sessions: SessionStoring = KeychainSessionStore(),
        onUnauthorized: @escaping @Sendable () async -> Void = {}
    ) {
        self.server = server
        self.transport = transport
        self.sessions = sessions
        self.onUnauthorized = onUnauthorized
    }

    nonisolated func absoluteURL(_ path: String?) -> URL? {
        guard let path, !path.isEmpty else { return nil }
        if let url = URL(string: path), url.scheme != nil { return url }
        return URL(string: server.origin + (path.hasPrefix("/") ? path : "/\(path)"))
    }

    func status() async throws -> Status { try await get("/status") }

    func login(username: String, password: String) async throws -> UserProfile {
        let body = try encoder.encode(Credentials(username: username, password: password))
        let (_, response) = try await call(method: "POST", path: "/auth/login", body: body)
        guard let token = Self.sessionToken(from: cookieHeaders(response)) else {
            throw APIError(code: 500, message: "Réponse de connexion inattendue.")
        }
        try await sessions.setToken(token, for: server.origin)
        return try await me()
    }

    func linkStart(deviceName: String) async throws -> LinkStart {
        try await post("/auth/link/start", body: LinkName(name: deviceName))
    }

    func linkPoll(secret: String) async throws -> UserProfile? {
        let body = try encoder.encode(LinkPoll(poll: secret))
        let (data, response) = try await call(method: "POST", path: "/auth/link/poll", body: body)
        let status: LinkStatus = try decode(data)
        guard status.status == "ok" else { return nil }
        guard let token = Self.sessionToken(from: cookieHeaders(response)) else {
            throw APIError(code: 500, message: "Réponse d’association inattendue.")
        }
        try await sessions.setToken(token, for: server.origin)
        return try await me()
    }

    func me() async throws -> UserProfile { try await get("/auth/me") }

    func logout() async {
        _ = try? await call(method: "POST", path: "/auth/logout", body: Data("{}".utf8))
        try? await sessions.setToken(nil, for: server.origin)
    }

    func home() async throws -> Home { try await get("/home") }
    func libraries() async throws -> [LibraryRef] { try await get("/libraries") }

    func list(kind: String, libraryID: Int64?, sort: String, descending: Bool, offset: Int, limit: Int = 500, unwatched: Bool = false) async throws -> ListPage {
        var query = [
            URLQueryItem(name: "sort", value: sort),
            URLQueryItem(name: "order", value: descending ? "desc" : "asc"),
            URLQueryItem(name: "offset", value: String(offset)),
            URLQueryItem(name: "limit", value: String(limit))
        ]
        if let libraryID { query.append(URLQueryItem(name: "library", value: String(libraryID))) }
        if unwatched { query.append(URLQueryItem(name: "watched", value: "unwatched")) }
        return try await get(kind == "movie" ? "/movies" : "/series", query: query)
    }

    func detail(itemID: Int64) async throws -> Detail { try await get("/items/\(itemID)") }
    func episodes(seriesID: Int64, season: Int) async throws -> [Card] { try await get("/items/\(seriesID)/seasons/\(season)") }
    func next(episodeID: Int64) async throws -> Card? { try await get( "/items/\(episodeID)/next", as: NextResponse.self).next }

    func search(_ query: String) async throws -> SearchResults {
        try await get("/search", query: [URLQueryItem(name: "q", value: query), URLQueryItem(name: "limit", value: "30")])
    }

    func play(_ request: PlayRequest) async throws -> PlayResponse {
        var safe = request
        safe.external = false
        return try await post("/play", body: safe)
    }

    func progress(itemID: Int64, fileID: Int64, position: Double, duration: Double) async {
        _ = try? await post("/items/\(itemID)/progress", body: ProgressBody(fileId: fileID, position: position, duration: duration), as: EmptyResponse.self)
    }

    func setState(itemID: Int64, favorite: Bool?, watchlist: Bool?, watched: Bool?, season: Int?) async throws {
        _ = try await post(
            "/items/\(itemID)/state",
            body: StateBody(favorite: favorite, watchlist: watchlist, watched: watched, season: season),
            as: EmptyResponse.self
        )
    }

    func hideContinue(itemID: Int64) async throws {
        _ = try await call(method: "POST", path: "/items/\(itemID)/continue/hide", body: Data("{}".utf8))
    }

    func stopSession(id: String) async {
        _ = try? await call(method: "DELETE", path: "/hls/\(id)")
    }

    func imageData(path: String) async throws -> Data {
        guard let url = absoluteURL(path) else { throw APIError(code: 0, message: "Adresse d’image invalide.") }
        let request = try await authorizedRequest(url: url, method: "GET")
        let (data, response) = try await transport.send(request)
        try await validate(response: response, data: data)
        return data
    }

    static func sessionToken(from headers: [String]) -> String? {
        for header in headers {
            for cookie in header.split(separator: ",") {
                let pair = cookie.split(separator: ";", maxSplits: 1)[0].trimmingCharacters(in: .whitespaces)
                if pair.hasPrefix("aurora_session=") {
                    return String(pair.dropFirst("aurora_session=".count))
                }
            }
        }
        return nil
    }

    private struct EmptyResponse: Decodable {}

    private func get<T: Decodable>(_ path: String, query: [URLQueryItem] = [], as type: T.Type = T.self) async throws -> T {
        let (data, _) = try await call(method: "GET", path: path, query: query)
        return try decode(data)
    }

    private func post<T: Decodable, Body: Encodable>(_ path: String, body: Body, as type: T.Type = T.self) async throws -> T {
        let data = try encoder.encode(body)
        let (responseData, _) = try await call(method: "POST", path: path, body: data)
        if T.self == EmptyResponse.self, responseData.isEmpty || String(decoding: responseData, as: UTF8.self) == "{}" {
            return EmptyResponse() as! T
        }
        return try decode(responseData)
    }

    private func call(method: String, path: String, query: [URLQueryItem] = [], body: Data? = nil) async throws -> (Data, HTTPURLResponse) {
        var components = URLComponents(url: server.url, resolvingAgainstBaseURL: false)
        components?.path = "/api/v1" + path
        components?.queryItems = query.isEmpty ? nil : query
        guard let url = components?.url else { throw APIError(code: 0, message: "Adresse AURORA invalide.") }
        var request = try await authorizedRequest(url: url, method: method)
        if let body {
            request.httpBody = body
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        let (data, response) = try await transport.send(request)
        try await validate(response: response, data: data)
        return (data, response)
    }

    private func authorizedRequest(url: URL, method: String) async throws -> URLRequest {
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("1", forHTTPHeaderField: "X-Aurora")
        if let token = try await sessions.token(for: server.origin) {
            request.setValue("aurora_session=\(token)", forHTTPHeaderField: "Cookie")
        }
        return request
    }

    private func validate(response: HTTPURLResponse, data: Data) async throws {
        guard !(200..<300).contains(response.statusCode) else { return }
        if response.statusCode == 401 { await onUnauthorized() }
        let serverMessage = try? decoder.decode(ErrorBody.self, from: data).error
        let message = serverMessage.map(Self.uppercaseFirst) ?? "Erreur \(response.statusCode)"
        throw APIError(code: response.statusCode, message: message)
    }

    private func decode<T: Decodable>(_ data: Data) throws -> T {
        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.unexpected(server: server)
        }
    }

    private func cookieHeaders(_ response: HTTPURLResponse) -> [String] {
        response.allHeaderFields.compactMap { key, value in
            String(describing: key).caseInsensitiveCompare("Set-Cookie") == .orderedSame ? String(describing: value) : nil
        }
    }

    private static func uppercaseFirst(_ message: String) -> String {
        guard let first = message.first else { return message }
        return first.uppercased() + message.dropFirst()
    }
}

