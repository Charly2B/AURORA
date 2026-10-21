import Foundation
import Testing
@testable import AURORA

@Suite("Intégration application", .serialized)
@MainActor
struct AppSessionIntegrationTests {
    @Test func authenticatedLaunchOpensHome() async throws {
        let context = try IntegrationContext()
        await context.session.restore()

        #expect(context.session.phase == .authenticated)
        #expect(AppRoute.initial(for: context.session) == .home)
    }

    @Test func changingNASReusesItsValidKeychainSessionAndClearsNavigationIdentity() async throws {
        let context = try IntegrationContext()
        await context.session.restore()
        let previousIdentity = context.session.shellIdentity

        await context.session.select(context.second)

        #expect(context.session.selectedServer == context.second)
        #expect(context.session.phase == .authenticated)
        #expect(context.session.profile?.username == "second")
        #expect(context.session.shellIdentity != previousIdentity)
    }

    @Test func logoutDeletesOnlyCurrentServerToken() async throws {
        let context = try IntegrationContext()
        try await context.sessions.setToken("one", for: context.first.origin)
        try await context.sessions.setToken("two", for: context.second.origin)
        await context.session.restore()

        await context.session.logout()

        #expect(try await context.sessions.token(for: context.first.origin) == nil)
        #expect(try await context.sessions.token(for: context.second.origin) == "two")
        #expect(context.session.phase == .login)
    }

    @Test func adjacentEpisodesHaveDistinctViewIdentity() {
        let first = AppRoute.player(PlayIntent(itemID: 10, fileID: nil, startPosition: 0, restart: false), title: "E1")
        let second = AppRoute.player(PlayIntent(itemID: 11, fileID: nil, startPosition: 0, restart: false), title: "E2")
        #expect(first.identity != second.identity)
        #expect(AppRoute.detail(itemID: 10).identity != AppRoute.detail(itemID: 11).identity)
    }
}

@MainActor
private struct IntegrationContext {
    let suite: String
    let defaults: UserDefaults
    let sessions: MemorySessionStore
    let first: ServerAddress
    let second: ServerAddress
    let session: AppSession

    init() throws {
        suite = "AURORA-Integration-\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suite)!
        defaults.removePersistentDomain(forName: suite)
        sessions = MemorySessionStore()
        let store = ServerStore(defaults: defaults, sessions: sessions)
        let firstServer = try store.add("nas-one")
        let secondServer = try store.add("nas-two")
        store.select(firstServer)
        let user = testUser()
        let firstAPI = FakeAuroraAPI(server: firstServer, status: [.value(testStatus(user: user))])
        let secondAPI = FakeAuroraAPI(server: secondServer, status: [.value(testStatus(user: testUser("second")))])
        first = firstServer
        second = secondServer
        session = AppSession(store: store) { server, _ in server == firstServer ? firstAPI : secondAPI }
    }
}
