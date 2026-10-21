enum SubtitleRoute: Equatable, Sendable {
    case externalText
    case serverBurnIn
}

enum TrackSelection {
    struct Initial: Equatable, Sendable {
        let audioIndex: Int?
        let subtitleIndex: Int?
    }

    static func preferredAudioID(in tracks: [MPVTrack], language: String) -> Int? {
        guard !language.isEmpty else { return nil }
        let accepted = languageCodes(language)
        return tracks.first { $0.kind == .audio && accepted.contains($0.language.lowercased()) }?.id
    }

    static func preferredSubtitleID(
        in tracks: [MPVTrack],
        audioLanguage: String?,
        preferences: Preferences
    ) -> Int? {
        guard preferences.subMode != "off" else { return nil }
        let language = preferences.subLang.isEmpty ? preferences.audioLang : preferences.subLang
        let candidates = tracks.filter {
            $0.kind == .subtitle && (language.isEmpty || languageCodes(language).contains($0.language.lowercased()))
        }
        switch preferences.subMode {
        case "always":
            return candidates.first(where: { !$0.forced })?.id ?? candidates.first?.id
        default:
            guard !language.isEmpty else { return candidates.first(where: \.forced)?.id }
            if let audioLanguage, !sameLanguage(audioLanguage, language) {
                return candidates.first(where: { !$0.forced })?.id ?? candidates.first?.id
            }
            return candidates.first(where: \.forced)?.id
        }
    }

    static func initial(
        audio: [AudioTrack],
        subtitles: [PlaySubtitle],
        preferences: Preferences
    ) -> Initial {
        let preferredAudio = audio.first { sameLanguage($0.language, preferences.audioLang) }
            ?? audio.first(where: \.isDefault)
            ?? audio.first
        let language = preferences.subLang.isEmpty ? preferences.audioLang : preferences.subLang
        let text = subtitles.filter { $0.text && sameLanguage($0.language, language) }
        let subtitle: PlaySubtitle?
        switch preferences.subMode {
        case "off":
            subtitle = nil
        case "always":
            subtitle = text.first(where: { !$0.forced }) ?? text.first
        default:
            if let audioLanguage = preferredAudio?.language,
               !audioLanguage.isEmpty,
               !language.isEmpty,
               !sameLanguage(audioLanguage, language) {
                subtitle = text.first(where: { !$0.forced }) ?? text.first
            } else {
                subtitle = text.first(where: \.forced)
            }
        }
        return Initial(audioIndex: preferredAudio?.index, subtitleIndex: subtitle?.index)
    }

    static func subtitleScale(_ size: String) -> Double {
        switch size { case "s": 0.82; case "l": 1.25; default: 1 }
    }

    static func route(_ subtitle: PlaySubtitle) -> SubtitleRoute {
        subtitle.text && subtitle.url != nil ? .externalText : .serverBurnIn
    }

    static func languageName(_ code: String) -> String {
        switch code.lowercased() {
        case "", "und": "Langue inconnue"
        case "fre", "fra", "fr": "Français"
        case "eng", "en": "Anglais"
        case "ger", "deu", "de": "Allemand"
        case "spa", "es": "Espagnol"
        case "ita", "it": "Italien"
        default: code.uppercased()
        }
    }

    static func sameLanguage(_ lhs: String, _ rhs: String) -> Bool {
        !lhs.isEmpty && !rhs.isEmpty && !languageCodes(lhs).isDisjoint(with: languageCodes(rhs))
    }

    private static func languageCodes(_ code: String) -> Set<String> {
        switch code.lowercased() {
        case "fre", "fra", "fr": ["fre", "fra", "fr"]
        case "eng", "en": ["eng", "en"]
        case "ger", "deu", "de": ["ger", "deu", "de"]
        default: [code.lowercased()]
        }
    }
}
