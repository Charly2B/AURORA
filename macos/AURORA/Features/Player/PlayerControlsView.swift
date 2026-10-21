import SwiftUI

struct PlayerControlsView: View {
    @Bindable var coordinator: PlaybackCoordinator
    let title: String
    @Binding var menu: PlayerTrackMenu?
    let onStop: () -> Void

    var body: some View {
        VStack {
            HStack {
                Text(title).font(.title2.bold()).lineLimit(1)
                Spacer()
                Button(action: onStop) { Label("Fermer", systemImage: "xmark") }
            }
            Spacer()
            VStack(spacing: 12) {
                ZStack(alignment: .leading) {
                    GeometryReader { proxy in
                        Capsule().fill(.white.opacity(0.18)).frame(height: 4)
                        Capsule().fill(.white.opacity(0.35)).frame(width: proxy.size.width * progress(coordinator.buffered), height: 4)
                        ForEach(coordinator.chapters) { chapter in
                            Rectangle().fill(.white.opacity(0.7)).frame(width: 1, height: 10)
                                .offset(x: proxy.size.width * progress(chapter.start))
                        }
                    }.frame(height: 10)
                    Slider(value: Binding(get: { coordinator.position }, set: { coordinator.seek(to: $0) }), in: 0...max(coordinator.duration, 1))
                        .tint(AuroraTheme.accent)
                }
                HStack {
                    Text(ChapterMenuView.clock(coordinator.position)).monospacedDigit()
                    Spacer()
                    Text("−\(ChapterMenuView.clock(max(coordinator.duration - coordinator.position, 0)))").monospacedDigit()
                }.font(.caption)
                HStack(spacing: 18) {
                    Button { coordinator.seek(by: -10) } label: { Image(systemName: "gobackward.10") }
                    Button { coordinator.togglePause() } label: { Image(systemName: coordinator.state == .paused ? "play.fill" : "pause.fill") }.keyboardShortcut(.space, modifiers: [])
                    Button { coordinator.seek(by: 30) } label: { Image(systemName: "goforward.30") }
                    Divider().frame(height: 22)
                    Button("Audio") { menu = menu == .audio ? nil : .audio }
                    Button("Sous-titres") { menu = menu == .subtitles ? nil : .subtitles }
                    if !coordinator.chapters.isEmpty { Button("Chapitres") { menu = menu == .chapters ? nil : .chapters } }
                    Menu("Vitesse") {
                        ForEach([0.5, 0.75, 1, 1.25, 1.5, 2], id: \.self) { rate in
                            Button("\(rate, specifier: "%g")×") { coordinator.setRate(rate) }
                        }
                    }
                    Spacer()
                    Button("Arrêter", action: onStop)
                }
                .buttonStyle(.borderless)
                .font(.title3)
            }
        }
        .padding(24)
        .background(.linearGradient(colors: [.black.opacity(0.8), .clear, .black.opacity(0.92)], startPoint: .top, endPoint: .bottom))
    }

    private func progress(_ value: Double) -> Double {
        guard coordinator.duration > 0 else { return 0 }
        return min(max(value / coordinator.duration, 0), 1)
    }
}
