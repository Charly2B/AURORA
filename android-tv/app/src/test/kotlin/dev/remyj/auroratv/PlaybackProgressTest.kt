package dev.remyj.auroratv

import dev.remyj.auroratv.player.PlaybackProgressRegistry
import dev.remyj.auroratv.player.PlaybackSample
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test

@OptIn(kotlinx.coroutines.ExperimentalCoroutinesApi::class)
class PlaybackProgressTest {
    private fun sample(position: Double, duration: Double = 100.0) = PlaybackSample(7, position, duration)

    @Test fun thresholdIncludesEightyPercentAndStopsAllLaterProgress() = runTest {
        val events = mutableListOf<String>()
        val writer = PlaybackProgressRegistry(backgroundScope).begin(Any(), 1,
            report = { events += "progress:${it.position}" }, markWatched = { events += "watched" })
        writer.observe(sample(79.999)); writer.report(sample(79.999)); runCurrent()
        assertEquals(listOf("progress:79.999"), events)
        writer.observe(sample(80.0)); runCurrent()
        writer.report(sample(90.0)); writer.close(sample(20.0)); runCurrent()
        assertEquals(listOf("progress:79.999", "watched"), events)
    }

    @Test fun unknownDurationKeepsResumeButInvalidNumbersNeverComplete() = runTest {
        val events = mutableListOf<String>()
        val writer = PlaybackProgressRegistry(backgroundScope).begin(Any(), 1,
            report = { events += "progress:${it.duration}" }, markWatched = { events += "watched" })
        writer.report(sample(80.0, 0.0))
        writer.observe(sample(80.0, 0.0))
        writer.report(sample(80.0, Double.NaN))
        writer.observe(sample(Double.NaN))
        writer.observe(sample(80.0, Double.POSITIVE_INFINITY))
        runCurrent()
        assertEquals(listOf("progress:0.0"), events)
    }

    @Test fun completionWaitsForSlowProgressAndSuppressesQueuedOldSamples() = runTest {
        val release = CompletableDeferred<Unit>()
        val events = mutableListOf<String>()
        val writer = PlaybackProgressRegistry(backgroundScope).begin(Any(), 1,
            report = { events += "start"; release.await(); events += "end" },
            markWatched = { events += "watched" })
        writer.report(sample(10.0)); runCurrent()
        writer.report(sample(20.0)); writer.observe(sample(80.0)); writer.report(sample(30.0)); runCurrent()
        assertEquals(listOf("start"), events)
        release.complete(Unit); runCurrent()
        assertEquals(listOf("start", "end", "watched"), events)
    }

    @Test fun failedCompletionRetriesAfterDelayEvenAfterRewindAndClose() = runTest {
        var attempts = 0
        val progress = mutableListOf<Double>()
        val writer = PlaybackProgressRegistry(backgroundScope, retryDelayMs = 1_000).begin(Any(), 1,
            report = { progress += it.position }, markWatched = { if (++attempts < 3) error("offline") })
        writer.observe(sample(80.0)); runCurrent()
        writer.report(sample(20.0)); runCurrent()
        advanceTimeBy(999); runCurrent()
        assertEquals(1, attempts)
        writer.close(sample(10.0)); runCurrent()
        assertEquals(2, attempts)
        advanceTimeBy(1_000); runCurrent()
        assertEquals(3, attempts)
        assertEquals(emptyList<Double>(), progress)
    }

    @Test fun duplicateFinalSamplesMarkOnlyOnce() = runTest {
        var watched = 0
        val writer = PlaybackProgressRegistry(backgroundScope).begin(Any(), 1,
            report = { error("must not overwrite watched") }, markWatched = { watched++ })
        writer.close(sample(80.0)); writer.close(sample(99.0)); runCurrent()
        assertEquals(1, watched)
    }

    @Test fun nextEpisodeHasItsOwnProgressAndThreshold() = runTest {
        val events = mutableListOf<String>()
        val owner = Any()
        val registry = PlaybackProgressRegistry(backgroundScope)
        registry.begin(owner, 1, report = { events += "episode1-progress" }, markWatched = { events += "episode1-watched" })
            .close(sample(80.0))
        val next = registry.begin(owner, 2, report = { events += "episode2-progress" }, markWatched = { events += "episode2-watched" })
        next.report(sample(10.0)); runCurrent()
        assertEquals(listOf("episode1-watched", "episode2-progress"), events)
        next.observe(sample(80.0)); runCurrent()
        assertEquals(listOf("episode1-watched", "episode2-progress", "episode2-watched"), events)
    }

    @Test fun newPlaybackOfSameItemInvalidatesOldCompletionRetry() = runTest {
        var oldAttempts = 0
        val events = mutableListOf<String>()
        val owner = Any()
        val registry = PlaybackProgressRegistry(backgroundScope, retryDelayMs = 1_000)
        val old = registry.begin(owner, 1, report = {}, markWatched = { oldAttempts++; error("offline") })
        old.observe(sample(80.0)); runCurrent()
        val current = registry.begin(owner, 1, report = { events += "new-progress" }, markWatched = { events += "new-watched" })
        current.report(sample(10.0)); runCurrent()
        advanceTimeBy(1_000); runCurrent()
        assertEquals(1, oldAttempts)
        assertEquals(listOf("new-progress"), events)
        current.observe(sample(80.0)); runCurrent()
        assertEquals(listOf("new-progress", "new-watched"), events)
    }

    @Test fun failedFinalProgressIsRetriedWithoutLosingItsSnapshot() = runTest {
        val positions = mutableListOf<Double>()
        val writer = PlaybackProgressRegistry(backgroundScope, retryDelayMs = 1_000).begin(Any(), 1,
            report = { positions += it.position; if (positions.size == 1) error("offline") }, markWatched = {})
        writer.close(sample(25.0)); runCurrent()
        advanceTimeBy(1_000); runCurrent()
        assertEquals(listOf(25.0, 25.0), positions)
    }
}
