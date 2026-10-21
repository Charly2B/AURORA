import Foundation
@testable import AURORA

final class MockURLProtocol: URLProtocol, @unchecked Sendable {
    nonisolated(unsafe) static var handler: ((URLRequest) throws -> (HTTPURLResponse, Data))?

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        do {
            guard let handler = Self.handler else { throw URLError(.badServerResponse) }
            let (response, data) = try handler(request)
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}
}

actor RecordingTransport: HTTPTransporting {
    private var outcomes: [Result<(Data, HTTPURLResponse), Error>]
    private(set) var requests: [URLRequest] = []

    init(_ outcomes: [Result<(Data, HTTPURLResponse), Error>]) {
        self.outcomes = outcomes
    }

    func send(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) {
        requests.append(request)
        guard !outcomes.isEmpty else { throw URLError(.badServerResponse) }
        return try outcomes.removeFirst().get()
    }
}

actor MemorySessionStore: SessionStoring {
    private var values: [String: String] = [:]

    init(origin: String? = nil, token: String? = nil) {
        if let origin, let token { values[origin] = token }
    }

    func token(for origin: String) async throws -> String? { values[origin] }

    func setToken(_ token: String?, for origin: String) async throws {
        values[origin] = token
    }
}

func response(_ code: Int, headers: [String: String] = [:], url: URL = URL(string: "http://nas:8890/api/v1/status")!) -> HTTPURLResponse {
    HTTPURLResponse(url: url, statusCode: code, httpVersion: "HTTP/1.1", headerFields: headers)!
}

