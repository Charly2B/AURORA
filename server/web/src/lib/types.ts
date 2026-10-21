// Types partagés avec l'API (voir internal/catalog/*.go).

export interface Card {
  id: number
  kind: 'movie' | 'series' | 'episode' | 'collection'
  title: string
  year?: number
  poster?: string
  backdrop?: string
  thumb?: string
  rating?: number
  runtime?: number
  duration?: number
  resolution?: string
  hdr?: string
  watched: boolean
  position?: number
  favorite: boolean
  watchlist: boolean
  overview?: string
  genres?: string[]
  addedAt: number
  seriesId?: number
  seriesTitle?: string
  season?: number
  episode?: number
  episodeEnd?: number
  unwatched?: number
  total?: number
  versions?: number
}

export interface Audio {
  index: number
  stream: number
  codec: string
  profile: string
  channels: number
  layout: string
  language: string
  title: string
  default: boolean
  atmos: boolean
}

export interface Subtitle {
  index: number
  codec: string
  language: string
  title: string
  default: boolean
  forced: boolean
  text: boolean
}

export interface Chapter {
  start: number
  end: number
  title: string
}

export interface FileInfo {
  id: number
  name: string
  label: string
  edition?: string
  size: number
  container: string
  duration: number
  bitrate: number
  videoCodec: string
  videoProfile: string
  width: number
  height: number
  resolution: string
  hdr: string
  audio: Audio[]
  subtitles: Subtitle[]
  chapters: Chapter[]
  probeState: string
  probeError?: string
  addedAt: number
}

export interface Person {
  name: string
  role: 'actor' | 'director' | 'creator'
  character?: string
  photo?: string
}

export interface Season {
  number: number
  title: string
  overview?: string
  poster?: string
  episodes: number
  unwatched: number
}

export interface Detail extends Card {
  originalTitle?: string
  tagline?: string
  ratingSource?: string
  contentRating?: string
  status?: string
  airDate?: string
  trailer?: string
  matchState: string
  tmdbId?: number
  libraryId: number
  files: FileInfo[]
  people: Person[]
  state: {
    position: number
    duration: number
    watched: boolean
    playCount: number
    lastPlayedAt?: number
    lastFileId?: number
    favorite: boolean
    watchlist: boolean
  }
  collections?: Card[]
  seasons?: Season[]
  nextUp?: Card
  similar?: Card[]
}

export interface HomePref {
  id: string
  visible: boolean
}

export interface Prefs {
  home: HomePref[]
  autoplayNext: boolean
  audioLang: string
  subLang: string
  subMode: 'auto' | 'always' | 'off'
  subSize: 's' | 'm' | 'l'
}

export interface User {
  id: number
  username: string
  role: 'admin' | 'user' | 'child'
  maxAge?: number | null
  hideUnrated: boolean
  prefs: Prefs
}

export interface Status {
  version: string
  hasAdmin: boolean
  setupDone: boolean
  user?: User
}

export interface Section {
  id: string
  title: string
  libraryId?: number
  kind?: 'movie' | 'series'
  items: Card[]
}

export interface LibraryRef {
  id: number
  name: string
  movies: number
  series: number
}

export interface Facets {
  genres: string[]
  years: number[]
  resolutions: string[]
  languages: string[]
  hasHdr: boolean
  people: string[]
}

export interface ScanProgress {
  taskId: number
  libraryId: number
  library: string
  phase: 'walk' | 'identify' | 'probe' | 'metadata' | 'done'
  done: number
  total: number
  errors: number
  current: string
  status: 'running' | 'done' | 'failed' | 'cancelled'
  message: string
}

export interface PlayResponse {
  mode: 'direct' | 'remux' | 'transcode'
  reason: string
  url: string
  sessionId?: string
  offset: number
  start: number
  duration: number
  chapters: Chapter[]
  itemId: number
  fileId: number
  audio: Audio[]
  audioStream: number
  subtitles: (Subtitle & { label: string; external: boolean; url?: string })[]
  burnedSubtitle: boolean
}
