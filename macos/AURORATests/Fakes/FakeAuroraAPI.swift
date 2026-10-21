import Foundation
@testable import AURORA

enum FakeOutcome<Value: Sendable>: Sendable {
    case value(Value)
    case failure(APIError)

    func get() throws -> Value {
        switch self {
        case .value(let value): value
        case .failure(let error): throw error
        }
    }
}

struct FakePlayPlan: Sendable {
    let outcome: FakeOutcome<PlayResponse>
    let delay: Duration

    init(_ outcome: FakeOutcome<PlayResponse>, delay: Duration = .zero) {
        self.outcome = outcome
        self.delay = delay
    }
}

actor FakeAuroraAPI: AuroraServing {
    struct ProgressCall: Sendable {
        let itemID: Int64
        let fileID: Int64
        let position: Double
        let duration: Double
    }

    struct Snapshot: Sendable {
        let statusCalls: Int
        let linkStartCalls: Int
        let linkPollCalls: Int
        let loginCalls: Int
        let loginUsername: String?
        let loginPassword: String?
        let logoutCalls: Int
        let listOffsets: [Int]
        let stateCalls: Int
        let hiddenItemIDs: [Int64]
        let playRequests: [PlayRequest]
        let stoppedSessionIDs: [String]
        let progressCalls: [ProgressCall]
    }

    nonisolated let server: ServerAddress
    var statusOutcomes: [FakeOutcome<Status>]
    var linkStartOutcome: FakeOutcome<LinkStart>
    var linkPollOutcomes: [FakeOutcome<UserProfile?>]
    var loginOutcome: FakeOutcome<UserProfile>
    private var statusCalls = 0
    private var linkStartCalls = 0
    private var linkPollCalls = 0
    private var loginCalls = 0
    private var loginUsername: String?
    private var loginPassword: String?
    private var logoutCalls = 0
    private var homeOutcome: FakeOutcome<Home>?
    private var librariesOutcome: FakeOutcome<[LibraryRef]>?
    private var listOutcomes: [String: FakeOutcome<ListPage>]
    private var listOffsets: [Int] = []
    private var searchOutcomes: [String: FakeOutcome<SearchResults>]
    private let searchDelay: Duration
    private var detailOutcomes: [Int64: FakeOutcome<Detail>]
    private var episodeOutcomes: [String: FakeOutcome<[Card]>]
    private let episodeDelays: [String: Duration]
    private var setStateOutcome: FakeOutcome<Bool>
    private var hideContinueOutcome: FakeOutcome<Bool>
    private var stateCalls = 0
    private var hiddenItemIDs: [Int64] = []
    private var playPlans: [Double: FakePlayPlan]
    private var defaultPlayPlans: [FakePlayPlan]
    private var playRequests: [PlayRequest] = []
    private var stoppedSessionIDs: [String] = []
    private var progressCalls: [ProgressCall] = []
    private var nextOutcome: FakeOutcome<Card?>

    init(
        server: ServerAddress,
        status: [FakeOutcome<Status>] = [],
        linkStart: FakeOutcome<LinkStart> = .failure(APIError(code: 500, message: "Non configuré")),
        linkPoll: [FakeOutcome<UserProfile?>] = [],
        login: FakeOutcome<UserProfile> = .failure(APIError(code: 500, message: "Non configuré")),
        home: FakeOutcome<Home>? = nil,
        libraries: FakeOutcome<[LibraryRef]>? = nil,
        lists: [String: FakeOutcome<ListPage>] = [:],
        searches: [String: FakeOutcome<SearchResults>] = [:],
        searchDelay: Duration = .zero,
        details: [Int64: FakeOutcome<Detail>] = [:],
        episodes: [String: FakeOutcome<[Card]>] = [:],
        episodeDelays: [String: Duration] = [:],
        setState: FakeOutcome<Bool> = .value(true),
        hideContinue: FakeOutcome<Bool> = .value(true),
        plays: [Double: FakePlayPlan] = [:],
        defaultPlays: [FakePlayPlan] = [],
        next: FakeOutcome<Card?> = .value(nil)
    ) {
        self.server = server
        statusOutcomes = status
        linkStartOutcome = linkStart
        linkPollOutcomes = linkPoll
        loginOutcome = login
        homeOutcome = home
        librariesOutcome = libraries
        listOutcomes = lists
        searchOutcomes = searches
        self.searchDelay = searchDelay
        detailOutcomes = details
        episodeOutcomes = episodes
        self.episodeDelays = episodeDelays
        setStateOutcome = setState
        hideContinueOutcome = hideContinue
        playPlans = plays
        defaultPlayPlans = defaultPlays
        nextOutcome = next
    }

    func snapshot() -> Snapshot {
        Snapshot(statusCalls: statusCalls, linkStartCalls: linkStartCalls, linkPollCalls: linkPollCalls, loginCalls: loginCalls, loginUsername: loginUsername, loginPassword: loginPassword, logoutCalls: logoutCalls, listOffsets: listOffsets, stateCalls: stateCalls, hiddenItemIDs: hiddenItemIDs, playRequests: playRequests, stoppedSessionIDs: stoppedSessionIDs, progressCalls: progressCalls)
    }

    func status() async throws -> Status {
        statusCalls += 1
        guard !statusOutcomes.isEmpty else { throw APIError(code: 500, message: "Statut non configuré") }
        return try statusOutcomes.removeFirst().get()
    }

    func login(username: String, password: String) async throws -> UserProfile {
        loginCalls += 1
        loginUsername = username
        loginPassword = password
        return try loginOutcome.get()
    }

    func linkStart(deviceName: String) async throws -> LinkStart {
        linkStartCalls += 1
        return try linkStartOutcome.get()
    }

    func linkPoll(secret: String) async throws -> UserProfile? {
        linkPollCalls += 1
        guard !linkPollOutcomes.isEmpty else { throw APIError(code: 410, message: "Code expiré") }
        return try linkPollOutcomes.removeFirst().get()
    }

    func me() async throws -> UserProfile { try loginOutcome.get() }
    func logout() async { logoutCalls += 1 }
    func home() async throws -> Home { try homeOutcome?.get() ?? { throw unused() }() }
    func libraries() async throws -> [LibraryRef] { try librariesOutcome?.get() ?? { throw unused() }() }
    func list(kind: String, libraryID: Int64?, sort: String, descending: Bool, offset: Int, limit: Int, unwatched: Bool) async throws -> ListPage {
        listOffsets.append(offset)
        let key = "\(kind):\(libraryID ?? -1):\(offset)"
        return try listOutcomes[key]?.get() ?? { throw unused() }()
    }
    func detail(itemID: Int64) async throws -> Detail { try detailOutcomes[itemID]?.get() ?? { throw unused() }() }
    func episodes(seriesID: Int64, season: Int) async throws -> [Card] {
        let key = "\(seriesID):\(season)"
        if let delay = episodeDelays[key], delay > .zero { try await Task.sleep(for: delay) }
        return try episodeOutcomes[key]?.get() ?? { throw unused() }()
    }
    func next(episodeID: Int64) async throws -> Card? { try nextOutcome.get() }
    func search(_ query: String) async throws -> SearchResults {
        if searchDelay > .zero { try await Task.sleep(for: searchDelay) }
        return try searchOutcomes[query]?.get() ?? { throw unused() }()
    }
    func play(_ request: PlayRequest) async throws -> PlayResponse {
        playRequests.append(request)
        let plan: FakePlayPlan
        if let matched = playPlans[request.start] {
            plan = matched
        } else if !defaultPlayPlans.isEmpty {
            plan = defaultPlayPlans.removeFirst()
        } else {
            throw unused()
        }
        if plan.delay > .zero { try await Task.sleep(for: plan.delay) }
        return try plan.outcome.get()
    }
    func progress(itemID: Int64, fileID: Int64, position: Double, duration: Double) async {
        progressCalls.append(ProgressCall(itemID: itemID, fileID: fileID, position: position, duration: duration))
    }
    func setState(itemID: Int64, favorite: Bool?, watchlist: Bool?, watched: Bool?, season: Int?) async throws {
        stateCalls += 1
        _ = try setStateOutcome.get()
    }
    func hideContinue(itemID: Int64) async throws {
        _ = try hideContinueOutcome.get()
        hiddenItemIDs.append(itemID)
    }
    func stopSession(id: String) async { stoppedSessionIDs.append(id) }
    func imageData(path: String) async throws -> Data { throw unused() }
    nonisolated func absoluteURL(_ path: String?) -> URL? { path.flatMap { URL(string: server.origin + $0) } }

    private func unused() -> APIError { APIError(code: 500, message: "Méthode de test non configurée") }
}

func testUser(_ name: String = "remy") -> UserProfile {
    try! JSONDecoder().decode(UserProfile.self, from: Data(#"{"id":7,"username":"\#(name)","role":"admin"}"#.utf8))
}

func testStatus(user: UserProfile?) -> Status {
    if let user {
        let userData = try! JSONEncoder().encode(user)
        let object = try! JSONSerialization.jsonObject(with: userData)
        let data = try! JSONSerialization.data(withJSONObject: ["version": "0.1.8", "setupDone": true, "user": object])
        return try! JSONDecoder().decode(Status.self, from: data)
    }
    return try! JSONDecoder().decode(Status.self, from: Data(#"{"version":"0.1.8","setupDone":true}"#.utf8))
}

