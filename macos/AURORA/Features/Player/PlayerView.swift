import AppKit
import SwiftUI

struct PlayerView: View {
    @Bindable var coordinator: PlaybackCoordinator
    let controller: any MPVControlling
    let intent: PlayIntent
    let title: String
    let images: ImageRepository
    let onClose: () -> Void

    @State private var controlsVisible = true
    @State private var menu: PlayerTrackMenu?
    @State private var keyMonitor: Any?
    @State private var hideTask: Task<Void, Never>?

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()
            MPVRenderView(controller: controller).ignoresSafeArea()

            if controlsVisible {
                PlayerControlsView(coordinator: coordinator, title: title, menu: $menu, onStop: close)
                    .transition(.opacity)
                if let menu {
                    VStack { Spacer()
                        HStack { Spacer()
                            Group {
                                if menu == .chapters {
                                    ChapterMenuView(chapters: coordinator.chapters, onSelect: coordinator.seek(to:), onClose: { self.menu = nil })
                                } else if menu == .rate {
                                    EmptyView()
                                } else {
                                    TrackMenuView(coordinator: coordinator, kind: menu, onClose: { self.menu = nil })
                                }
                            }.padding(.trailing, 30).padding(.bottom, 92)
                        }
                    }
                }
            }

            if coordinator.state == .preparing || coordinator.state == .buffering {
                ProgressView(coordinator.state == .preparing ? "Préparation de la lecture…" : "Mise en mémoire tampon…")
                    .padding(18).background(.black.opacity(0.7), in: RoundedRectangle(cornerRadius: 12))
            }
            if let next = coordinator.nextEpisode {
                VStack { Spacer(); HStack { Spacer()
                    NextEpisodeView(card: next, countdown: coordinator.countdown, images: images, onPlay: { Task { await coordinator.playNext() } }, onCancel: coordinator.cancelCountdown)
                        .padding(38)
                }}
            }
            if let error = coordinator.error {
                VStack(spacing: 14) {
                    Image(systemName: "exclamationmark.triangle.fill").font(.largeTitle).foregroundStyle(.orange)
                    Text(error).multilineTextAlignment(.center)
                    Button("Fermer", action: close).buttonStyle(.borderedProminent)
                }.padding(28).background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 16)).frame(maxWidth: 520)
            }
        }
        .foregroundStyle(.white)
        .onHover { _ in showControls() }
        .onChange(of: coordinator.state) { _, _ in scheduleHide() }
        .onChange(of: menu != nil) { _, _ in scheduleHide() }
        .onAppear {
            ApplicationTerminationCoordinator.shared.register(coordinator)
            installKeyMonitor()
            Task { await coordinator.start(intent: intent) }
            scheduleHide()
        }
        .onDisappear {
            ApplicationTerminationCoordinator.shared.unregister(coordinator)
            removeKeyMonitor()
            hideTask?.cancel()
            Task { await coordinator.stop() }
        }
    }

    private func installKeyMonitor() {
        keyMonitor = NSEvent.addLocalMonitorForEvents(matching: [.keyDown, .systemDefined]) { event in
            guard let command = PlayerKeyCommand.map(event) else { return event }
            handle(command)
            return nil
        }
    }

    private func removeKeyMonitor() {
        if let keyMonitor { NSEvent.removeMonitor(keyMonitor) }
        keyMonitor = nil
    }

    private func handle(_ command: PlayerKeyCommand) {
        showControls()
        switch command {
        case .togglePause: coordinator.togglePause()
        case .seek(let delta): coordinator.seek(by: delta)
        case .fullscreen: NSApp.keyWindow?.toggleFullScreen(nil)
        case .closePanelOrPlayer:
            if menu != nil { menu = nil } else { close() }
        case .cycleAudio:
            Task { await coordinator.cycleAudio() }
        case .cycleSubtitles:
            Task { await coordinator.cycleSubtitles() }
        }
    }

    private func showControls() { controlsVisible = true; scheduleHide() }

    private func scheduleHide() {
        hideTask?.cancel()
        guard PlayerControlsVisibility.shouldAutoHide(playback: coordinator.state, menuOpen: menu != nil) else { controlsVisible = true; return }
        hideTask = Task {
            try? await Task.sleep(for: .seconds(6))
            guard !Task.isCancelled else { return }
            withAnimation { controlsVisible = false }
        }
    }

    private func close() {
        Task { await coordinator.stop(); onClose() }
    }
}
