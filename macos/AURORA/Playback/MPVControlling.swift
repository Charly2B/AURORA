import AppKit
import Foundation

@MainActor
protocol MPVControlling: AnyObject {
    var events: AsyncStream<MPVEvent> { get }
    func attach(to view: NSView)
    func load(url: URL, headers: [String: String], start: Double)
    func setPaused(_ paused: Bool)
    func seek(to seconds: Double)
    func selectAudio(id: Int)
    func selectSubtitle(id: Int?)
    func addSubtitle(url: URL, select: Bool)
    func setSubtitleScale(_ scale: Double)
    func setRate(_ rate: Double)
    func stop()
}
