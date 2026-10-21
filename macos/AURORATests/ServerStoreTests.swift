import Foundation
import Testing
@testable import AURORA

@Suite("Liste des NAS", .serialized)
@MainActor
struct ServerStoreTests {
    @Test func preservesOrderDeduplicatesAndRemembersSelection() async throws {
        let context = StoreTestContext()
        let firstStore = ServerStore(defaults: context.defaults, sessions: context.sessions)

        let first = try firstStore.add("nas-one")
        let second = try firstStore.add("nas-two")
        let duplicate = try firstStore.add("http://nas-one:8890/")
        firstStore.select(second)

        #expect(first == duplicate)
        #expect(firstStore.allServers() == [first, second])

        let restored = ServerStore(defaults: context.defaults, sessions: context.sessions)
        #expect(restored.allServers() == [first, second])
        #expect(restored.selectedServer() == second)
    }

    @Test func deletingServerAlsoDeletesOnlyItsSession() async throws {
        let context = StoreTestContext()
        let store = ServerStore(defaults: context.defaults, sessions: context.sessions)
        let first = try store.add("nas-one")
        let second = try store.add("nas-two")
        try await context.sessions.setToken("one", for: first.origin)
        try await context.sessions.setToken("two", for: second.origin)

        try await store.remove(first)

        #expect(try await context.sessions.token(for: first.origin) == nil)
        #expect(try await context.sessions.token(for: second.origin) == "two")
        #expect(store.allServers() == [second])
    }
}

@MainActor
private final class StoreTestContext {
    let suite = "AURORA-ServerStoreTests-\(UUID().uuidString)"
    let defaults: UserDefaults
    let sessions = MemorySessionStore()

    init() {
        defaults = UserDefaults(suiteName: suite)!
        defaults.removePersistentDomain(forName: suite)
    }
}

