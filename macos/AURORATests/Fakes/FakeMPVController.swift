import AppKit
import Foundation
@testable import AURORA

@MainActor
final class FakeMPVController: MPVControlling {
    struct Load: Equatable {
        let url: URL
        let headers: [String: String]
        let start: Double
    }
    private let continuation: AsyncStream<MPVEvent>.Continuation
    let events: AsyncStream<MPVEvent>
    private(set) var commands: [String] = []
    private(set) var loads: [Load] = []

    init() {
        (events, continuation) = AsyncStream.makeStream(of: MPVEvent.self)
    }

    func attach(to view: NSView) { commands.append("attach") }
    func load(url: URL, headers: [String: String], start: Double) {
        loads.append(Load(url: url, headers: headers, start: start))
        commands.append("load:\(url.absoluteString):\(start)")
    }
    func setPaused(_ paused: Bool) { commands.append("pause:\(paused)") }
    func seek(to seconds: Double) { commands.append("seek:\(seconds)") }
    func selectAudio(id: Int) { commands.append("audio:\(id)") }
    func selectSubtitle(id: Int?) { commands.append("subtitle:\(id.map(String.init) ?? "no")") }
    func addSubtitle(url: URL, select: Bool) { commands.append("sub-add:\(url.absoluteString):\(select ? "cached" : "auto")") }
    func setSubtitleScale(_ scale: Double) { commands.append("sub-scale:\(scale)") }
    func setRate(_ rate: Double) { commands.append("rate:\(rate)") }
    func stop() { commands.append("stop") }
    func emit(_ event: MPVEvent) { continuation.yield(event) }
}
