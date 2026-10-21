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
    func artworkChoices(itemID: Int64) async throws -> ArtworkChoices
    func setArtwork(itemID: Int64, type: String, path: String) async throws
    func candidates(itemID: Int64, query: String) async throws -> [TMDbCandidate]
    func match(itemID: Int64, tmdbID: Int) async throws
    func hideContinue(itemID: Int64) async throws
    func stopSession(id: String) async
    func imageData(path: String) async throws -> Data
    // Événements serveur (SSE admin) : phases d'analyse. Flux auto-reconnecté, ne jette pas d'erreur.
    func scanEvents() async -> AsyncStream<ScanUpdate>
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

    // Candidats TMDB (affiches/fonds) : `preview` est une URL publique TMDB,
    // seul le chemin `path` est renvoyé au NAS au moment d'appliquer.
    func artworkChoices(itemID: Int64) async throws -> ArtworkChoices {
        try await get("/admin/items/\(itemID)/artwork")
    }

    func setArtwork(itemID: Int64, type: String, path: String) async throws {
        struct Body: Encodable { let type: String; let path: String }
        _ = try await post("/admin/items/\(itemID)/artwork", body: Body(type: type, path: path), as: EmptyResponse.self)
    }

    // Correction d'identification : candidats TMDB puis association verrouillée.
    func candidates(itemID: Int64, query: String) async throws -> [TMDbCandidate] {
        var queryItems: [URLQueryItem] = []
        if !query.isEmpty { queryItems.append(URLQueryItem(name: "q", value: query)) } // sans q : le serveur recherche le titre de la fiche
        return try await get("/admin/items/\(itemID)/candidates", query: queryItems)
    }

    func match(itemID: Int64, tmdbID: Int) async throws {
        struct Body: Encodable { let tmdbId: Int }
        _ = try await post("/admin/items/\(itemID)/match", body: Body(tmdbId: tmdbID), as: EmptyResponse.self)
    }

    // Événements serveur (SSE, endpoint admin) : flux auto-reconnecté à l'image d'EventSource.
    func scanEvents() async -> AsyncStream<ScanUpdate> {
        AuroraAPI.eventStream(server: server, sessions: sessions, onUnauthorized: onUnauthorized)
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

    // Flux SSE : une URLSession dédiée, reconnexion périodique avec cookie courant.
    // Hors actor (la boucle ne doit pas occuper l'isolat d'AuroraAPI).
    private static func eventStream(
        server: ServerAddress,
        sessions: SessionStoring,
        onUnauthorized: @escaping @Sendable () async -> Void
    ) -> AsyncStream<ScanUpdate> {
        let http = URLSession(configuration: .ephemeral)
        return AsyncStream { continuation in
            let task = Task.detached(priority: .utility) {
                while !Task.isCancelled {
                    guard let url = URL(string: server.origin + "/api/v1/events") else { break }
                    var request = URLRequest(url: url)
                    request.httpMethod = "GET"
                    request.setValue("1", forHTTPHeaderField: "X-Aurora")
                    if let token = try? await sessions.token(for: server.origin), !token.isEmpty {
                        request.setValue("aurora_session=\(token)", forHTTPHeaderField: "Cookie")
                    }
                    do {
                        let (bytes, response) = try await http.bytes(for: request)
                        guard let http = response as? HTTPURLResponse else { throw URLError(.badServerResponse) }
                        guard (200..<300).contains(http.statusCode) else {
                            if http.statusCode == 401 { await onUnauthorized() }
                            break // 401/403 : pas de reconexion en boucle sur un refus d'accès.
                        }
                        for try await line in bytes.lines {
                            guard line.hasPrefix("data: "), let data = line.dropFirst(6).data(using: .utf8) else { continue }
                            if let envelope = try? JSONDecoder().decode(SSEEnvelope.self, from: data) {
                                continuation.yield(envelope.update)
                            }
                        }
                    } catch {
                        if Task.isCancelled { break }
                        // Connexion perdue : délai court comme un EventSource navigateur.
                        try? await Task.sleep(for: .seconds(3))
                    }
                    if Task.isCancelled { break }
                    try? await Task.sleep(for: .seconds(2))
                }
                continuation.finish()
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }
}

// Enveloppe « {"type":"scan","data":{...}} » : seuls les champs utiles au client sont décodés.
struct SSEEnvelope: Decodable, Sendable {
    let type: String
    let data: ScanUpdate

    var update: ScanUpdate { type == "scan" ? data : ScanUpdate() }
}

// Événement d'analyse : terminal quand phase == "done" ou status != "running".
struct ScanUpdate: Decodable, Sendable, Equatable {
    var phase: String?
    var status: String?

    var isTerminal: Bool {
        phase == "done" || (status != nil && status != "running")
    }
}

