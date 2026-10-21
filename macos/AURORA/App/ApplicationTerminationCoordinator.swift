import AppKit

@MainActor
final class ApplicationTerminationCoordinator {
    static let shared = ApplicationTerminationCoordinator()

    private weak var playback: PlaybackCoordinator?
    private var isTerminating = false
    private var didReply = false
    private var stopTask: Task<Void, Never>?
    private var timeoutTask: Task<Void, Never>?

    private init() {}

    func register(_ playback: PlaybackCoordinator) { self.playback = playback }

    func unregister(_ playback: PlaybackCoordinator) {
        if self.playback === playback { self.playback = nil }
    }

    func applicationShouldTerminate(_ application: NSApplication) -> NSApplication.TerminateReply {
        guard let playback else { return .terminateNow }
        guard !isTerminating else { return .terminateLater }
        isTerminating = true

        stopTask = Task { [weak self, weak application] in
            await playback.stop()
            self?.finish(application)
        }
        timeoutTask = Task { [weak self, weak application] in
            try? await Task.sleep(for: .seconds(2))
            guard !Task.isCancelled else { return }
            self?.stopTask?.cancel()
            self?.finish(application)
        }
        return .terminateLater
    }

    private func finish(_ application: NSApplication?) {
        guard !didReply else { return }
        didReply = true
        timeoutTask?.cancel()
        application?.reply(toApplicationShouldTerminate: true)
    }
}

@MainActor
final class AURORAAppDelegate: NSObject, NSApplicationDelegate {
    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        ApplicationTerminationCoordinator.shared.applicationShouldTerminate(sender)
    }
}
