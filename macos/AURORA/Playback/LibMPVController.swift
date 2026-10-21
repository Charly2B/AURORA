import AppKit
import Foundation
import Libmpv
import QuartzCore

@MainActor
final class LibMPVController: MPVControlling {
    private var handle: OpaquePointer?
    private var eventTimer: Timer?
    private let continuation: AsyncStream<MPVEvent>.Continuation
    let events: AsyncStream<MPVEvent>

    init() {
        (events, continuation) = AsyncStream.makeStream(of: MPVEvent.self, bufferingPolicy: .bufferingNewest(64))
    }

    isolated deinit {
        eventTimer?.invalidate()
        if let handle { mpv_terminate_destroy(handle) }
        continuation.finish()
    }

    func attach(to view: NSView) {
        guard handle == nil else { return }
        guard let context = mpv_create() else {
            continuation.yield(.failed(MPVFailure(code: -1, message: "Impossible d’initialiser libmpv.")))
            return
        }

        handle = context
        setOption("terminal", "no")
        setOption("msg-level", "all=no")
        setOption("input-media-keys", "yes")
        setOption("subs-match-os-language", "yes")
        setOption("subs-fallback", "yes")
        setOption("hwdec", "videotoolbox-copy,auto-safe")
        setOption("vo", "gpu-next")
        setOption("gpu-api", "vulkan")
        setOption("gpu-context", "moltenvk")
        setOption("ytdl", "no")

        view.wantsLayer = true
        if !(view.layer is CAMetalLayer) { view.layer = CAMetalLayer() }
        guard let layer = view.layer else {
            continuation.yield(.failed(MPVFailure(code: -1, message: "Surface vidéo indisponible.")))
            return
        }
        layer.frame = view.bounds
        layer.contentsScale = view.window?.backingScaleFactor ?? NSScreen.main?.backingScaleFactor ?? 2
        var windowID = Int64(Int(bitPattern: Unmanaged.passUnretained(layer).toOpaque()))
        let widResult = mpv_set_option(context, "wid", MPV_FORMAT_INT64, &windowID)
        guard widResult >= 0, mpv_initialize(context) >= 0 else {
            continuation.yield(.failed(failure(widResult < 0 ? widResult : -1)))
            mpv_terminate_destroy(context)
            handle = nil
            return
        }

        observe("time-pos", format: MPV_FORMAT_DOUBLE)
        observe("duration", format: MPV_FORMAT_DOUBLE)
        observe("pause", format: MPV_FORMAT_FLAG)
        observe("paused-for-cache", format: MPV_FORMAT_FLAG)
        observe("track-list", format: MPV_FORMAT_NODE)
        eventTimer = Timer.scheduledTimer(withTimeInterval: 1.0 / 30.0, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated { self?.drainEvents() }
        }
    }

    func load(url: URL, headers: [String: String], start: Double) {
        guard handle != nil else {
            continuation.yield(.failed(MPVFailure(code: -1, message: "Le lecteur vidéo n’est pas prêt.")))
            return
        }
        let request = MPVLoadRequest(url: url, headers: headers, start: start)
        setPropertyString("http-header-fields", request.headerFieldOption)
        for command in request.commands { send(command.arguments) }
    }

    func setPaused(_ paused: Bool) {
        guard let handle else { return }
        var value: Int32 = paused ? 1 : 0
        report(mpv_set_property(handle, "pause", MPV_FORMAT_FLAG, &value))
    }

    func seek(to seconds: Double) { send(["seek", String(seconds), "absolute+exact"]) }
    func selectAudio(id: Int) { send(MPVCommand.selectAudio(id: id).arguments) }
    func selectSubtitle(id: Int?) { send(MPVCommand.selectSubtitle(id: id).arguments) }
    func addSubtitle(url: URL, select: Bool) {
        send(["sub-add", url.absoluteString, select ? "cached" : "auto"])
    }
    func setSubtitleScale(_ scale: Double) { setPropertyString("sub-scale", String(scale)) }
    func setRate(_ rate: Double) { setPropertyString("speed", String(rate)) }

    func stop() {
        guard let handle else { return }
        send(["stop"])
        drainEvents()
        eventTimer?.invalidate()
        eventTimer = nil
        mpv_terminate_destroy(handle)
        self.handle = nil
        continuation.finish()
    }

    private func setOption(_ name: String, _ value: String) {
        guard let handle else { return }
        report(mpv_set_option_string(handle, name, value))
    }

    private func setPropertyString(_ name: String, _ value: String) {
        guard let handle else { return }
        report(mpv_set_property_string(handle, name, value))
    }

    private func observe(_ name: String, format: mpv_format) {
        guard let handle else { return }
        report(mpv_observe_property(handle, 0, name, format))
    }

    private func send(_ arguments: [String]) {
        guard let handle else { return }
        var cArguments: [UnsafePointer<CChar>?] = arguments.map { UnsafePointer(strdup($0)) }
        cArguments.append(nil)
        defer { cArguments.dropLast().forEach { free(UnsafeMutablePointer(mutating: $0)) } }
        report(mpv_command(handle, &cArguments))
    }

    private func report(_ status: Int32) {
        guard status < 0 else { return }
        continuation.yield(.failed(failure(status)))
    }

    private func failure(_ code: Int32) -> MPVFailure {
        let message = mpv_error_string(code).map(String.init(cString:)) ?? "Erreur libmpv \(code)"
        return MPVFailure(code: code, message: message)
    }

    private func drainEvents() {
        guard let handle else { return }
        while let pointer = mpv_wait_event(handle, 0), pointer.pointee.event_id != MPV_EVENT_NONE {
            let event = pointer.pointee
            if event.error < 0 { continuation.yield(.failed(failure(event.error))) }
            switch event.event_id {
            case MPV_EVENT_FILE_LOADED:
                continuation.yield(.ready)
                continuation.yield(.tracks(readTracks()))
            case MPV_EVENT_PROPERTY_CHANGE:
                handleProperty(event.data)
            case MPV_EVENT_END_FILE:
                handleEnd(event.data)
            case MPV_EVENT_SHUTDOWN:
                eventTimer?.invalidate()
                eventTimer = nil
            default:
                break
            }
        }
    }

    private func handleProperty(_ raw: UnsafeMutableRawPointer?) {
        guard let raw else { return }
        let property = raw.assumingMemoryBound(to: mpv_event_property.self).pointee
        guard let namePointer = property.name else { return }
        let name = String(cString: namePointer)
        switch name {
        case "time-pos":
            if let data = property.data { continuation.yield(.time(data.assumingMemoryBound(to: Double.self).pointee)) }
        case "duration":
            if let data = property.data { continuation.yield(.duration(data.assumingMemoryBound(to: Double.self).pointee)) }
        case "pause":
            if let data = property.data { continuation.yield(.paused(data.assumingMemoryBound(to: Int32.self).pointee != 0)) }
        case "paused-for-cache":
            if let data = property.data { continuation.yield(.buffering(data.assumingMemoryBound(to: Int32.self).pointee != 0)) }
        case "track-list":
            continuation.yield(.tracks(readTracks()))
        default:
            break
        }
    }

    private func handleEnd(_ raw: UnsafeMutableRawPointer?) {
        guard let raw else { continuation.yield(.ended); return }
        let end = raw.assumingMemoryBound(to: mpv_event_end_file.self).pointee
        if end.reason == MPV_END_FILE_REASON_ERROR {
            continuation.yield(.failed(failure(end.error)))
        } else {
            continuation.yield(.ended)
        }
    }

    private func readTracks() -> [MPVTrack] {
        guard let count = intProperty("track-list/count") else { return [] }
        return (0..<count).compactMap { index in
            guard let id = intProperty("track-list/\(index)/id"),
                  let type = stringProperty("track-list/\(index)/type"),
                  let kind = MPVTrack.Kind(rawValue: type == "sub" ? "subtitle" : type) else { return nil }
            return MPVTrack(
                id: id,
                kind: kind,
                title: stringProperty("track-list/\(index)/title") ?? "",
                language: stringProperty("track-list/\(index)/lang") ?? "",
                codec: stringProperty("track-list/\(index)/codec") ?? "",
                selected: flagProperty("track-list/\(index)/selected"),
                forced: flagProperty("track-list/\(index)/forced"),
                isDefault: flagProperty("track-list/\(index)/default")
            )
        }
    }

    private func intProperty(_ name: String) -> Int? {
        guard let handle else { return nil }
        var value: Int64 = 0
        return mpv_get_property(handle, name, MPV_FORMAT_INT64, &value) >= 0 ? Int(value) : nil
    }

    private func flagProperty(_ name: String) -> Bool {
        guard let handle else { return false }
        var value: Int32 = 0
        return mpv_get_property(handle, name, MPV_FORMAT_FLAG, &value) >= 0 && value != 0
    }

    private func stringProperty(_ name: String) -> String? {
        guard let handle, let value = mpv_get_property_string(handle, name) else { return nil }
        defer { mpv_free(value) }
        return String(cString: value)
    }
}
