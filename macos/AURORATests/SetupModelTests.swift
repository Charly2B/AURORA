import Foundation
import Testing
@testable import AURORA

@Suite("Connexion multi-NAS", .serialized)
@MainActor
struct SetupModelTests {
    @Test func restoreAuthenticatesRememberedServerAutomatically() async throws {
        let context = try await SetupContext.make(status: [.value(testStatus(user: testUser()))])

        await context.session.restore()

        #expect(context.session.phase == .authenticated)
        #expect(context.session.profile?.username == "remy")
        #expect(context.session.selectedServer == context.server)
    }

    @Test func linkPollsEveryTwoSecondsAndAuthenticatesWithoutNewCode() async throws {
        let context = try await SetupContext.make(
            linkStart: .value(try link(code: "ABCDEF", poll: "poll-secret")),
            linkPoll: [.value(nil), .value(testUser())]
        )
        let model = SetupModel(session: context.session, sleeper: context.clock)

        try await model.beginLink(deviceName: "iMac Salon")
        await model.pollLink()

        let snapshot = await context.api.snapshot()
        #expect(snapshot.linkStartCalls == 1)
        #expect(snapshot.linkPollCalls == 2)
        #expect(await context.clock.sleeps == [.seconds(2)])
        #expect(model.link?.code == "ABCDEF")
        #expect(context.session.phase == .authenticated)
    }

    @Test func transientPollFailureKeepsCodeAndRetriesSameSecret() async throws {
        let context = try await SetupContext.make(
            linkStart: .value(try link(code: "STABLE", poll: "same-secret")),
            linkPoll: [.failure(APIError(code: 0, message: "Réseau")), .value(testUser())]
        )
        let model = SetupModel(session: context.session, sleeper: context.clock)

        try await model.beginLink(deviceName: "iMac")
        await model.pollLink()

        let snapshot = await context.api.snapshot()
        #expect(snapshot.linkStartCalls == 1)
        #expect(snapshot.linkPollCalls == 2)
        #expect(model.link?.code == "STABLE")
        #expect(context.session.phase == .authenticated)
    }

    @Test func expiredCodeStopsWithoutSilentlyCreatingAnother() async throws {
        let context = try await SetupContext.make(
            linkStart: .value(try link(code: "EXPIRE", poll: "expired-secret")),
            linkPoll: [.failure(APIError(code: 410, message: "Code expiré"))]
        )
        let model = SetupModel(session: context.session, sleeper: context.clock)

        try await model.beginLink(deviceName: "iMac")
        await model.pollLink()

        let snapshot = await context.api.snapshot()
        #expect(snapshot.linkStartCalls == 1)
        #expect(model.state == .expired)
        #expect(model.link?.code == "EXPIRE")
    }

    @Test func passwordFallbackAuthenticatesAndNeverPersistsPasswordInModel() async throws {
        let context = try await SetupContext.make(login: .value(testUser("alice")))
        let model = SetupModel(session: context.session, sleeper: context.clock)

        await model.login(username: "  alice  ", password: "secret")

        let snapshot = await context.api.snapshot()
        #expect(snapshot.loginUsername == "alice")
        #expect(snapshot.loginPassword == "secret")
        #expect(context.session.profile?.username == "alice")
        #expect(model.password.isEmpty)
    }

    @Test func unauthorizedAffectsOnlyCurrentlySelectedServer() async throws {
        let context = try await SetupContext.make(status: [.value(testStatus(user: testUser()))])
        await context.session.restore()
        let other = try ServerAddress.normalize("other-nas")

        await context.session.unauthorized(origin: other.origin)
        #expect(context.session.phase == .authenticated)

        await context.session.unauthorized(origin: context.server.origin)
        #expect(context.session.phase == .login)
        #expect(context.session.profile == nil)
    }

    private func link(code: String, poll: String) throws -> LinkStart {
        try JSONDecoder().decode(LinkStart.self, from: Data(#"{"code":"\#(code)","poll":"\#(poll)","expiresIn":600}"#.utf8))
    }
}

@MainActor
private struct SetupContext {
    let suite: String
    let defaults: UserDefaults
    let sessions: MemorySessionStore
    let store: ServerStore
    let server: ServerAddress
    let api: FakeAuroraAPI
    let clock: ImmediateClock
    let session: AppSession

    static func make(
        status: [FakeOutcome<Status>] = [],
        linkStart: FakeOutcome<LinkStart> = .failure(APIError(code: 500, message: "Non configuré")),
        linkPoll: [FakeOutcome<UserProfile?>] = [],
        login: FakeOutcome<UserProfile> = .failure(APIError(code: 500, message: "Non configuré"))
    ) async throws -> SetupContext {
        let suite = "AURORA-SetupTests-\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: suite)!
        defaults.removePersistentDomain(forName: suite)
        let sessions = MemorySessionStore()
        let store = ServerStore(defaults: defaults, sessions: sessions)
        let server = try store.add("nas")
        store.select(server)
        let api = FakeAuroraAPI(server: server, status: status, linkStart: linkStart, linkPoll: linkPoll, login: login)
        let clock = ImmediateClock()
        let session = AppSession(store: store) { _, _ in api }
        return SetupContext(suite: suite, defaults: defaults, sessions: sessions, store: store, server: server, api: api, clock: clock, session: session)
    }
}

