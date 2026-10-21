import Testing
@testable import AURORA

@Suite("Sélection des pistes")
struct TrackSelectionTests {
    @Test func frenchPreferenceUsesMpvTrackIDOnce() {
        let tracks = [
            MPVTrack(id: 4, kind: .audio, title: "English", language: "eng", codec: "aac", selected: true, forced: false, isDefault: true),
            MPVTrack(id: 17, kind: .audio, title: "Français", language: "fre", codec: "eac3", selected: false, forced: false, isDefault: false)
        ]
        #expect(TrackSelection.preferredAudioID(in: tracks, language: "fra") == 17)
        #expect(TrackSelection.languageName("fre") == "Français")
        #expect(TrackSelection.languageName("eng") == "Anglais")
    }

    @Test func textAndImageSubtitlesRouteDifferently() {
        let text: PlaySubtitle = decode(#"{"index":3,"label":"Français","language":"fre","codec":"srt","text":true,"url":"/sub/3.vtt"}"#)
        let image: PlaySubtitle = decode(#"{"index":9,"label":"PGS","codec":"hdmv_pgs_subtitle","text":false}"#)
        #expect(TrackSelection.route(text) == .externalText)
        #expect(TrackSelection.route(image) == .serverBurnIn)
    }

    @Test func profilePreferencesChooseAudioSubtitleModeLanguageAndSize() {
        let audio: [AudioTrack] = decode(#"[{"index":0,"stream":1,"language":"eng","default":true},{"index":1,"stream":2,"language":"fre"}]"#)
        let subtitles: [PlaySubtitle] = decode(#"[{"index":0,"language":"fre","forced":true,"text":true,"url":"/forced.vtt"},{"index":1,"language":"fre","text":true,"url":"/full.vtt"},{"index":2,"language":"fre","text":false}]"#)
        var preferences = Preferences()
        preferences.audioLang = "eng"
        preferences.subLang = "fre"
        preferences.subMode = "auto"

        #expect(TrackSelection.initial(audio: audio, subtitles: subtitles, preferences: preferences) == .init(audioIndex: 0, subtitleIndex: 1))
        preferences.subMode = "off"
        #expect(TrackSelection.initial(audio: audio, subtitles: subtitles, preferences: preferences).subtitleIndex == nil)
        #expect(TrackSelection.subtitleScale("s") < TrackSelection.subtitleScale("m"))
        #expect(TrackSelection.subtitleScale("l") > TrackSelection.subtitleScale("m"))
    }
}
