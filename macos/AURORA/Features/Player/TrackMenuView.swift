import SwiftUI

enum PlayerTrackMenu { case audio, subtitles, chapters, rate }

struct TrackMenuView: View {
    @Bindable var coordinator: PlaybackCoordinator
    let kind: PlayerTrackMenu
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(kind == .audio ? "Audio" : "Sous-titres").font(AuroraTheme.serif(16, .semibold)).foregroundStyle(AuroraTheme.paper)
                Spacer()
                Button(action: onClose) { Image(systemName: "xmark") }.buttonStyle(.plain)
            }
            Rectangle().fill(AuroraTheme.line).frame(height: 1)
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
        .background(AuroraTheme.background.opacity(0.95), in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(AuroraTheme.line))
        .shadow(color: .black.opacity(0.5), radius: 24, y: 12)
    }

    private func trackButton(_ label: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(spacing: 8) {
                Image(systemName: "checkmark").font(.system(size: 12, weight: .semibold)).foregroundStyle(AuroraTheme.accent).opacity(selected ? 1 : 0)
                Text(label).foregroundStyle(selected ? AuroraTheme.paper : AuroraTheme.mist)
                Spacer()
            }
            .font(.system(size: 13))
            .padding(.horizontal, 8).frame(height: 30)
            .background(selected ? AuroraTheme.accent.opacity(0.12) : .clear, in: RoundedRectangle(cornerRadius: 5))
            .contentShape(Rectangle())
        }.buttonStyle(.plain)
    }

    private func audioLabel(_ track: AudioTrack) -> String {
        [TrackSelection.languageName(track.language), track.codec.uppercased(), track.channels == 6 ? "5.1" : track.channels == 8 ? "7.1" : ""]
            .filter { !$0.isEmpty }.joined(separator: " · ")
    }
}
