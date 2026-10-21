import Foundation
import Observation

@MainActor
@Observable
final class PlaybackCoordinator {
    private let api: any AuroraServing
    private let mpv: any MPVControlling
    private let sessions: any SessionStoring
    private let server: ServerAddress
    private let capabilities: Capabilities
    private let preferences: Preferences
    private let clock: any PlaybackSleeping
    private let cleanup: PlaybackCleanup

    @ObservationIgnored private var eventTask: Task<Void, Never>?
    @ObservationIgnored private var progressTask: Task<Void, Never>?
    @ObservationIgnored private var seekTask: Task<Void, Never>?
    @ObservationIgnored private var countdownTask: Task<Void, Never>?

    private var generation = 0
    private var stopped = false
    private var response: PlayResponse?
    private var intent: PlayIntent?
    private var localDuration = 0.0
    private var chosenAudio: Int?
    private var chosenSubtitle: Int?
    private var forcedMode: String?
    private var trackPreferencesApplied = false
    private var triedRemux = false
    private var triedTranscode = false
    private var networkRetried = false

    private(set) var state: PlaybackState = .idle
    private(set) var position = 0.0
    private(set) var duration = 0.0
    private(set) var buffered = 0.0
    private(set) var tracks: [MPVTrack] = []
    private(set) var chapters: [Chapter] = []
    private(set) var error: String?
    private(set) var nextEpisode: Card?
    private(set) var countdown: Int?

    var isDirect: Bool { response?.mode == "direct" }
    var serverAudioTracks: [AudioTrack] { response?.audio ?? [] }
    var serverSubtitles: [PlaySubtitle] { response?.subtitles ?? [] }
    var activeAudioStream: Int { response?.audioStream ?? -1 }
    var activeSubtitleIndex: Int? { chosenSubtitle }

    init(
        api: any AuroraServing,
        mpv: any MPVControlling,
        sessions: any SessionStoring,
        server: ServerAddress,
        capabilities: Capabilities,
        preferences: Preferences,
        clock: any PlaybackSleeping = ContinuousPlaybackClock(),
        cleanup: PlaybackCleanup? = nil
    ) {
        self.api = api
        self.mpv = mpv
        self.sessions = sessions
        self.server = server
        self.capabilities = capabilities
        self.preferences = preferences
        self.clock = clock
        self.cleanup = cleanup ?? PlaybackCleanup(api: api)
    }

    func start(intent: PlayIntent) async {
        stopped = false
        triedRemux = false
        triedTranscode = false
        networkRetried = false
        forcedMode = nil
        chosenAudio = nil
        chosenSubtitle = nil
        await startSession(intent: intent)
    }

    func togglePause() {
        let pause = state == .playing || state == .buffering
        mpv.setPaused(pause)
        state = pause ? .paused : .playing
        if pause { Task { await saveProgress() } }
    }

    func seek(by delta: Double) { seek(to: position + delta) }

    func seek(to requested: Double) {
        guard let response, let intent else { return }
        let maximum = duration > 1 ? duration - 1 : requested
        let target = min(max(requested, 0), maximum)
        if response.mode == "direct" {
            position = target
            mpv.seek(to: target)
            return
        }

        let local = target - response.offset
        if local >= 0, local <= localDuration - 2, target >= position - 60 {
            position = target
            mpv.seek(to: local)
            return
        }

        seekTask?.cancel()
        seekTask = Task { [weak self] in
            guard let self else { return }
            await clock.sleep(for: .milliseconds(500))
            guard !Task.isCancelled else { return }
            await startSession(intent: PlayIntent(itemID: intent.itemID, fileID: intent.fileID, startPosition: target, restart: false))
        }
    }

    func selectAudio(index: Int) async {
        guard let response else { return }
        if response.mode == "direct" {
            mpv.selectAudio(id: index)
        } else {
            chosenAudio = index
            if let intent {
                await startSession(intent: PlayIntent(itemID: intent.itemID, fileID: intent.fileID, startPosition: position, restart: false))
            }
        }
    }

    func selectSubtitle(index: Int?) async {
        guard let response else { return }
        guard let index else {
            chosenSubtitle = nil
            mpv.selectSubtitle(id: nil)
            return
        }
        guard let subtitle = response.subtitles.first(where: { $0.index == index }) else {
            mpv.selectSubtitle(id: index)
            return
        }
        chosenSubtitle = index
        switch TrackSelection.route(subtitle) {
        case .externalText:
            if let url = api.absoluteURL(subtitle.url) { mpv.addSubtitle(url: url, select: true) }
        case .serverBurnIn:
            chosenSubtitle = index
            forcedMode = "transcode"
            if let intent {
                await startSession(intent: PlayIntent(itemID: intent.itemID, fileID: intent.fileID, startPosition: position, restart: false))
            }
        }
    }

    func setRate(_ rate: Double) { mpv.setRate(min(max(rate, 0.5), 2)) }

    func cycleAudio() async {
        guard let response else { return }
        if response.mode == "direct" {
            let audio = tracks.filter { $0.kind == .audio }
            guard !audio.isEmpty else { return }
            let current = audio.firstIndex(where: \.selected) ?? -1
            await selectAudio(index: audio[(current + 1) % audio.count].id)
        } else {
            let audio = response.audio
            guard !audio.isEmpty else { return }
            let current = audio.firstIndex { $0.stream == response.audioStream } ?? -1
            await selectAudio(index: audio[(current + 1) % audio.count].index)
        }
    }

    func cycleSubtitles() async {
        guard let response else { return }
        if response.mode == "direct" {
            let subtitles = tracks.filter { $0.kind == .subtitle }
            guard !subtitles.isEmpty else { return }
            if let current = subtitles.firstIndex(where: \.selected), current + 1 < subtitles.count {
                await selectSubtitle(index: subtitles[current + 1].id)
            } else if subtitles.contains(where: \.selected) {
                await selectSubtitle(index: nil)
            } else {
                await selectSubtitle(index: subtitles[0].id)
            }
        } else {
            let subtitles = response.subtitles
            guard !subtitles.isEmpty else { return }
            if let current = chosenSubtitle.flatMap({ selected in subtitles.firstIndex { $0.index == selected } }), current + 1 < subtitles.count {
                await selectSubtitle(index: subtitles[current + 1].index)
            } else if chosenSubtitle != nil {
                await selectSubtitle(index: nil)
            } else {
                await selectSubtitle(index: subtitles[0].index)
            }
        }
    }

    func stop() async {
        guard !stopped else { return }
        generation += 1
        stopped = true
        seekTask?.cancel()
        countdownTask?.cancel()
        progressTask?.cancel()
        await saveProgress()
        let sessionID = response?.sessionId
        response = nil
        mpv.stop()
        await cleanup.session(sessionID)
        state = .ended
    }

    func playNext() async {
        guard let nextEpisode else { return }
        countdownTask?.cancel()
        countdown = nil
        self.nextEpisode = nil
        await start(intent: PlayIntent(itemID: nextEpisode.id, fileID: nil, startPosition: nextEpisode.position ?? 0, restart: false))
    }

    func cancelCountdown() {
        countdownTask?.cancel()
        countdown = nil
        nextEpisode = nil
    }

    private func startSession(intent: PlayIntent) async {
        generation += 1
        let requestGeneration = generation
        self.intent = intent
        state = .preparing
        error = nil
        ensureEventLoop()

        do {
            let fileID: Int64
            if let selected = intent.fileID {
                fileID = selected
            } else {
                let detail = try await api.detail(itemID: intent.itemID)
                guard requestGeneration == generation, !stopped else { return }
                guard let file = detail.state.lastFileId.flatMap({ id in detail.files.first { $0.id == id } }) ?? detail.files.first else {
                    throw APIError(code: 404, message: "Aucun fichier vidéo pour cette œuvre.")
                }
                fileID = file.id
                self.intent = PlayIntent(itemID: intent.itemID, fileID: fileID, startPosition: intent.startPosition, restart: intent.restart)
            }

            let request = PlayRequest(
                fileId: fileID,
                caps: capabilities,
                start: intent.startPosition,
                audioIndex: chosenAudio,
                subtitleIndex: chosenSubtitle,
                forceMode: forcedMode,
                external: false
            )
            let received = try await api.play(request)
            guard requestGeneration == generation, !stopped else {
                await cleanup.session(received.sessionId)
                return
            }
            await apply(received, requestedStart: intent.startPosition, generation: requestGeneration)
        } catch is CancellationError {
        } catch {
            guard requestGeneration == generation, !stopped else { return }
            fail(error.localizedDescription)
        }
    }

    private func apply(_ received: PlayResponse, requestedStart: Double, generation requestGeneration: Int) async {
        guard let url = api.absoluteURL(received.url) else {
            await cleanup.session(received.sessionId)
            fail("Adresse vidéo invalide.")
            return
        }
        let oldSession = response?.sessionId
        let token = try? await sessions.token(for: server.origin)
        guard requestGeneration == generation, !stopped else {
            await cleanup.session(received.sessionId)
            return
        }

        let initial = TrackSelection.initial(audio: received.audio, subtitles: received.subtitles, preferences: preferences)
        if received.mode != "direct", chosenAudio == nil,
           let preferredAudio = initial.audioIndex,
           received.audio.first(where: { $0.stream == received.audioStream })?.index != preferredAudio {
            chosenAudio = preferredAudio
            chosenSubtitle = initial.subtitleIndex
            await cleanup.session(received.sessionId)
            guard requestGeneration == generation, !stopped, let intent else { return }
            await startSession(intent: intent)
            return
        }

        response = received
        position = requestedStart
        duration = received.duration
        buffered = received.offset
        localDuration = 0
        chapters = received.chapters
        tracks = []
        trackPreferencesApplied = false
        if chosenSubtitle == nil,
           let preferredSubtitle = initial.subtitleIndex,
           received.mode != "direct" || received.subtitles.first(where: { $0.index == preferredSubtitle })?.external == true {
            chosenSubtitle = preferredSubtitle
        }
        let headers = token.map { ["Cookie": "aurora_session=\($0)", "X-Aurora": "1"] } ?? ["X-Aurora": "1"]
        mpv.load(url: url, headers: headers, start: received.mode == "direct" ? requestedStart : 0)
        for subtitle in received.subtitles where subtitle.text && subtitle.url != nil && (received.mode != "direct" || subtitle.external) {
            if let url = api.absoluteURL(subtitle.url) {
                mpv.addSubtitle(url: url, select: subtitle.index == chosenSubtitle)
            }
        }
        mpv.setSubtitleScale(TrackSelection.subtitleScale(preferences.subSize))
        if received.mode == "direct" { await cleanup.session(oldSession) }
        state = .playing
        startProgressLoop()
    }

    private func ensureEventLoop() {
        guard eventTask == nil else { return }
        eventTask = Task { [weak self] in
            guard let self else { return }
            for await event in mpv.events {
                guard !Task.isCancelled else { return }
                await handle(event)
            }
        }
    }

    private func handle(_ event: MPVEvent) async {
        switch event {
        case .ready:
            state = .playing
            networkRetried = false
        case .time(let local):
            position = (response?.offset ?? 0) + local
        case .duration(let local):
            localDuration = local
            buffered = (response?.mode == "direct" ? 0 : response?.offset ?? 0) + local
        case .paused(let paused):
            state = paused ? .paused : .playing
            if paused { await saveProgress() }
        case .buffering(let waiting):
            state = waiting ? .buffering : .playing
        case .tracks(let updated):
            tracks = updated
            applyTrackPreferencesOnce(updated)
        case .ended:
            await playbackEnded()
        case .failed(let failure):
            await recover(from: failure)
        }
    }

    private func applyTrackPreferencesOnce(_ updated: [MPVTrack]) {
        guard !trackPreferencesApplied else { return }
        trackPreferencesApplied = true
        if preferences.subMode == "off" {
            mpv.selectSubtitle(id: nil)
        } else if response?.mode == "direct", chosenSubtitle == nil {
            let audioLanguage = preferences.audioLang.isEmpty
                ? updated.first { $0.kind == .audio && $0.selected }?.language
                : preferences.audioLang
            mpv.selectSubtitle(id: TrackSelection.preferredSubtitleID(in: updated, audioLanguage: audioLanguage, preferences: preferences))
        }
        guard response?.mode == "direct" else { return }
        if let id = TrackSelection.preferredAudioID(in: updated, language: preferences.audioLang),
           !updated.contains(where: { $0.id == id && $0.selected }) {
            mpv.selectAudio(id: id)
        }
    }

    private func recover(from failure: MPVFailure) async {
        guard let response, let intent else { fail("Le lecteur n’a pas pu lire ce fichier (\(failure.message))."); return }
        let message = failure.message.lowercased()
        let absolute = position
        if message.contains("video") || message.contains("decode") {
            if response.mode != "transcode", !triedTranscode {
                triedTranscode = true
                forcedMode = "transcode"
                await startSession(intent: PlayIntent(itemID: intent.itemID, fileID: intent.fileID, startPosition: absolute, restart: false))
                return
            }
        } else if response.mode != "direct", (message.contains("network") || message.contains("http")), !networkRetried {
            networkRetried = true
            await startSession(intent: PlayIntent(itemID: intent.itemID, fileID: intent.fileID, startPosition: absolute, restart: false))
            return
        } else if response.mode == "direct", !triedRemux {
            triedRemux = true
            forcedMode = "remux"
            await startSession(intent: PlayIntent(itemID: intent.itemID, fileID: intent.fileID, startPosition: absolute, restart: false))
            return
        }
        fail("Le lecteur intégré n’a pas pu lire ce fichier (\(failure.message)).")
    }

    private func playbackEnded() async {
        await saveProgress()
        state = .ended
        nextEpisode = try? await api.next(episodeID: intent?.itemID ?? 0)
        guard nextEpisode != nil, preferences.autoplayNext else { return }
        countdown = 8
        countdownTask?.cancel()
        countdownTask = Task { [weak self] in
            guard let self else { return }
            while let value = countdown, value > 0 {
                await clock.sleep(for: .seconds(1))
                guard !Task.isCancelled else { return }
                countdown = value - 1
            }
            if !Task.isCancelled { await playNext() }
        }
    }

    private func startProgressLoop() {
        progressTask?.cancel()
        progressTask = Task { [weak self] in
            guard let self else { return }
            while response != nil, !Task.isCancelled {
                await clock.sleep(for: .seconds(10))
                guard !Task.isCancelled else { return }
                if state == .playing { await saveProgress() }
            }
        }
    }

    private func saveProgress() async {
        guard let response, position > 0 else { return }
        await api.progress(itemID: response.itemId, fileID: response.fileId, position: position, duration: duration)
    }

    private func fail(_ message: String) {
        error = message
        state = .failed
    }
}
