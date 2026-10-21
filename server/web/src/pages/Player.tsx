import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import Hls from 'hls.js'
import {
  ArrowLeft, Captions, ChevronsRight, Gauge, ListVideo, Loader2, Maximize, Minimize, Pause, Play,
  RotateCcw, RotateCw, SkipForward, Volume1, Volume2, VolumeX, X,
} from 'lucide-react'
import { api, ApiError } from '../lib/api'
import { detectCaps } from '../lib/caps'
import { audioLabel, clock, episodeCode, sameLanguage } from '../lib/format'
import { useDocumentTitle } from '../lib/hooks'
import { useSession } from '../lib/session'
import type { Card, Detail, PlayResponse, Prefs } from '../lib/types'
import { Button, cx } from '../components/ui'

/*
 * Lecteur AURORA.
 * - Direct Play : <video src> sur le fichier original (requêtes Range).
 * - Remux / conversion : HLS fMP4 produit à la volée par ffmpeg, lu par hls.js (ou nativement par Safari).
 *   Une session HLS commence à un « offset » : position absolue = offset + currentTime.
 *   Une recherche hors de la partie déjà produite relance une session à la nouvelle position.
 */

const SAVE_EVERY_MS = 10_000
const SPEEDS = [0.75, 1, 1.25, 1.5, 2]

interface Choice { audioIndex: number; subtitleIndex: number } // -1 = défaut / aucun

export function initialChoice(file: Detail['files'][number], prefs: Prefs): Choice {
  let audioIndex = -1
  if (prefs.audioLang) {
    const a = file.audio.find((x) => sameLanguage(x.language, prefs.audioLang))
    if (a) audioIndex = a.index
  }
  const chosenAudio = file.audio.find((a) => a.index === audioIndex) ?? file.audio.find((a) => a.default) ?? file.audio[0]
  const subLang = prefs.subLang || prefs.audioLang
  let subtitleIndex = -1
  const textSubs = file.subtitles.filter((s) => s.text && sameLanguage(s.language, subLang))
  if (prefs.subMode === 'always') {
    subtitleIndex = (textSubs.find((s) => !s.forced) ?? textSubs[0])?.index ?? -1
  } else if (prefs.subMode === 'auto' && subLang) {
    if (chosenAudio && chosenAudio.language && !sameLanguage(chosenAudio.language, subLang)) {
      subtitleIndex = (textSubs.find((s) => !s.forced) ?? textSubs[0])?.index ?? -1
    } else {
      subtitleIndex = textSubs.find((s) => s.forced)?.index ?? -1
    }
  }
  return { audioIndex, subtitleIndex }
}

export default function Player() {
  const { id } = useParams()
  const [params] = useSearchParams()
  const nav = useNavigate()
  const { user } = useSession()
  const prefs = user!.prefs

  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const wrapRef = useRef<HTMLDivElement>(null)
  const [item, setItem] = useState<Detail>()
  const [play, setPlay] = useState<PlayResponse>()
  const [choice, setChoice] = useState<Choice>({ audioIndex: -1, subtitleIndex: -1 })
  const [fatal, setFatal] = useState<string>()
  const [notice, setNotice] = useState<string>()
  const [loading, setLoading] = useState(true)
  const [playing, setPlaying] = useState(false)
  const [time, setTime] = useState(0) // position absolue (s)
  const [buffered, setBuffered] = useState(0)
  const [volume, setVolume] = useState(() => Number(localStorage.getItem('aurora.volume') ?? 1))
  const [muted, setMuted] = useState(false)
  const [speed, setSpeed] = useState(1)
  const [fullscreen, setFullscreen] = useState(false)
  const [controls, setControls] = useState(true)
  const [panel, setPanel] = useState<'tracks' | 'speed' | 'chapters' | null>(null)
  const [next, setNext] = useState<Card | null>(null)
  const [countdown, setCountdown] = useState<number | null>(null)
  const [ended, setEnded] = useState(false)

  const nextLoaded = useRef(false)
  const playRef = useRef<PlayResponse>(undefined)
  playRef.current = play
  const retries = useRef(0)
  const lastSaved = useRef(0)
  const pendingSeek = useRef<number | null>(null)
  const caps = useMemo(() => detectCaps(), [])
  const fileId = useMemo(() => {
    const p = Number(params.get('file'))
    if (!item) return undefined
    if (p && item.files.some((f) => f.id === p)) return p
    if (item.state.lastFileId && item.files.some((f) => f.id === item.state.lastFileId)) return item.state.lastFileId
    return item.files[0]?.id
  }, [item, params])
  const file = item?.files.find((f) => f.id === fileId)
  const duration = play?.duration || file?.duration || 0
  const abs = useCallback(() => (playRef.current?.offset ?? 0) + (videoRef.current?.currentTime ?? 0), [])

  useDocumentTitle(item ? (item.kind === 'episode' ? `${item.seriesTitle} ${episodeCode(item.season, item.episode)}` : item.title) : 'Lecture')

  // 1. Fiche de l'œuvre.
  useEffect(() => {
    setItem(undefined); setPlay(undefined); setFatal(undefined); setEnded(false); setNext(null); setCountdown(null)
    nextLoaded.current = false
    retries.current = 0
    api.get<Detail>(`/items/${id}`).then(setItem).catch((e) => setFatal(e.message))
  }, [id])

  // 2. Démarrage : choix des pistes selon les préférences, position de reprise.
  const startAt = useRef<number>(0)
  useEffect(() => {
    if (!item || !file) {
      if (item && !file) setFatal("Aucun fichier vidéo disponible pour cette œuvre.")
      return
    }
    const restart = params.get('restart') === '1'
    startAt.current = !restart && !item.state.watched && item.state.position > 0 ? item.state.position : 0
    const c = initialChoice(file, prefs)
    setChoice(c)
    requestPlay(startAt.current, c)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [item?.id, fileId])

  const stopSession = useCallback((p?: PlayResponse) => {
    if (p?.sessionId) fetch(`/api/v1/hls/${p.sessionId}`, { method: 'DELETE', headers: { 'X-Aurora': '1' }, keepalive: true }).catch(() => {})
  }, [])

  /** Demande (ou redemande) une lecture au serveur à la position absolue `start`. */
  const requestPlay = useCallback(async (start: number, c: Choice, forceMode?: 'remux') => {
    if (!fileId) return
    setLoading(true)
    setNotice(undefined)
    // La piste de sous-titres n'est transmise que si c'est une image (incrustation côté serveur).
    const sub = file?.subtitles.find((s) => s.index === c.subtitleIndex)
    try {
      const r = await api.post<PlayResponse>('/play', {
        fileId, caps, start, forceMode,
        audioIndex: c.audioIndex >= 0 ? c.audioIndex : undefined,
        subtitleIndex: sub && !sub.text ? sub.index : undefined,
      })
      setPlay((old) => { if (old?.sessionId && old.sessionId !== r.sessionId) stopSession(old); return r })
      pendingSeek.current = r.mode === 'direct' ? start : null
    } catch (e) {
      setLoading(false)
      const msg = e instanceof ApiError ? e.message : String(e)
      if (playRef.current) setNotice(msg) // une lecture est déjà en cours : on la garde
      else setFatal(msg)
    }
  }, [fileId, file, caps, stopSession])

  // 3. Branchement de la source vidéo.
  useEffect(() => {
    const v = videoRef.current
    if (!v || !play) return
    hlsRef.current?.destroy()
    hlsRef.current = null
    const wasPlaying = true
    if (play.mode === 'direct' || caps.hls === 'native' || !Hls.isSupported()) {
      v.src = play.url
    } else {
      const hls = new Hls({ startPosition: 0, maxBufferLength: 45, backBufferLength: 60, enableWorker: true, lowLatencyMode: false,
        manifestLoadingTimeOut: 60_000, manifestLoadingMaxRetry: 2, fragLoadingMaxRetry: 6 })
      hlsRef.current = hls
      hls.on(Hls.Events.ERROR, (_, data) => {
        if (!data.fatal) return
        if (data.type === Hls.ErrorTypes.MEDIA_ERROR && retries.current < 1) {
          retries.current++
          hls.recoverMediaError()
          return
        }
        recover(data.type === Hls.ErrorTypes.NETWORK_ERROR ? 'network' : 'media', data.details)
      })
      hls.loadSource(play.url)
      hls.attachMedia(v)
    }
    v.playbackRate = speed
    if (wasPlaying) v.play().catch(() => { /* lecture automatique bloquée : l'utilisateur appuiera sur Lecture */ })
    return () => { hlsRef.current?.destroy(); hlsRef.current = null }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [play])

  // Sous-titre textuel : une seule piste <track> à la fois, recalée sur l'offset de la session.
  const subtitle = play?.subtitles.find((s) => s.index === choice.subtitleIndex && s.text)
  useEffect(() => {
    const v = videoRef.current
    if (!v) return
    for (const t of Array.from(v.textTracks)) t.mode = t.label === 'aurora' ? 'showing' : 'disabled'
  }, [subtitle?.url, play])

  // Fermeture : position enregistrée, session arrêtée.
  const saveProgress = useCallback((beacon = false) => {
    const p = playRef.current
    const v = videoRef.current
    if (!p || !v || !item) return
    const position = abs()
    if (!isFinite(position) || position <= 0) return
    const body = { fileId: p.fileId, position, duration: p.duration || v.duration || 0 }
    lastSaved.current = Date.now()
    if (beacon) api.beacon(`/items/${item.id}/progress`, body)
    else api.post(`/items/${item.id}/progress`, body).catch(() => {})
  }, [item, abs])
  useEffect(() => {
    const onHide = () => saveProgress(true)
    window.addEventListener('pagehide', onHide)
    return () => {
      window.removeEventListener('pagehide', onHide)
      saveProgress(true)
      stopSession(playRef.current)
    }
  }, [saveProgress, stopSession])

  /** Reprise après erreur : on relance à la position courante, trois fois au plus. */
  const recover = useCallback((kind: 'network' | 'media', detail?: string) => {
    const p = playRef.current
    const t = abs()
    if (retries.current >= 3) {
      setLoading(false)
      setFatal(kind === 'network'
        ? 'La connexion au NAS a été interrompue et n\'a pas pu être rétablie. Vérifiez le réseau puis réessayez.'
        : `La vidéo ne peut pas être décodée par ce navigateur${detail ? ` (${detail})` : ''}.`)
      return
    }
    retries.current++
    setNotice(kind === 'network' ? 'Connexion interrompue, reprise en cours…' : 'Problème de lecture, nouvelle tentative…')
    // Un échec de décodage en lecture directe : on bascule vers le remux, plus tolérant.
    const force = kind === 'media' && p?.mode === 'direct' ? 'remux' : undefined
    setTimeout(() => requestPlay(t, choiceRef.current, force), kind === 'network' ? 1500 * retries.current : 200)
  }, [abs, requestPlay])
  const choiceRef = useRef(choice)
  choiceRef.current = choice

  // Événements vidéo.
  const onLoadedMetadata = () => {
    const v = videoRef.current!
    if (pendingSeek.current !== null) {
      if (pendingSeek.current > 0) v.currentTime = pendingSeek.current
      pendingSeek.current = null
    } else if (play && play.mode !== 'direct' && v.currentTime > 1) {
      v.currentTime = 0 // Safari démarre les playlists « EVENT » au direct : on revient au début de la session
    }
    v.volume = volume
    v.muted = muted
  }
  const onTimeUpdate = () => {
    const v = videoRef.current!
    setTime(abs())
    if (v.buffered.length) setBuffered((playRef.current?.offset ?? 0) + v.buffered.end(v.buffered.length - 1))
    if (!v.paused && Date.now() - lastSaved.current > SAVE_EVERY_MS) saveProgress()
    if (retries.current && v.currentTime > 0 && !v.paused) { retries.current = 0; setNotice(undefined) }
    if (item?.kind === 'episode' && duration && duration - abs() < 30) loadNext()
  }
  const onError = () => {
    const v = videoRef.current!
    if (hlsRef.current) return // erreurs gérées par hls.js
    const code = v.error?.code
    recover(code === MediaError.MEDIA_ERR_NETWORK ? 'network' : 'media', v.error?.message)
  }

  const loadNext = useCallback(async () => {
    if (!item || item.kind !== 'episode' || nextLoaded.current) return
    nextLoaded.current = true // un seul appel par épisode
    try {
      const r = await api.get<{ next: Card | null }>(`/items/${item.id}/next`)
      setNext(r.next)
    } catch {
      setNext(null)
    }
  }, [item])

  const onEnded = () => {
    setEnded(true)
    setPlaying(false)
    saveProgress()
    if (item?.kind === 'episode') {
      loadNext()
      if (prefs.autoplayNext) setCountdown(8)
    }
  }
  useEffect(() => {
    if (countdown === null) return
    if (!next) return
    if (countdown <= 0) { nav(`/play/${next.id}`, { replace: true }); return }
    const t = setTimeout(() => setCountdown((c) => (c === null ? null : c - 1)), 1000)
    return () => clearTimeout(t)
  }, [countdown, next, nav])

  // Commandes.
  const toggle = useCallback(() => {
    const v = videoRef.current
    if (!v) return
    if (ended) { seek(0); setEnded(false); return }
    if (v.paused) v.play().catch(() => {})
    else { v.pause(); saveProgress() }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ended, saveProgress])

  const seek = useCallback((target: number) => {
    const v = videoRef.current
    const p = playRef.current
    if (!v || !p) return
    target = Math.max(0, Math.min(target, (p.duration || v.duration || target) - 0.5))
    setTime(target)
    setEnded(false)
    setCountdown(null)
    if (p.mode === 'direct') { v.currentTime = target; return }
    const local = target - p.offset
    const seekableEnd = v.seekable.length ? v.seekable.end(v.seekable.length - 1) : 0
    // Le serveur efface les segments déjà vus (au-delà d'environ 2 min derrière la lecture) :
    // un retour arrière de plus d'une minute ouvre directement une nouvelle session.
    const farBack = target < abs() - 60
    if (local >= 0 && local <= seekableEnd - 1 && !farBack) v.currentTime = local
    else requestPlay(target, choiceRef.current) // hors de la partie disponible : nouvelle session
  }, [requestPlay, abs])

  const changeTracks = (c: Choice) => {
    const prevSub = file?.subtitles.find((s) => s.index === choice.subtitleIndex)
    const nextSub = file?.subtitles.find((s) => s.index === c.subtitleIndex)
    setChoice(c)
    setPanel(null)
    // Changer d'audio, ou passer par un sous-titre image (incrustation), demande un nouveau flux.
    const needsRestart = c.audioIndex !== choice.audioIndex || (!!nextSub && !nextSub.text) || (!!prevSub && !prevSub.text)
    if (needsRestart) requestPlay(abs(), c)
  }

  const toggleFullscreen = useCallback(() => {
    const el = wrapRef.current as (HTMLDivElement & { webkitRequestFullscreen?: () => void }) | null
    const v = videoRef.current as (HTMLVideoElement & { webkitEnterFullscreen?: () => void }) | null
    if (document.fullscreenElement) document.exitFullscreen()
    else if (el?.requestFullscreen) el.requestFullscreen().catch(() => {})
    else if (el?.webkitRequestFullscreen) el.webkitRequestFullscreen()
    else v?.webkitEnterFullscreen?.() // iPhone : plein écran natif uniquement
  }, [])
  useEffect(() => {
    const on = () => setFullscreen(!!document.fullscreenElement)
    document.addEventListener('fullscreenchange', on)
    return () => document.removeEventListener('fullscreenchange', on)
  }, [])

  const setVol = (val: number) => {
    const v = videoRef.current
    const x = Math.max(0, Math.min(1, val))
    setVolume(x)
    setMuted(x === 0)
    if (v) { v.volume = x; v.muted = x === 0 }
    try { localStorage.setItem('aurora.volume', String(x)) } catch { /* stockage indisponible */ }
  }
  const toggleMute = () => { const v = videoRef.current; if (v) { v.muted = !muted; setMuted(!muted) } }

  const back = () => { saveProgress(); nav(item?.kind === 'episode' ? `/series/${item.seriesId}?season=${item.season}&episode=${item.id}` : item ? `/movies/${item.id}` : '/') }

  // Raccourcis clavier.
  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      if ((e.target as HTMLElement).tagName === 'INPUT' || e.metaKey || e.ctrlKey || e.altKey) return
      const k = e.key
      if (k === ' ' || k === 'k') { e.preventDefault(); toggle() }
      else if (k === 'ArrowLeft' || k === 'j') { e.preventDefault(); seek(abs() - 10) }
      else if (k === 'ArrowRight' || k === 'l') { e.preventDefault(); seek(abs() + 10) }
      else if (k === 'ArrowUp') { e.preventDefault(); setVol(volume + 0.1) }
      else if (k === 'ArrowDown') { e.preventDefault(); setVol(volume - 0.1) }
      else if (k === 'm') toggleMute()
      else if (k === 'f') toggleFullscreen()
      else if (k === 'c' && play) {
        const text = play.subtitles.filter((s) => s.text)
        const i = text.findIndex((s) => s.index === choice.subtitleIndex)
        const nx = i + 1 < text.length ? text[i + 1].index : -1
        setChoice({ ...choice, subtitleIndex: nx })
      } else if (k === 'Escape' && !document.fullscreenElement) {
        if (panel) setPanel(null)
        else back()
      } else return
      poke()
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  })

  // Masquage automatique des commandes pendant la lecture.
  const hideTimer = useRef<ReturnType<typeof setTimeout>>(undefined)
  const poke = useCallback(() => {
    setControls(true)
    clearTimeout(hideTimer.current)
    hideTimer.current = setTimeout(() => setControls(false), 3200)
  }, [])
  useEffect(() => { if (!playing || panel) { setControls(true); clearTimeout(hideTimer.current) } else poke() }, [playing, panel, poke])

  // Les répliques remontent au-dessus de la barre de commandes quand elle est affichée.
  useEffect(() => {
    const v = videoRef.current
    if (!v) return
    for (const t of Array.from(v.textTracks)) {
      for (const c of Array.from(t.cues ?? [])) (c as VTTCue).line = vis ? -5 : 'auto'
    }
  })

  const chapters = play?.chapters ?? []
  const chapterAt = chapters.filter((c) => c.start <= time).pop()
  const vis = controls || !playing || !!panel || ended

  if (fatal) {
    return (
      <div className="fixed inset-0 bg-black grid place-items-center p-6">
        <div className="max-w-lg text-center space-y-6">
          <p className="font-display text-3xl">Lecture impossible</p>
          <p className="text-mist leading-relaxed">{fatal}</p>
          {play && <p className="text-xs text-ash">Mode tenté : {play.mode} — {play.reason}</p>}
          <div className="flex justify-center gap-3">
            <Button onClick={back} icon={<ArrowLeft className="size-4" />}>Retour</Button>
            <Button variant="primary" onClick={() => { retries.current = 0; setFatal(undefined); requestPlay(abs() || startAt.current, choice) }}>Réessayer</Button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div ref={wrapRef} data-sub-size={prefs.subSize} onMouseMove={poke} onTouchStart={poke}
      className={cx('fixed inset-0 bg-black select-none', !vis && 'cursor-none')}>
      <video ref={videoRef} className="absolute inset-0 size-full" playsInline preload="auto"
        onClick={() => { if (panel) setPanel(null); else if (!vis) poke(); else toggle() }} onDoubleClick={toggleFullscreen}
        onLoadedMetadata={onLoadedMetadata} onTimeUpdate={onTimeUpdate} onError={onError} onEnded={onEnded}
        onPlay={() => { setPlaying(true); setEnded(false) }} onPause={() => setPlaying(false)}
        onWaiting={() => setLoading(true)} onPlaying={() => setLoading(false)} onCanPlay={() => setLoading(false)}
        onSeeked={() => setTime(abs())}>
        {subtitle?.url && <track key={subtitle.url} kind="subtitles" label="aurora" srcLang={subtitle.language || 'und'} src={subtitle.url} default />}
      </video>

      {loading && !ended && (
        <div className="absolute inset-0 grid place-items-center pointer-events-none">
          <Loader2 className="size-12 animate-spin text-paper/80" aria-label="Chargement" />
        </div>
      )}
      {notice && (
        <div role="status" className="absolute top-20 left-1/2 -translate-x-1/2 rounded-full bg-coal/90 px-5 py-2.5 text-sm ring-1 ring-white/10 flex items-center gap-3">
          {notice}
          <button aria-label="Fermer" onClick={() => setNotice(undefined)}><X className="size-4 text-ash" /></button>
        </div>
      )}

      {/* Barre supérieure */}
      <div className={cx('absolute inset-x-0 top-0 p-4 sm:p-6 bg-gradient-to-b from-black/80 to-transparent flex items-center gap-4 transition-opacity duration-300', vis ? 'opacity-100' : 'opacity-0 pointer-events-none')}>
        <button onClick={back} aria-label="Retour" className="grid size-11 place-items-center rounded-full hover:bg-white/10"><ArrowLeft className="size-6" /></button>
        <div className="min-w-0">
          {item?.kind === 'episode' && <div className="text-xs text-aurora font-semibold tracking-wide">{item.seriesTitle} · {episodeCode(item.season, item.episode, item.episodeEnd)}</div>}
          <div className="font-display text-xl sm:text-2xl truncate">{item?.title}</div>
          {chapterAt?.title && <div className="text-xs text-ash truncate">{chapterAt.title}</div>}
        </div>
      </div>

      {/* Fin d'épisode : suivant */}
      {item?.kind === 'episode' && next && (ended || (duration > 0 && duration - time < 25)) && (
        <div className="absolute right-4 sm:right-8 bottom-32 w-[min(92vw,360px)] rounded-2xl bg-coal/95 ring-1 ring-white/10 p-4 shadow-[var(--shadow-lift)] rise">
          <div className="eyebrow mb-2">Épisode suivant{countdown !== null ? ` dans ${countdown} s` : ''}</div>
          <div className="flex gap-3">
            {next.thumb && <img src={next.thumb} alt="" className="w-28 aspect-video object-cover rounded-md" />}
            <div className="min-w-0">
              <div className="text-xs text-ash">{episodeCode(next.season, next.episode, next.episodeEnd)}</div>
              <div className="font-medium leading-snug line-clamp-2">{next.title}</div>
            </div>
          </div>
          <div className="flex gap-2 mt-4">
            <Button variant="primary" size="sm" icon={<SkipForward className="size-4" />} onClick={() => nav(`/play/${next.id}`, { replace: true })}>Lire</Button>
            {countdown !== null && <Button size="sm" onClick={() => setCountdown(null)}>Annuler</Button>}
          </div>
        </div>
      )}
      {ended && item?.kind !== 'episode' && (
        <div className="absolute inset-0 grid place-items-center bg-black/60">
          <div className="text-center space-y-5">
            <p className="font-display text-4xl">Fin</p>
            <div className="flex gap-3 justify-center">
              <Button onClick={back} icon={<ArrowLeft className="size-4" />}>Retour à la fiche</Button>
              <Button variant="primary" onClick={() => seek(0)} icon={<RotateCcw className="size-4" />}>Revoir</Button>
            </div>
          </div>
        </div>
      )}

      {/* Commandes inférieures */}
      <div className={cx('absolute inset-x-0 bottom-0 px-4 sm:px-8 pb-4 sm:pb-6 pt-24 bg-gradient-to-t from-black/90 via-black/50 to-transparent transition-opacity duration-300', vis ? 'opacity-100' : 'opacity-0 pointer-events-none')}>
        <SeekBar time={time} duration={duration} buffered={buffered} chapters={chapters} onSeek={seek} />
        <div className="flex items-center gap-1 sm:gap-2 mt-2">
          <Ctl label={playing ? 'Pause (espace)' : 'Lecture (espace)'} onClick={toggle} big>
            {playing ? <Pause className="size-7 fill-current" /> : <Play className="size-7 fill-current" />}
          </Ctl>
          <Ctl label="Reculer de 10 s (←)" onClick={() => seek(abs() - 10)}><RotateCcw className="size-5" /></Ctl>
          <Ctl label="Avancer de 10 s (→)" onClick={() => seek(abs() + 10)}><RotateCw className="size-5" /></Ctl>
          <div className="hidden sm:flex items-center gap-1 group/vol">
            <Ctl label="Couper le son (m)" onClick={toggleMute}>
              {muted || volume === 0 ? <VolumeX className="size-5" /> : volume < 0.5 ? <Volume1 className="size-5" /> : <Volume2 className="size-5" />}
            </Ctl>
            <input type="range" min={0} max={1} step={0.05} value={muted ? 0 : volume} onChange={(e) => setVol(Number(e.target.value))}
              aria-label="Volume" className="w-0 opacity-0 group-hover/vol:w-24 group-hover/vol:opacity-100 focus:w-24 focus:opacity-100 transition-[width,opacity] duration-200" />
          </div>
          <span className="ml-2 text-sm tabular-nums text-mist">{clock(time)} <span className="text-ash">/ {clock(duration)}</span></span>
          <div className="ml-auto flex items-center gap-1 sm:gap-2">
            {item?.kind === 'episode' && (
              <Ctl label="Épisode suivant" onClick={async () => {
                let n = next
                if (!n) { try { n = (await api.get<{ next: Card | null }>(`/items/${item.id}/next`)).next } catch { n = null } }
                if (n) nav(`/play/${n.id}`, { replace: true })
                else setNotice("C'est le dernier épisode disponible.")
              }}><ChevronsRight className="size-5" /></Ctl>
            )}
            {chapters.length > 1 && <Ctl label="Chapitres" onClick={() => setPanel(panel === 'chapters' ? null : 'chapters')} active={panel === 'chapters'}><ListVideo className="size-5" /></Ctl>}
            <Ctl label="Audio et sous-titres" onClick={() => setPanel(panel === 'tracks' ? null : 'tracks')} active={panel === 'tracks'}><Captions className="size-5" /></Ctl>
            <Ctl label="Vitesse" onClick={() => setPanel(panel === 'speed' ? null : 'speed')} active={panel === 'speed'}><Gauge className="size-5" /></Ctl>
            <Ctl label="Plein écran (f)" onClick={toggleFullscreen}>{fullscreen ? <Minimize className="size-5" /> : <Maximize className="size-5" />}</Ctl>
          </div>
        </div>
      </div>

      {/* Panneaux */}
      {panel && (
        <div className="absolute right-4 sm:right-8 bottom-28 w-[min(92vw,420px)] max-h-[60vh] overflow-y-auto rounded-2xl bg-coal/95 backdrop-blur ring-1 ring-white/10 p-4 text-sm shadow-[var(--shadow-lift)] rise">
          {panel === 'tracks' && play && file && (
            <div className="space-y-5">
              <div>
                <div className="eyebrow mb-2">Audio</div>
                {play.audio.length === 0 && <p className="text-ash">Aucune piste audio.</p>}
                {play.audio.map((a) => {
                  const active = (choice.audioIndex >= 0 ? choice.audioIndex : (play.audio.find((x) => x.stream === play.audioStream)?.index ?? -1)) === a.index
                  return <Opt key={a.index} active={active} onClick={() => changeTracks({ ...choice, audioIndex: a.index })}>{audioLabel(a)}</Opt>
                })}
              </div>
              <div>
                <div className="eyebrow mb-2">Sous-titres</div>
                <Opt active={choice.subtitleIndex === -1} onClick={() => changeTracks({ ...choice, subtitleIndex: -1 })}>Aucun</Opt>
                {play.subtitles.map((s) => (
                  <Opt key={s.index} active={choice.subtitleIndex === s.index} onClick={() => changeTracks({ ...choice, subtitleIndex: s.index })}>
                    {s.label}{!s.text && <span className="block text-[11px] text-ember">Incrustation : demande une conversion vidéo au NAS</span>}
                  </Opt>
                ))}
                {subtitle && file.subtitles.find((s) => s.index === subtitle.index)?.codec.match(/ass|ssa/) && (
                  <p className="text-[11px] text-ash mt-2">Les styles ASS/SSA (couleurs, positions) ne sont pas reproduits dans le navigateur : texte seul.</p>
                )}
              </div>
              <p className="text-[11px] text-ash border-t border-white/5 pt-3">
                {play.reason}
              </p>
            </div>
          )}
          {panel === 'speed' && (
            <div>
              <div className="eyebrow mb-2">Vitesse de lecture</div>
              {SPEEDS.map((s) => (
                <Opt key={s} active={speed === s} onClick={() => { setSpeed(s); if (videoRef.current) videoRef.current.playbackRate = s; setPanel(null) }}>
                  {s === 1 ? 'Normale' : `${String(s).replace('.', ',')}×`}
                </Opt>
              ))}
            </div>
          )}
          {panel === 'chapters' && (
            <div>
              <div className="eyebrow mb-2">Chapitres</div>
              {chapters.map((c, i) => (
                <Opt key={i} active={chapterAt === c} onClick={() => { seek(c.start); setPanel(null) }}>
                  <span className="tabular-nums text-ash mr-3">{clock(c.start)}</span>{c.title || `Chapitre ${i + 1}`}
                </Opt>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function Ctl({ label, onClick, children, big, active }: { label: string; onClick: () => void; children: React.ReactNode; big?: boolean; active?: boolean }) {
  return (
    <button aria-label={label} title={label} onClick={onClick}
      className={cx('grid place-items-center rounded-full transition-colors hover:bg-white/10', big ? 'size-12' : 'size-10', active ? 'text-aurora' : 'text-paper')}>
      {children}
    </button>
  )
}

function Opt({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button onClick={onClick} aria-pressed={active}
      className={cx('w-full text-left px-3 py-2 rounded-lg flex items-start gap-2', active ? 'bg-aurora/[0.12] text-paper' : 'text-mist hover:bg-white/[0.06]')}>
      <span className={cx('mt-1.5 size-1.5 rounded-full shrink-0', active ? 'bg-aurora' : 'bg-transparent')} />
      <span>{children}</span>
    </button>
  )
}

/** Barre de progression : tampon, chapitres, aperçu du temps au survol, accessible au clavier. */
function SeekBar({ time, duration, buffered, chapters, onSeek }: {
  time: number; duration: number; buffered: number; chapters: { start: number; title: string }[]; onSeek: (t: number) => void
}) {
  const ref = useRef<HTMLDivElement>(null)
  const [hover, setHover] = useState<number | null>(null)
  const [drag, setDrag] = useState<number | null>(null)
  const at = (clientX: number) => {
    const r = ref.current!.getBoundingClientRect()
    return Math.max(0, Math.min(1, (clientX - r.left) / r.width)) * duration
  }
  const shown = drag ?? time
  const pct = (t: number) => (duration ? `${(t / duration) * 100}%` : '0%')
  return (
    <div ref={ref} role="slider" tabIndex={0} aria-label="Position" aria-valuemin={0} aria-valuemax={Math.round(duration)}
      aria-valuenow={Math.round(shown)} aria-valuetext={`${clock(shown)} sur ${clock(duration)}`}
      className="relative h-6 flex items-center cursor-pointer group/seek touch-none"
      onPointerDown={(e) => { (e.target as HTMLElement).setPointerCapture(e.pointerId); setDrag(at(e.clientX)) }}
      onPointerMove={(e) => { setHover(at(e.clientX)); if (drag !== null) setDrag(at(e.clientX)) }}
      onPointerUp={(e) => { if (drag !== null) { onSeek(at(e.clientX)); setDrag(null) } }}
      onPointerLeave={() => setHover(null)}>
      <div className="relative w-full h-1 group-hover/seek:h-1.5 transition-[height] rounded-full bg-white/20 overflow-hidden">
        <div className="absolute inset-y-0 left-0 bg-white/30" style={{ width: pct(Math.min(buffered, duration)) }} />
        <div className="absolute inset-y-0 left-0 bg-aurora" style={{ width: pct(shown) }} />
        {chapters.slice(1).map((c, i) => <div key={i} className="absolute inset-y-0 w-0.5 bg-black/70" style={{ left: pct(c.start) }} />)}
      </div>
      <div className="absolute size-3.5 -ml-[7px] rounded-full bg-paper shadow scale-0 group-hover/seek:scale-100 transition-transform" style={{ left: pct(shown) }} />
      {hover !== null && (
        <div className="absolute -top-8 -translate-x-1/2 rounded bg-black/85 px-2 py-1 text-xs tabular-nums pointer-events-none" style={{ left: pct(hover) }}>
          {clock(hover)}{chapters.filter((c) => c.start <= hover).pop()?.title ? ` · ${chapters.filter((c) => c.start <= hover).pop()!.title}` : ''}
        </div>
      )}
    </div>
  )
}
