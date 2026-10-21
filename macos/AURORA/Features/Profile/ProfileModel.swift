import Observation

@MainActor
@Observable
final class ProfileModel {
    let username: String
    let serverAddress: String
    let playbackSummary: String
    private let api: any AuroraServing
    private(set) var serverVersion = "—"
    private(set) var errorMessage: String?

    init(user: UserProfile, server: ServerAddress, api: any AuroraServing, capabilities: Capabilities) {
        username = user.username
        serverAddress = server.origin
        self.api = api
        playbackSummary = "\(capabilities.video.map { $0.uppercased() }.joined(separator: ", ")) · audio \(capabilities.audio.map { $0.uppercased() }.joined(separator: ", ")) · jusqu’à \(capabilities.maxHeight)p"
    }

    func load() async {
        do {
            serverVersion = try await api.status().version
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
