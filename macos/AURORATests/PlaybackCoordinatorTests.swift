import Foundation
import Testing
@testable import AURORA

@Suite("Coordinateur de lecture", .serialized)
@MainActor
struct PlaybackCoordinatorTests {
    @Test func directLoadsAtAbsoluteResumeAndHLSUsesLocalZeroWithOffset() async throws {
        let direct = play(mode: "direct", url: "/video.mkv", offset: 0, session: nil, start: 120)
        let context = try await Context.make(defaultPlays: [FakePlayPlan(.value(direct))])

        await context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 120, restart: false))
        #expect(context.mpv.loads.last?.start == 120)
        #expect(context.mpv.loads.last?.headers["Cookie"] == "aurora_session=token")

        let hls = play(mode: "remux", url: "/hls/a/index.m3u8", offset: 100, session: "a", start: 100)
        let hlsContext = try await Context.make(defaultPlays: [FakePlayPlan(.value(hls))])
        await hlsContext.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 100, restart: false))
        #expect(hlsContext.mpv.loads.last?.start == 0)
        hlsContext.mpv.emit(.time(5))
        await Task.yield()
        #expect(hlsContext.coordinator.position == 105)
    }

    @Test func HLSLocalSeekAndDebouncedFarSeek() async throws {
        let first = play(mode: "remux", url: "/hls/a/index.m3u8", offset: 100, session: "a", start: 100)
        let second = play(mode: "remux", url: "/hls/b/index.m3u8", offset: 300, session: "b", start: 300)
        let context = try await Context.make(plays: [100: FakePlayPlan(.value(first)), 300: FakePlayPlan(.value(second))])
        await context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 100, restart: false))
        context.mpv.emit(.duration(120))
        context.mpv.emit(.time(20))
        await Task.yield()

        context.coordinator.seek(to: 150)
        #expect(context.mpv.commands.contains("seek:50.0"))
        context.coordinator.seek(to: 260)
        context.coordinator.seek(to: 300)
        let debounceRegistered = await eventually { await context.clock.pending.contains(.milliseconds(500)) }
        #expect(debounceRegistered)
        await context.clock.advance(.milliseconds(500))
        try await Task.sleep(for: .milliseconds(20))
        #expect(await context.api.snapshot().playRequests.map(\.start) == [100, 300])
    }

    @Test func serverAudioAndSubtitleRoutesAreApplied() async throws {
        let response = play(mode: "remux", url: "/hls/a/index.m3u8", offset: 0, session: "a", start: 0, subtitles: [
            #"{"index":3,"label":"SRT","text":true,"external":true,"url":"/sub.vtt"}"#,
            #"{"index":9,"label":"PGS","text":false}"#
        ])
        let audioResponse = play(mode: "remux", url: "/hls/b/index.m3u8", offset: 0, session: "b", start: 0, subtitles: [
            #"{"index":3,"label":"SRT","text":true,"external":true,"url":"/sub.vtt"}"#,
            #"{"index":9,"label":"PGS","text":false}"#
        ])
        let burnResponse = play(mode: "transcode", url: "/hls/c/index.m3u8", offset: 0, session: "c", start: 0)
        let context = try await Context.make(defaultPlays: [FakePlayPlan(.value(response)), FakePlayPlan(.value(audioResponse)), FakePlayPlan(.value(burnResponse))])
        await context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))

        await context.coordinator.selectSubtitle(index: 3)
        #expect(context.mpv.commands.contains { $0.contains("sub-add:") })
        await context.coordinator.selectAudio(index: 2)
        await context.coordinator.selectSubtitle(index: 9)
        let requests = await context.api.snapshot().playRequests
        #expect(requests.count == 3)
        if requests.count == 3 {
            #expect(requests[1].audioIndex == 2)
            #expect(requests[2].subtitleIndex == 9)
            #expect(requests[2].forceMode == "transcode")
        }
    }

    @Test func HLSSubtitleShortcutUsesServerIndexesAndProfileDoesNotSelectEveryTrack() async throws {
        let response = play(mode: "remux", url: "/hls/a/index.m3u8", offset: 0, session: "a", start: 0, subtitles: [
            #"{"index":3,"label":"SRT","language":"fre","text":true,"external":true,"url":"/sub.vtt"}"#,
            #"{"index":9,"label":"PGS","language":"fre","text":false}"#
        ])
        var preferences = Preferences()
        preferences.subMode = "off"
        preferences.subSize = "l"
        let context = try await Context.make(defaultPlays: [FakePlayPlan(.value(response))], preferences: preferences)

        await context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))

        #expect(context.mpv.commands.contains("sub-add:http://nas:8890/sub.vtt:auto"))
        #expect(context.mpv.commands.contains("sub-scale:1.25"))
        await context.coordinator.cycleSubtitles()
        #expect(context.mpv.commands.contains("sub-add:http://nas:8890/sub.vtt:cached"))
        #expect(context.coordinator.activeSubtitleIndex == 3)
    }

    @Test func staleResponseLoadsOnlyNewestAndCleansOldSession() async throws {
        let slow = play(mode: "remux", url: "/hls/old/index.m3u8", offset: 10, session: "old", start: 10)
        let fast = play(mode: "remux", url: "/hls/new/index.m3u8", offset: 20, session: "new", start: 20)
        let context = try await Context.make(plays: [
            10: FakePlayPlan(.value(slow), delay: .milliseconds(80)),
            20: FakePlayPlan(.value(fast), delay: .milliseconds(10))
        ])

        async let old: Void = context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 10, restart: false))
        try await Task.sleep(for: .milliseconds(5))
        async let new: Void = context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 20, restart: false))
        _ = await (old, new)

        #expect(context.mpv.loads.map(\.url.lastPathComponent) == ["index.m3u8"])
        #expect(context.mpv.loads.last?.url.absoluteString.contains("/new/") == true)
        #expect(await context.api.snapshot().stoppedSessionIDs.contains("old"))
    }

    @Test func stopDuringSuspendedPlayCleansSessionWithoutLoading() async throws {
        let late = play(mode: "remux", url: "/hls/late/index.m3u8", offset: 0, session: "late", start: 0)
        let context = try await Context.make(defaultPlays: [FakePlayPlan(.value(late), delay: .milliseconds(60))])

        async let starting: Void = context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))
        try await Task.sleep(for: .milliseconds(5))
        await context.coordinator.stop()
        await starting

        #expect(context.mpv.loads.isEmpty)
        #expect(await context.api.snapshot().stoppedSessionIDs == ["late"])
    }

    @Test func progressSavesEveryTenSecondsOnPauseAndClose() async throws {
        let direct = play(mode: "direct", url: "/video.mkv", offset: 0, session: nil, start: 0)
        let context = try await Context.make(defaultPlays: [FakePlayPlan(.value(direct))])
        await context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))
        context.mpv.emit(.time(15))
        _ = await eventually { context.coordinator.position == 15 }

        #expect(await eventually { await context.clock.pending.contains(.seconds(10)) })
        await context.clock.advance(.seconds(10))
        #expect(await eventually { await context.api.snapshot().progressCalls.count == 1 })
        context.coordinator.togglePause()
        #expect(await eventually { await context.api.snapshot().progressCalls.count == 2 })
        await context.coordinator.stop()
        #expect(await context.api.snapshot().progressCalls.count == 3)
    }

    @Test func failuresRetryHLSThenEndExplicitlyWithoutVLC() async throws {
        let first = play(mode: "remux", url: "/hls/a/index.m3u8", offset: 0, session: "a", start: 0)
        let retry = play(mode: "remux", url: "/hls/b/index.m3u8", offset: 12, session: "b", start: 12)
        let context = try await Context.make(defaultPlays: [FakePlayPlan(.value(first)), FakePlayPlan(.value(retry))])
        await context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))
        context.mpv.emit(.time(12))
        context.mpv.emit(.failed(MPVFailure(code: -1, message: "HTTP network error")))
        #expect(await eventually { await context.api.snapshot().playRequests.count == 2 })

        context.mpv.emit(.failed(MPVFailure(code: -1, message: "HTTP network error")))
        #expect(await eventually { context.coordinator.state == .failed })
        #expect(await context.api.snapshot().playRequests.count == 2)
        #expect(context.coordinator.error?.contains("VLC") == false)
    }

    @Test func directFailureFallsBackToRemuxAndVideoDecodeToTranscode() async throws {
        let direct = play(mode: "direct", url: "/video.mkv", offset: 0, session: nil, start: 0)
        let remux = play(mode: "remux", url: "/hls/r/index.m3u8", offset: 0, session: "r", start: 0)
        let remuxContext = try await Context.make(defaultPlays: [FakePlayPlan(.value(direct)), FakePlayPlan(.value(remux))])
        await remuxContext.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))
        remuxContext.mpv.emit(.failed(MPVFailure(code: -2, message: "format audio inconnu")))
        #expect(await eventually { await remuxContext.api.snapshot().playRequests.count == 2 })
        #expect(await remuxContext.api.snapshot().playRequests.last?.forceMode == "remux")

        let transcode = play(mode: "transcode", url: "/hls/t/index.m3u8", offset: 0, session: "t", start: 0)
        let transcodeContext = try await Context.make(defaultPlays: [FakePlayPlan(.value(direct)), FakePlayPlan(.value(transcode))])
        await transcodeContext.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))
        transcodeContext.mpv.emit(.failed(MPVFailure(code: -3, message: "video decode failed")))
        #expect(await eventually { await transcodeContext.api.snapshot().playRequests.count == 2 })
        #expect(await transcodeContext.api.snapshot().playRequests.last?.forceMode == "transcode")
    }

    @Test func endedEpisodeStartsCancelableEightSecondCountdown() async throws {
        let direct = play(mode: "direct", url: "/episode.mkv", offset: 0, session: nil, start: 0)
        let apiNext = card(43, "Épisode suivant")
        let context = try await Context.make(defaultPlays: [FakePlayPlan(.value(direct))], next: .value(apiNext))
        await context.coordinator.start(intent: PlayIntent(itemID: 42, fileID: 10, startPosition: 0, restart: false))

        context.mpv.emit(.ended)
        #expect(await eventually { context.coordinator.countdown == 8 })
        #expect(context.coordinator.nextEpisode?.id == 43)
        #expect(await eventually { await context.clock.pending.contains(.seconds(1)) })
        await context.clock.advance(.seconds(1))
        #expect(await eventually { context.coordinator.countdown == 7 })
        context.coordinator.cancelCountdown()
        #expect(context.coordinator.countdown == nil)
        #expect(context.coordinator.nextEpisode == nil)
    }

    private func play(mode: String, url: String, offset: Double, session: String?, start: Double, subtitles: [String] = []) -> PlayResponse {
        let sub = "[" + subtitles.joined(separator: ",") + "]"
        return decode(#"{"mode":"\#(mode)","url":"\#(url)","offset":\#(offset),"start":\#(start),"duration":7200,"itemId":42,"fileId":10,"sessionId":\#(session.map { "\"\($0)\"" } ?? "null"),"subtitles":\#(sub)}"#)
    }
}

@MainActor
private func eventually(_ condition: @escaping @MainActor () async -> Bool) async -> Bool {
    let clock = ContinuousClock()
    let deadline = clock.now + .milliseconds(200)
    while clock.now < deadline {
        if await condition() { return true }
        await Task.yield()
    }
    return await condition()
}

@MainActor
private struct Context {
    let server: ServerAddress
    let sessions: MemorySessionStore
    let api: FakeAuroraAPI
    let mpv: FakeMPVController
    let clock: ManualPlaybackClock
    let coordinator: PlaybackCoordinator

    static func make(
        plays: [Double: FakePlayPlan] = [:],
        defaultPlays: [FakePlayPlan] = [],
        next: FakeOutcome<Card?> = .value(nil),
        preferences: Preferences = Preferences()
    ) async throws -> Context {
        let server = try ServerAddress.normalize("nas")
        let sessions = MemorySessionStore()
        try await sessions.setToken("token", for: server.origin)
        let api = FakeAuroraAPI(server: server, plays: plays, defaultPlays: defaultPlays, next: next)
        let mpv = FakeMPVController()
        let clock = ManualPlaybackClock()
        let coordinator = PlaybackCoordinator(
            api: api,
            mpv: mpv,
            sessions: sessions,
            server: server,
            capabilities: PlaybackCapabilities.detect { _ in true },
            preferences: preferences,
            clock: clock
        )
        return Context(server: server, sessions: sessions, api: api, mpv: mpv, clock: clock, coordinator: coordinator)
    }
}
