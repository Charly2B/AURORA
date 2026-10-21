import Foundation
@testable import AURORA

actor ImmediateClock: SetupSleeping {
    private(set) var sleeps: [Duration] = []

    func sleep(for duration: Duration) async throws {
        sleeps.append(duration)
        try Task.checkCancellation()
    }
}

