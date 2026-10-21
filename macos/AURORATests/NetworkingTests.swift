import Foundation
import Testing
@testable import AURORA

@Suite("Client réseau AURORA", .serialized)
struct NetworkingTests {
    private let server = try! ServerAddress.normalize("nas")

    @Test func sendsCookieAndCSRFGateOnAuthenticatedRequests() async throws {
        let transport = RecordingTransport([.success((Data(#"{"version":"0.1.8"}"#.utf8), response(200)))])
        let sessions = MemorySessionStore(origin: server.origin, token: "secret-token")
        let api = AuroraAPI(server: server, transport: transport, sessions: sessions)

        _ = try await api.status()
        let recordedRequests = await transport.requests
        let request = try #require(recordedRequests.first)

        #expect(request.url?.absoluteString == "http://nas:8890/api/v1/status")
        #expect(request.httpMethod == "GET")
        #expect(request.value(forHTTPHeaderField: "Cookie") == "aurora_session=secret-token")
        #expect(request.value(forHTTPHeaderField: "X-Aurora") == "1")
    }

    @Test func percentEncodesSearchWithoutPuttingSecretsInURL() async throws {
        let transport = RecordingTransport([.success((Data(#"{"movies":[]}"#.utf8), response(200)))])
        let api = AuroraAPI(server: server, transport: transport, sessions: MemorySessionStore())

        _ = try await api.search("été & mer")
        let recordedRequests = await transport.requests
        let request = try #require(recordedRequests.first)
        let requestURL = try #require(request.url)
        let components = try #require(URLComponents(url: requestURL, resolvingAgainstBaseURL: false))

        #expect(components.path == "/api/v1/search")
        #expect(components.queryItems?.first(where: { $0.name == "q" })?.value == "été & mer")
        #expect(components.queryItems?.first(where: { $0.name == "limit" })?.value == "30")
        #expect(request.url?.absoluteString.contains("secret-token") == false)
    }

    @Test func encodesStateMutationAsJSON() async throws {
        let transport = RecordingTransport([.success((Data("{}".utf8), response(200)))])
        let api = AuroraAPI(server: server, transport: transport, sessions: MemorySessionStore())

        try await api.setState(itemID: 42, favorite: true, watchlist: nil, watched: false, season: 2)
        let recordedRequests = await transport.requests
        let request = try #require(recordedRequests.first)
        let body = try #require(request.httpBody)
        let object = try #require(JSONSerialization.jsonObject(with: body) as? [String: Any])

        #expect(request.httpMethod == "POST")
        #expect(object["favorite"] as? Bool == true)
        #expect(object["watched"] as? Bool == false)
        #expect(object["season"] as? Int == 2)
        #expect(object["watchlist"] == nil)
    }

    @Test func extractsOnlyAuroraCookieFromMultipleHeaders() {
        let token = AuroraAPI.sessionToken(from: [
            "theme=dark; Path=/",
            "aurora_session=abc123; HttpOnly; SameSite=Lax",
            "other=value"
        ])
        #expect(token == "abc123")
    }

    @Test func loginStoresCookieWithoutReturningTheSecret() async throws {
        let headers = ["Set-Cookie": "aurora_session=sensitive; Path=/; HttpOnly"]
        let profile = #"{"id":7,"username":"remy","role":"admin"}"#
        let transport = RecordingTransport([
            .success((Data("{}".utf8), response(200, headers: headers))),
            .success((Data(profile.utf8), response(200)))
        ])
        let sessions = MemorySessionStore()
        let api = AuroraAPI(server: server, transport: transport, sessions: sessions)

        let user = try await api.login(username: "remy", password: "mot-de-passe")

        #expect(user.username == "remy")
        #expect(try await sessions.token(for: server.origin) == "sensitive")
    }

    @Test func unauthorizedInvokesCallbackAndReturnsFrenchError() async {
        let flag = UnauthorizedFlag()
        let transport = RecordingTransport([.success((Data(#"{"error":"session expirée"}"#.utf8), response(401)))])
        let api = AuroraAPI(server: server, transport: transport, sessions: MemorySessionStore()) {
            await flag.mark()
        }

        do {
            _ = try await api.home()
            Issue.record("La requête 401 devait échouer")
        } catch let error as APIError {
            #expect(error.code == 401)
            #expect(error.localizedDescription == "Session expirée")
        } catch {
            Issue.record("Type d’erreur inattendu : \(error)")
        }
        #expect(await flag.value)
    }

    @Test func malformedHTMLBodyBecomesDisplayableError() async {
        let transport = RecordingTransport([.success((Data("<html>proxy</html>".utf8), response(200)))])
        let api = AuroraAPI(server: server, transport: transport, sessions: MemorySessionStore())

        do {
            _ = try await api.status()
            Issue.record("Le HTML ne doit pas être décodé comme un statut")
        } catch let error as APIError {
            #expect(error.localizedDescription.contains("est-ce bien AURORA"))
        } catch {
            Issue.record("Type d’erreur inattendu : \(error)")
        }
    }

    @Test(arguments: [
        (URLError.Code.cannotConnectToHost, "ne répond pas"),
        (URLError.Code.networkConnectionLost, "interrompue")
    ])
    func transportMapsNetworkFailures(code: URLError.Code, message: String) async {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [MockURLProtocol.self]
        MockURLProtocol.handler = { _ in throw URLError(code) }
        let transport = HTTPTransport(session: URLSession(configuration: configuration))

        do {
            _ = try await transport.send(URLRequest(url: server.url))
            Issue.record("Le transport devait échouer")
        } catch let error as APIError {
            #expect(error.localizedDescription.contains(message))
        } catch {
            Issue.record("Type d’erreur inattendu : \(error)")
        }
    }
}

private actor UnauthorizedFlag {
    private(set) var value = false
    func mark() { value = true }
}

