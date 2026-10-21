import AppKit

enum PlayerKeyCommand: Equatable {
    case togglePause
    case seek(Double)
    case fullscreen
    case closePanelOrPlayer
    case cycleSubtitles
    case cycleAudio

    @MainActor
    static func map(_ event: NSEvent) -> PlayerKeyCommand? {
        map(event, firstResponder: event.window?.firstResponder)
    }

    @MainActor
    static func map(_ event: NSEvent, firstResponder: NSResponder?) -> PlayerKeyCommand? {
        if firstResponder is NSTextView || firstResponder is NSTextField { return nil }
        if event.type == .systemDefined, event.subtype.rawValue == 8 {
            let mediaKey = (event.data1 & 0xFFFF0000) >> 16
            let state = (event.data1 & 0x0000FF00) >> 8
            guard state == 0x0A else { return nil }
            switch mediaKey {
            case 16: return .togglePause
            case 19: return .seek(30)
            case 20: return .seek(-10)
            default: return nil
            }
        }
        guard event.type == .keyDown else { return nil }
        switch event.keyCode {
        case 49: return .togglePause
        case 123: return .seek(-10)
        case 124: return .seek(30)
        case 53: return .closePanelOrPlayer
        default:
            switch event.charactersIgnoringModifiers?.lowercased() {
            case "f": return .fullscreen
            case "c": return .cycleSubtitles
            case "a": return .cycleAudio
            default: return nil
            }
        }
    }
}

enum PlayerControlsVisibility {
    static func shouldAutoHide(playback: PlaybackState, menuOpen: Bool) -> Bool {
        playback == .playing && !menuOpen
    }
}
