package dev.remyj.auroratv.player

import java.util.WeakHashMap
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/** Instantané capturé sur le thread du lecteur, jamais une position de saut en attente. */
data class PlaybackSample(val fileId: Long, val position: Double, val duration: Double) {
    val valid: Boolean get() = position.isFinite() && position > 0 && duration.isFinite() && duration >= 0
    val watched: Boolean get() = valid && duration > 0 && position >= duration * 0.8
}

internal class PlaybackGate {
    val mutex = Mutex()
    @Volatile var generation = 0L
}

/** Même élément/client : les écritures d'une nouvelle lecture passent après celles déjà en vol. */
class PlaybackProgressRegistry(
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.IO),
    private val retryDelayMs: Long = 5_000,
) {
    private val clients = WeakHashMap<Any, MutableMap<Long, PlaybackGate>>()

    fun begin(client: Any, itemId: Long, report: suspend (PlaybackSample) -> Unit, markWatched: suspend () -> Unit): PlaybackProgress =
        synchronized(clients) {
            val gate = clients.getOrPut(client) { mutableMapOf() }.getOrPut(itemId) { PlaybackGate() }
            PlaybackProgress(scope, gate, ++gate.generation, retryDelayMs, report, markWatched)
        }

    companion object {
        val shared = PlaybackProgressRegistry()
    }
}

/** Après le seuil, l'intention de marquage est irréversible pour cette lecture, même après un recul. */
class PlaybackProgress internal constructor(
    private val scope: CoroutineScope,
    private val gate: PlaybackGate,
    private val generation: Long,
    private val retryDelayMs: Long,
    private val writeProgress: suspend (PlaybackSample) -> Unit,
    private val markWatched: suspend () -> Unit,
) {
    private data class Command(val sample: PlaybackSample? = null, val closing: Boolean = false, val retry: Boolean = false)
    private val commands = Channel<Command>(Channel.UNLIMITED)
    private val inputLock = Any()
    @Volatile private var completionPending = false
    private var closed = false
    private val current: Boolean get() = gate.generation == generation

    init {
        scope.launch {
            var closing = false
            var latest: PlaybackSample? = null
            var reported: PlaybackSample? = null
            var completed = false
            var retryJob: Job? = null
            try {
                for (command in commands) {
                    if (!current) break
                    command.sample?.let { latest = it }
                    closing = closing || command.closing
                    if (command.retry) retryJob = null
                    // Un échantillon par seconde ne doit pas transformer une panne en boucle de requêtes.
                    if (retryJob?.isActive == true && !command.closing) continue
                    if (command.closing) { retryJob?.cancel(); retryJob = null }
                    try {
                        gate.mutex.withLock {
                            if (!current) return@withLock
                            if (completionPending) {
                                markWatched()
                                completed = true
                            } else {
                                val sample = latest
                                if (sample != null && sample != reported) {
                                    writeProgress(sample)
                                    reported = sample
                                }
                            }
                        }
                        if (!current || completed || closing) break
                    } catch (e: CancellationException) {
                        throw e
                    } catch (_: Exception) {
                        retryJob = scope.launch {
                            delay(retryDelayMs)
                            commands.trySend(Command(retry = true))
                        }
                    }
                }
            } finally {
                retryJob?.cancel()
                commands.close()
            }
        }
    }

    /** Observation légère chaque seconde ; seule la première traversée du seuil réveille l'écriture. */
    fun observe(sample: PlaybackSample) = synchronized(inputLock) {
        if (closed || !current || !sample.watched || completionPending) return@synchronized
        completionPending = true
        commands.trySend(Command())
        Unit
    }

    fun report(sample: PlaybackSample) = synchronized(inputLock) {
        if (closed || !current || !sample.valid) return@synchronized
        if (sample.watched) completionPending = true
        commands.trySend(Command(sample))
        Unit
    }

    /** Le scope réseau survit à Compose ; deux fermetures ne produisent pas deux écritures finales. */
    fun close(sample: PlaybackSample?) = synchronized(inputLock) {
        if (closed || !current) return@synchronized
        closed = true
        val valid = sample?.takeIf { it.valid }
        if (valid?.watched == true) completionPending = true
        commands.trySend(Command(valid, closing = true))
        Unit
    }
}
