import SwiftUI

struct PlayerControlsView: View {
    @Bindable var coordinator: PlaybackCoordinator
    let title: String
    @Binding var menu: PlayerTrackMenu?
    let onStop: () -> Void

    var body: some View {
        VStack(alignment: .leading) {
            HStack(alignment: .top) {
                Text(title).font(AuroraTheme.serif(24, .semibold)).foregroundStyle(AuroraTheme.paper).lineLimit(1)
                Spacer()
                Button(action: onStop) { Image(systemName: "xmark") }
                    .buttonStyle(PlayerIconButtonStyle())
                    .help("Fermer")
                    .accessibilityLabel("Fermer le lecteur")
            }
            .padding(.top, 22)
            Spacer()
            VStack(spacing: 12) {
                HStack(spacing: 14) {
                    Text(ChapterMenuView.clock(coordinator.position))
                    ZStack(alignment: .leading) {
                        GeometryReader { proxy in
                            Rectangle().fill(AuroraTheme.paper.opacity(0.35))
                                .frame(width: proxy.size.width * progress(coordinator.buffered), height: 3)
                                .frame(maxHeight: .infinity)
                            ForEach(coordinator.chapters) { chapter in
                                Rectangle().fill(AuroraTheme.paper.opacity(0.6)).frame(width: 1, height: 9)
                                    .offset(x: proxy.size.width * progress(chapter.start))
                                    .frame(maxHeight: .infinity)
                            }
                        }
                        Slider(value: Binding(get: { coordinator.position }, set: { coordinator.seek(to: $0) }), in: 0...max(coordinator.duration, 1))
                            .tint(AuroraTheme.accent)
                            .controlSize(.small)
                    }
                    .frame(height: 16)
                    Text("−\(ChapterMenuView.clock(max(coordinator.duration - coordinator.position, 0)))")
                }
                .font(AuroraTheme.serif(14)).monospacedDigit().foregroundStyle(AuroraTheme.mist)

                ZStack {
                    HStack(spacing: 22) {
                        Button { coordinator.seek(by: -10) } label: { Image(systemName: "gobackward.10") }
                            .buttonStyle(PlayerIconButtonStyle(size: 22)).help("Reculer de 10 s")
                        Button { coordinator.togglePause() } label: {
                            Image(systemName: coordinator.state == .paused ? "play.fill" : "pause.fill")
                                .font(.system(size: 22))
                                .foregroundStyle(AuroraTheme.ink)
                                .frame(width: 52, height: 52)
                                .background(AuroraTheme.buttonFill, in: Circle())
                        }
                        .buttonStyle(.plain)
                        .keyboardShortcut(.space, modifiers: [])
                        .accessibilityLabel(coordinator.state == .paused ? "Lecture" : "Pause")
                        Button { coordinator.seek(by: 30) } label: { Image(systemName: "goforward.30") }
                            .buttonStyle(PlayerIconButtonStyle(size: 22)).help("Avancer de 30 s")
                    }
                    HStack(spacing: 4) {
                        Spacer()
                        if !coordinator.chapters.isEmpty {
                            toggle(.chapters, symbol: "list.bullet", help: "Chapitres")
                        }
                        toggle(.audio, symbol: "speaker.wave.2", help: "Audio")
                        toggle(.subtitles, symbol: "captions.bubble", help: "Sous-titres")
                        Menu {
                            ForEach([0.5, 0.75, 1, 1.25, 1.5, 2], id: \.self) { rate in
                                Button("\(rate, specifier: "%g")×") { coordinator.setRate(rate) }
                            }
                        } label: {
                            Image(systemName: "gauge.with.dots.needle.67percent")
                        }
                        .menuStyle(.borderlessButton)
                        .menuIndicator(.hidden)
                        .fixedSize()
                        .foregroundStyle(AuroraTheme.paper)
                        .frame(width: 38, height: 38)
                        .help("Vitesse")
                        Button(action: onStop) { Image(systemName: "stop.fill") }
                            .buttonStyle(PlayerIconButtonStyle()).help("Arrêter")
                    }
                }
            }
            .padding(.bottom, 22)
        }
        .padding(.horizontal, 40)
        .background(
            LinearGradient(stops: [
                .init(color: .black.opacity(0.65), location: 0),
                .init(color: .clear, location: 0.2),
                .init(color: .clear, location: 0.55),
                .init(color: AuroraTheme.background.opacity(0.94), location: 1)
            ], startPoint: .top, endPoint: .bottom)
        )
    }

    private func toggle(_ kind: PlayerTrackMenu, symbol: String, help: String) -> some View {
        Button { menu = menu == kind ? nil : kind } label: { Image(systemName: symbol) }
            .buttonStyle(PlayerIconButtonStyle(active: menu == kind))
            .help(help)
            .accessibilityLabel(help)
    }

    private func progress(_ value: Double) -> Double {
        guard coordinator.duration > 0 else { return 0 }
        return min(max(value / coordinator.duration, 0), 1)
    }
}

/// Bouton rond du lecteur : crème au repos, or quand son menu est ouvert.
struct PlayerIconButtonStyle: ButtonStyle {
    var size: CGFloat = 17
    var active = false
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: size))
            .foregroundStyle(active ? AuroraTheme.accent : AuroraTheme.paper)
            .frame(width: 38, height: 38)
            .contentShape(Circle())
            .opacity(configuration.isPressed ? 0.6 : 1)
    }
}
