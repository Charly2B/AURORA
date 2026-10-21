import SwiftUI

enum PlayerTrackMenu { case audio, subtitles, chapters, rate }

struct TrackMenuView: View {
    @Bindable var coordinator: PlaybackCoordinator
    let kind: PlayerTrackMenu
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(kind == .audio ? "Audio" : "Sous-titres").font(.headline)
                Spacer()
                Button(action: onClose) { Image(systemName: "xmark") }.buttonStyle(.plain)
            }
            Divider()
            if kind == .audio {
                if coordinator.isDirect {
                    ForEach(coordinator.tracks.filter { $0.kind == .audio }) { track in
                        trackButton(TrackSelection.languageName(track.language), selected: track.selected) {
                            Task { await coordinator.selectAudio(index: track.id); onClose() }
                        }
                    }
                } else {
                    ForEach(coordinator.serverAudioTracks) { track in
                        trackButton(audioLabel(track), selected: track.stream == coordinator.activeAudioStream) {
                            Task { await coordinator.selectAudio(index: track.index); onClose() }
                        }
                    }
                }
            } else {
                trackButton("Aucun", selected: coordinator.activeSubtitleIndex == nil && !coordinator.tracks.contains(where: { $0.kind == .subtitle && $0.selected })) { Task { await coordinator.selectSubtitle(index: nil); onClose() } }
                if !coordinator.isDirect {
                    ForEach(coordinator.serverSubtitles) { track in
                        trackButton(track.label.isEmpty ? TrackSelection.languageName(track.language) : track.label, selected: track.index == coordinator.activeSubtitleIndex) {
                            Task { await coordinator.selectSubtitle(index: track.index); onClose() }
                        }
                    }
                } else {
                    ForEach(coordinator.tracks.filter { $0.kind == .subtitle }) { track in
                        trackButton(track.title.isEmpty ? TrackSelection.languageName(track.language) : track.title, selected: track.selected) {
                            Task { await coordinator.selectSubtitle(index: track.id); onClose() }
                        }
                    }
                }
            }
        }
        .padding(16)
        .frame(width: 310, alignment: .leading)
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 14))
    }

    private func trackButton(_ label: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack { Image(systemName: selected ? "checkmark.circle.fill" : "circle"); Text(label); Spacer() }
        }.buttonStyle(.plain).padding(.vertical, 5)
    }

    private func audioLabel(_ track: AudioTrack) -> String {
        [TrackSelection.languageName(track.language), track.codec.uppercased(), track.channels == 6 ? "5.1" : track.channels == 8 ? "7.1" : ""]
            .filter { !$0.isEmpty }.joined(separator: " · ")
    }
}
