import Foundation

protocol PlaybackSleeping: Sendable {
    func sleep(for duration: Duration) async
}

struct ContinuousPlaybackClock: PlaybackSleeping {
    func sleep(for duration: Duration) async {
        try? await Task.sleep(for: duration)
    }
}

actor PlaybackCleanup {
    private let api: any AuroraServing

    init(api: any AuroraServing) { self.api = api }

    func session(_ id: String?) async {
        guard let id else { return }
        await api.stopSession(id: id)
    }
}
