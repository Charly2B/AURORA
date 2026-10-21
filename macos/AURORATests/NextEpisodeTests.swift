import Testing
@testable import AURORA

@Suite("Épisode suivant")
struct NextEpisodeTests {
    @Test func countdownRunsEightToZeroThenRequestsReplacement() {
        var state = NextEpisodeCountdown(remaining: 8)
        var shouldReplace = false
        for _ in 0..<8 { shouldReplace = state.tick() }
        #expect(state.remaining == 0)
        #expect(shouldReplace)
    }

    @Test func cancelPreventsReplacement() {
        var state = NextEpisodeCountdown(remaining: 8)
        state.cancel()
        let shouldReplace = state.tick()
        #expect(!shouldReplace)
        #expect(state.remaining == nil)
    }
}
