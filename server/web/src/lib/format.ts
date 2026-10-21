// Formatage d'affichage (français).

export function clock(seconds: number): string {
  if (!isFinite(seconds) || seconds < 0) seconds = 0
  const s = Math.floor(seconds % 60)
  const m = Math.floor((seconds / 60) % 60)
  const h = Math.floor(seconds / 3600)
  const pad = (n: number) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`
}

export function runtime(minutes?: number, seconds?: number): string {
  const m = minutes ?? (seconds ? Math.round(seconds / 60) : 0)
  if (!m) return ''
  const h = Math.floor(m / 60)
  return h ? `${h} h ${String(m % 60).padStart(2, '0')}` : `${m} min`
}

export function remaining(position: number, duration: number): string {
  const left = Math.max(0, duration - position)
  if (left < 60) return "Moins d'une minute restante"
  const m = Math.round(left / 60)
  return m >= 60 ? `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, '0')} restantes` : `${m} min restantes`
}

export function bytes(n: number): string {
  if (n < 1024) return `${n} o`
  const u = ['Ko', 'Mo', 'Go', 'To']
  let i = -1
  do {
    n /= 1024
    i++
  } while (n >= 1024 && i < u.length - 1)
  return `${n.toFixed(n < 10 ? 1 : 0).replace('.', ',')} ${u[i]}`
}

export function date(unix?: number): string {
  if (!unix) return '—'
  return new Date(unix * 1000).toLocaleString('fr-FR', { dateStyle: 'medium', timeStyle: 'short' })
}

export function relative(unix?: number): string {
  if (!unix) return 'jamais'
  const d = Date.now() / 1000 - unix
  if (d < 60) return "à l'instant"
  if (d < 3600) return `il y a ${Math.round(d / 60)} min`
  if (d < 86400) return `il y a ${Math.round(d / 3600)} h`
  return `il y a ${Math.round(d / 86400)} j`
}

const LANGS: Record<string, string> = {
  fre: 'Français', fra: 'Français', fr: 'Français', eng: 'Anglais', en: 'Anglais', ger: 'Allemand', deu: 'Allemand',
  de: 'Allemand', spa: 'Espagnol', es: 'Espagnol', ita: 'Italien', it: 'Italien', jpn: 'Japonais', ja: 'Japonais',
  por: 'Portugais', pt: 'Portugais', dut: 'Néerlandais', nld: 'Néerlandais', kor: 'Coréen', chi: 'Chinois', zho: 'Chinois',
  rus: 'Russe', ara: 'Arabe',
}

export function language(code?: string): string {
  if (!code) return 'Inconnue'
  return LANGS[code.toLowerCase()] ?? code.toUpperCase()
}

/** Égalité de langue entre codes ISO 639-1 / 639-2 (fr, fre, fra). */
export function sameLanguage(a?: string, b?: string): boolean {
  if (!a || !b) return false
  return language(a) === language(b)
}

const CHANNELS: Record<number, string> = { 1: 'Mono', 2: 'Stéréo', 6: '5.1', 8: '7.1' }

export function audioLabel(a: { codec: string; channels: number; language: string; title: string; atmos: boolean; profile?: string }): string {
  const codec = ({ ac3: 'Dolby Digital', eac3: 'Dolby Digital+', truehd: 'Dolby TrueHD', dts: a.profile?.includes('MA') ? 'DTS-HD MA' : 'DTS', aac: 'AAC', flac: 'FLAC', opus: 'Opus', mp3: 'MP3' } as Record<string, string>)[a.codec] ?? a.codec.toUpperCase()
  const parts = [language(a.language), codec, CHANNELS[a.channels] ?? `${a.channels} canaux`]
  if (a.atmos) parts.push('Atmos')
  return parts.join(' · ') + (a.title && !a.title.toLowerCase().includes(codec.toLowerCase()) ? ` — ${a.title}` : '')
}

export function episodeCode(season?: number, episode?: number, end?: number): string {
  if (season === undefined || episode === undefined) return ''
  const e = `E${String(episode).padStart(2, '0')}${end ? `-E${String(end).padStart(2, '0')}` : ''}`
  return season === 0 ? `Spécial ${e}` : `S${String(season).padStart(2, '0')} ${e}`
}

export function videoCodec(c: string): string {
  return ({ h264: 'H.264', hevc: 'HEVC', av1: 'AV1', vp9: 'VP9', mpeg2video: 'MPEG-2', mpeg4: 'MPEG-4' } as Record<string, string>)[c] ?? c.toUpperCase()
}
