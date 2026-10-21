import Foundation
import Testing
@testable import AURORA

@Suite("Sessions Trousseau")
struct KeychainSessionStoreTests {
    @Test func savesLoadsAndDeletesTokenPerOrigin() async throws {
        let service = "com.remyj.AURORA.tests.\(UUID().uuidString)"
        let store = KeychainSessionStore(service: service)
        let origin = "http://nas:8890"

        try await store.setToken("top-secret", for: origin)
        #expect(try await store.token(for: origin) == "top-secret")

        try await store.setToken(nil, for: origin)
        #expect(try await store.token(for: origin) == nil)
    }
}

