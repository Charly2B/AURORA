import Foundation

struct MPVTrack: Sendable, Identifiable, Equatable {
    enum Kind: String, Sendable { case video, audio, subtitle }

    let id: Int
    let kind: Kind
    let title: String
    let language: String
    let codec: String
    let selected: Bool
    let forced: Bool
    let isDefault: Bool
}

struct MPVFailure: Error, Sendable, Equatable {
    let code: Int32
    let message: String
}

enum MPVEvent: Sendable, Equatable {
    case ready
    case time(Double)
    case duration(Double)
    case paused(Bool)
    case buffering(Bool)
    case tracks([MPVTrack])
    case ended
    case failed(MPVFailure)
}

enum MPVCommand: Equatable, Sendable {
    case setProperty(String, String)
    case loadFile(String)
    case selectAudio(id: Int)
    case selectSubtitle(id: Int?)

    var arguments: [String] {
        switch self {
        case .setProperty(let name, let value): ["set", name, value]
        case .loadFile(let url): ["loadfile", url, "replace"]
        case .selectAudio(let id): ["set", "aid", String(id)]
        case .selectSubtitle(let id): ["set", "sid", id.map(String.init) ?? "no"]
        }
    }
}

struct MPVLoadRequest: CustomStringConvertible, Sendable {
    let url: URL
    let headers: [String: String]
    let start: Double

    var headerFieldOption: String {
        headers.keys.sorted().map { "\($0): \(headers[$0]!)" }.joined(separator: ",")
    }

    var commands: [MPVCommand] {
        [.setProperty("start", String(start)), .loadFile(url.absoluteString)]
    }

    var description: String {
        "MPVLoadRequest(url: \(url.absoluteString), headers: \(headers.keys.sorted()), start: \(start))"
    }
}
