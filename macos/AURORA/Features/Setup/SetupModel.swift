import Foundation
import Observation

protocol SetupSleeping: Sendable {
    func sleep(for duration: Duration) async throws
}

struct ContinuousSetupSleeper: SetupSleeping {
    func sleep(for duration: Duration) async throws {
        try await Task.sleep(for: duration)
    }
}

enum SetupState: Equatable {
    case idle
    case requestingCode
    case waiting
    case expired
    case authenticated
}

@MainActor
@Observable
final class SetupModel {
    private let session: AppSession
    private let sleeper: any SetupSleeping
    private var pollGeneration = 0

    private(set) var link: LinkStart?
    private(set) var state: SetupState = .idle
    var serverAddress = ""
    var username = ""
    var password = ""
    var errorMessage: String?
    private(set) var isAuthenticating = false

    init(session: AppSession, sleeper: any SetupSleeping = ContinuousSetupSleeper()) {
        self.session = session
        self.sleeper = sleeper
    }

    @discardableResult
    func addServer() async -> ServerAddress? {
        do {
            errorMessage = nil
            return try await session.addServer(serverAddress)
        } catch {
            errorMessage = error.localizedDescription
            return nil
        }
    }

    func beginLink(deviceName: String = Host.current().localizedName ?? "Mac") async throws {
        guard link == nil || state == .expired else { return }
        guard let api = session.service() else {
            throw APIError(code: 0, message: "Sélectionnez un NAS.")
        }
        state = .requestingCode
        errorMessage = nil
        do {
            link = try await api.linkStart(deviceName: deviceName)
            state = .waiting
        } catch {
            state = .idle
            errorMessage = error.localizedDescription
            throw error
        }
    }

    func pollLink() async {
        guard let secret = link?.poll, let api = session.service() else { return }
        pollGeneration += 1
        let generation = pollGeneration
        state = .waiting

        while generation == pollGeneration, !Task.isCancelled {
            do {
                if let user = try await api.linkPoll(secret: secret) {
                    state = .authenticated
                    session.authenticated(user)
                    return
                }
                errorMessage = nil
            } catch let error as APIError where error.code == 410 {
                errorMessage = error.message
                state = .expired
                return
            } catch let error as APIError where error.code == 0 {
                errorMessage = error.message
            } catch {
                errorMessage = error.localizedDescription
                return
            }

            do {
                try await sleeper.sleep(for: .seconds(2))
            } catch {
                return
            }
        }
    }

    func restartLink(deviceName: String = Host.current().localizedName ?? "Mac") async {
        cancelPolling()
        link = nil
        state = .idle
        try? await beginLink(deviceName: deviceName)
    }

    func login(username: String, password: String) async {
        guard !isAuthenticating else { return }
        let username = username.trimmingCharacters(in: .whitespacesAndNewlines)
        self.password = ""
        guard !username.isEmpty, !password.isEmpty else {
            errorMessage = "Saisissez votre identifiant et votre mot de passe AURORA."
            return
        }
        guard let api = session.service() else {
            errorMessage = "Sélectionnez un NAS."
            return
        }
        isAuthenticating = true
        defer { isAuthenticating = false }
        do {
            errorMessage = nil
            let user = try await api.login(username: username, password: password)
            state = .authenticated
            session.authenticated(user)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func cancelPolling() {
        pollGeneration += 1
    }
}
