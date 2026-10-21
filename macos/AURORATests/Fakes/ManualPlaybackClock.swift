import Foundation
@testable import AURORA

actor ManualPlaybackClock: PlaybackSleeping {
    private struct Waiter {
        let duration: Duration
        let continuation: CheckedContinuation<Void, Never>
    }
    private var waiters: [Waiter] = []

    func sleep(for duration: Duration) async {
        await withCheckedContinuation { continuation in
            waiters.append(Waiter(duration: duration, continuation: continuation))
        }
    }

    func advance(_ duration: Duration) {
        let ready = waiters.filter { $0.duration <= duration }
        waiters.removeAll { $0.duration <= duration }
        ready.forEach { $0.continuation.resume() }
    }

    var pending: [Duration] { waiters.map(\.duration) }
}
