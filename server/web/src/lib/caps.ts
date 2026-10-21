// Détection des capacités de lecture du navigateur. Rien n'est supposé : chaque codec est testé
// via canPlayType / MediaSource.isTypeSupported, qui reflètent le système et le matériel réels.

export interface Caps {
  containers: string[]
  video: string[]
  audio: string[]
  hls: 'native' | 'mse' | 'none'
  maxHeight: number
}

const VIDEO_TESTS: Record<string, string[]> = {
  h264: ['video/mp4; codecs="avc1.640028"'],
  hevc: ['video/mp4; codecs="hvc1.2.4.L153.B0"', 'video/mp4; codecs="hev1.2.4.L153.B0"', 'video/mp4; codecs="hvc1.1.6.L120.90"'],
  av1: ['video/mp4; codecs="av01.0.08M.08"'],
  vp9: ['video/webm; codecs="vp9"', 'video/mp4; codecs="vp09.00.40.08"'],
}

const AUDIO_TESTS: Record<string, string[]> = {
  aac: ['audio/mp4; codecs="mp4a.40.2"'],
  mp3: ['audio/mpeg', 'audio/mp4; codecs="mp4a.6B"'],
  opus: ['audio/webm; codecs="opus"', 'audio/mp4; codecs="opus"'],
  flac: ['audio/flac', 'audio/mp4; codecs="flac"'],
  ac3: ['audio/mp4; codecs="ac-3"'],
  eac3: ['audio/mp4; codecs="ec-3"'],
}

type Tester = (type: string) => boolean

export function detectCaps(video?: HTMLVideoElement, mse?: { isTypeSupported(t: string): boolean } | null): Caps {
  const v = video ?? document.createElement('video')
  const ms = mse === undefined ? (window.ManagedMediaSource ?? window.MediaSource ?? null) : mse
  const direct: Tester = (t) => v.canPlayType(t) === 'probably' || v.canPlayType(t) === 'maybe'
  const viaMse: Tester = (t) => !!ms && ms.isTypeSupported(t)
  // HLS natif réservé aux navigateurs Apple (ou sans MSE) : certains Chromium répondent « maybe » à
  // canPlayType('application/vnd.apple.mpegurl') puis échouent sur le fMP4 (constaté pendant les tests).
  const apple = /Apple/.test(navigator.vendor ?? '')
  const m3u8 = direct('application/vnd.apple.mpegurl')
  const nativeHls = m3u8 && (apple || !ms)
  // Un codec est retenu s'il est lisible directement ET via MSE (HLS par hls.js), ou nativement en HLS (Safari).
  const ok: Tester = (t) => direct(t) && (nativeHls || viaMse(t) || !ms)
  const pick = (tests: Record<string, string[]>) => Object.keys(tests).filter((k) => tests[k].some(ok))

  const containers = ['mp4']
  if (direct('video/webm')) containers.push('webm')
  if (direct('video/quicktime')) containers.push('mov')
  return {
    containers,
    video: pick(VIDEO_TESTS),
    audio: pick(AUDIO_TESTS),
    hls: nativeHls ? 'native' : ms ? 'mse' : 'none',
    maxHeight: 2160,
  }
}

declare global {
  interface Window {
    ManagedMediaSource?: typeof MediaSource
  }
}
