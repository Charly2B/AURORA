// Éléments partagés des fiches film / série : décor, distribution, informations techniques, correction.
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Check, ImageIcon, Search, Wand2 } from 'lucide-react'
import { api, ApiError } from '../lib/api'
import type { Detail, FileInfo, Person } from '../lib/types'
import { audioLabel, bytes, language, videoCodec } from '../lib/format'
import { Badge, Button, Field, Modal, Notice, Spinner, cx, useToast } from './ui'

export function Backdrop({ src, children }: { src?: string; children: ReactNode }) {
  return (
    <section className="relative overflow-hidden">
      {src && <img src={src} alt="" className="absolute inset-0 size-full object-cover object-[center_20%] opacity-90" />}
      <div className="absolute inset-0 hero-fade" />
      <div className="absolute inset-0 bg-gradient-to-t from-ink via-ink/40 to-transparent" />
      <div className="relative px-[var(--gutter)] pt-28 sm:pt-36 pb-10">{children}</div>
    </section>
  )
}

export function MetaLine({ d }: { d: Detail }) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-mist">
      {d.year && <span>{d.year}</span>}
      {d.contentRating && <span className="px-1.5 rounded ring-1 ring-white/25 text-xs">{d.contentRating}</span>}
      {d.rating != null && d.ratingSource && (
        <span title={`Note moyenne ${d.ratingSource}`}><span className="text-paper font-semibold">{d.rating.toFixed(1)}</span><span className="text-ash"> /10 {d.ratingSource}</span></span>
      )}
      {d.genres?.map((g) => <span key={g} className="text-ash">{g}</span>)}
    </div>
  )
}

export function Cast({ people }: { people: Person[] }) {
  const crew = people.filter((p) => p.role !== 'actor')
  const cast = people.filter((p) => p.role === 'actor')
  if (!people.length) return null
  return (
    <section className="space-y-5" aria-label="Distribution">
      {crew.length > 0 && (
        <p className="px-[var(--gutter)] text-sm text-ash">
          {crew[0].role === 'director' ? 'Réalisation' : 'Création'} :{' '}
          {crew.map((p, i) => (
            <span key={p.name}>{i > 0 && ', '}<Link className="text-paper hover:text-aurora" to={`/movies?person=${encodeURIComponent(p.name)}`}>{p.name}</Link></span>
          ))}
        </p>
      )}
      {cast.length > 0 && (
        <div className="rail">
          {cast.map((p) => (
            <Link key={p.name + p.character} to={`/search?q=${encodeURIComponent(p.name)}`} className="w-[104px] shrink-0 group">
              <div className="size-[104px] rounded-full overflow-hidden bg-coal ring-1 ring-white/10 group-hover:ring-aurora/50">
                {p.photo ? <img src={p.photo} alt="" loading="lazy" className="size-full object-cover" /> :
                  <div className="size-full grid place-items-center font-display text-3xl text-ash">{p.name.slice(0, 1)}</div>}
              </div>
              <div className="mt-2 text-[13px] text-center leading-tight">{p.name}</div>
              {p.character && <div className="text-[11.5px] text-ash text-center leading-tight mt-0.5 line-clamp-2">{p.character}</div>}
            </Link>
          ))}
        </div>
      )}
    </section>
  )
}

/** Fiche technique d'une version : lisible, sans jargon inutile, mais complète. */
export function TechInfo({ file }: { file: FileInfo }) {
  return (
    <dl className="grid sm:grid-cols-2 gap-x-10 gap-y-4 text-sm">
      <Row label="Vidéo">
        <span className="flex flex-wrap gap-1.5 items-center">
          {file.resolution && <Badge>{file.resolution}</Badge>}
          {file.hdr && <Badge tone="accent">{file.hdr}</Badge>}
          <span>{videoCodec(file.videoCodec)} {file.videoProfile && <span className="text-ash">({file.videoProfile})</span>}</span>
          {file.width > 0 && <span className="text-ash">{file.width}×{file.height}</span>}
        </span>
      </Row>
      <Row label="Fichier"><span className="break-all">{file.name}</span> <span className="text-ash">· {bytes(file.size)}</span></Row>
      <Row label="Pistes audio">
        {file.audio.length ? <ul className="space-y-0.5">{file.audio.map((a) => <li key={a.index}>{audioLabel(a)}</li>)}</ul> : '—'}
      </Row>
      <Row label="Sous-titres">
        {file.subtitles.length ? (
          <ul className="space-y-0.5">{file.subtitles.map((s) => (
            <li key={s.index}>{language(s.language)}{s.forced && ' (forcés)'}{!s.text && <span className="text-ash"> · image</span>}</li>
          ))}</ul>
        ) : 'Aucun'}
      </Row>
      {file.probeState !== 'ok' && (
        <div className="sm:col-span-2">
          <Notice tone={file.probeState === 'error' ? 'warn' : 'info'}>
            {file.probeState === 'error' ? `Analyse technique impossible : ${file.probeError}` : 'Analyse technique en attente : elle sera faite au lancement de la lecture.'}
          </Notice>
        </div>
      )}
    </dl>
  )
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="eyebrow mb-1.5">{label}</dt>
      <dd className="text-mist">{children}</dd>
    </div>
  )
}

interface Candidate { id: number; title: string; originalTitle: string; year: number; overview: string; poster?: string }

/** Correction d'identification (administrateur) : choisir la bonne fiche TMDB, ou corriger titre/année. */
export function FixMatch({ item, onDone }: { item: Detail; onDone: (newId: number) => void }) {
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState(item.originalTitle || item.title)
  const [year, setYear] = useState(item.year ? String(item.year) : '')
  const [results, setResults] = useState<Candidate[]>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState<number | 'search' | 'manual'>()
  const [title, setTitle] = useState(item.title)
  const toast = useToast()
  const search = async () => {
    setBusy('search')
    setError(undefined)
    try {
      setResults(await api.get<Candidate[]>(`/admin/items/${item.id}/candidates?q=${encodeURIComponent(q)}${year ? `&year=${year}` : ''}`))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    }
    setBusy(undefined)
  }
  const pick = async (id: number) => {
    setBusy(id)
    try {
      const r = await api.post<{ itemId: number }>(`/admin/items/${item.id}/match`, { tmdbId: id })
      toast('ok', 'Identification corrigée et verrouillée')
      setOpen(false)
      onDone(r.itemId)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    }
    setBusy(undefined)
  }
  const manual = async () => {
    setBusy('manual')
    try {
      await api.put(`/admin/items/${item.id}`, { title, year: Number(year) || 0 })
      toast('ok', 'Titre corrigé et verrouillé')
      setOpen(false)
      onDone(item.id)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    }
    setBusy(undefined)
  }
  return (
    <>
      <Button variant="ghost" size="sm" icon={<Wand2 className="size-4" />} onClick={() => { setOpen(true); if (!results) search() }}>
        Corriger l'identification
      </Button>
      <Modal open={open} onClose={() => setOpen(false)} title="Corriger l'identification" wide>
        <div className="space-y-5">
          {item.matchState === 'ambiguous' && <Notice tone="warn">Plusieurs fiches correspondaient : AURORA n'a pas choisi à votre place.</Notice>}
          <div className="flex flex-wrap gap-3 items-end">
            <Field label="Titre recherché" value={q} onChange={(e) => setQ(e.target.value)} className="flex-1 min-w-48" onKeyDown={(e) => e.key === 'Enter' && search()} />
            <Field label="Année" value={year} onChange={(e) => setYear(e.target.value.replace(/\D/g, ''))} className="w-24" inputMode="numeric" />
            <Button icon={<Search className="size-4" />} busy={busy === 'search'} onClick={search}>Chercher</Button>
          </div>
          {error && <Notice tone="error">{error}</Notice>}
          {busy === 'search' && <Spinner label="Recherche sur TMDB" />}
          {results && busy !== 'search' && (
            <ul className="max-h-[46vh] overflow-y-auto space-y-2 -mx-1 px-1">
              {results.length === 0 && <li className="text-ash text-sm">Aucun résultat.</li>}
              {results.map((r) => (
                <li key={r.id}>
                  <button onClick={() => pick(r.id)} disabled={busy !== undefined}
                    className={cx('w-full flex gap-4 text-left rounded-xl p-3 ring-1 ring-white/[0.06] hover:ring-aurora/50 hover:bg-white/[0.03]', item.tmdbId === r.id && 'ring-aurora/60')}>
                    <div className="w-14 aspect-[2/3] rounded bg-coal overflow-hidden shrink-0">{r.poster && <img src={r.poster} alt="" className="size-full object-cover" />}</div>
                    <div className="min-w-0">
                      <div className="font-medium">{r.title} <span className="text-ash font-normal">{r.year || ''}</span></div>
                      {r.originalTitle !== r.title && <div className="text-xs text-ash">{r.originalTitle}</div>}
                      <p className="text-xs text-ash line-clamp-2 mt-1">{r.overview}</p>
                    </div>
                    {busy === r.id && <Spinner label="Application" />}
                  </button>
                </li>
              ))}
            </ul>
          )}
          <details className="rounded-xl ring-1 ring-white/[0.06] p-4">
            <summary className="cursor-pointer text-sm text-mist">Corriger sans TMDB (titre et année)</summary>
            <div className="mt-4 flex flex-wrap gap-3 items-end">
              <Field label="Titre" value={title} onChange={(e) => setTitle(e.target.value)} className="flex-1 min-w-48" />
              <Button busy={busy === 'manual'} onClick={manual}>Enregistrer</Button>
            </div>
          </details>
          <p className="text-[11px] text-ash">Données fournies par TMDB. This product uses the TMDB API but is not endorsed or certified by TMDB.</p>
        </div>
      </Modal>
    </>
  )
}

interface ArtworkOption { path: string; preview: string; language: string; votes: number; width: number; height: number }

/** Choix d'une autre affiche ou d'un autre fond parmi ceux que propose TMDB (administrateur). */
export function ArtworkPicker({ item, onDone }: { item: Detail; onDone: () => void }) {
  const [open, setOpen] = useState(false)
  const [tab, setTab] = useState<'posters' | 'backdrops'>('posters')
  const [lang, setLang] = useState<string>('all')
  const [data, setData] = useState<{ posters: ArtworkOption[]; backdrops: ArtworkOption[] }>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState<string>()
  const toast = useToast()
  const load = async () => {
    setOpen(true)
    if (data) return
    setError(undefined)
    try {
      setData(await api.get(`/admin/items/${item.id}/artwork`))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    }
  }
  const choose = async (a: ArtworkOption) => {
    setBusy(a.path)
    try {
      await api.post(`/admin/items/${item.id}/artwork`, { type: tab === 'posters' ? 'poster' : 'backdrop', path: a.path })
      toast('ok', tab === 'posters' ? 'Affiche changée' : "Fond d'écran changé")
      onDone()
    } catch (e) {
      toast('error', e instanceof ApiError ? e.message : String(e))
    }
    setBusy(undefined)
  }
  const list = (data?.[tab] ?? []).filter((a) => lang === 'all' || (lang === 'none' ? !a.language : a.language === lang))
  const langs = Array.from(new Set((data?.[tab] ?? []).map((a) => a.language || 'none')))
  const label = (l: string) => ({ fr: 'Français', en: 'Anglais', none: 'Sans texte' } as Record<string, string>)[l] ?? l.toUpperCase()
  return (
    <>
      <Button variant="ghost" size="sm" icon={<ImageIcon className="size-4" />} onClick={load}>Changer l'affiche</Button>
      <Modal open={open} onClose={() => setOpen(false)} title="Affiche et fond d'écran" wide>
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            {(['posters', 'backdrops'] as const).map((t) => (
              <button key={t} onClick={() => { setTab(t); setLang('all') }} aria-pressed={tab === t}
                className={cx('h-9 px-4 rounded-full text-sm ring-1', tab === t ? 'bg-paper text-ink ring-paper font-semibold' : 'ring-graphite text-mist')}>
                {t === 'posters' ? `Affiches${data ? ` (${data.posters.length})` : ''}` : `Fonds${data ? ` (${data.backdrops.length})` : ''}`}
              </button>
            ))}
            <span className="mx-1 h-5 w-px bg-white/10" />
            {['all', ...langs].map((l) => (
              <button key={l} onClick={() => setLang(l)} aria-pressed={lang === l}
                className={cx('h-8 px-3 rounded-full text-xs ring-1', lang === l ? 'ring-aurora text-paper bg-aurora/[0.08]' : 'ring-graphite text-ash')}>
                {l === 'all' ? 'Toutes' : label(l)}
              </button>
            ))}
          </div>
          {error && <Notice tone="error">{error}</Notice>}
          {!data && !error && <Spinner label="Chargement des images TMDB" />}
          {data && list.length === 0 && <p className="text-sm text-ash">Aucune image dans cette catégorie.</p>}
          <div className={cx('grid gap-3 max-h-[60vh] overflow-y-auto pr-1', tab === 'posters' ? 'grid-cols-3 sm:grid-cols-5' : 'grid-cols-1 sm:grid-cols-2')}>
            {list.map((a) => {
              const current = tab === 'posters' ? item.poster : item.backdrop
              return (
                <button key={a.path} onClick={() => choose(a)} disabled={!!busy}
                  className="group relative rounded-lg overflow-hidden ring-1 ring-white/10 hover:ring-aurora focus-visible:ring-aurora bg-coal">
                  <img src={a.preview} alt="" loading="lazy" className={cx('w-full object-cover', tab === 'posters' ? 'aspect-[2/3]' : 'aspect-video')} />
                  <span className="absolute left-1.5 top-1.5 rounded bg-ink/80 px-1.5 py-0.5 text-[10px] text-mist">{label(a.language || 'none')}</span>
                  {busy === a.path && <span className="absolute inset-0 grid place-items-center bg-ink/60"><Spinner label="Application" /></span>}
                  {current && busy !== a.path && <span className="sr-only">Choisir cette image</span>}
                  <span className="absolute inset-x-0 bottom-0 translate-y-full group-hover:translate-y-0 transition-transform bg-ink/85 py-1.5 text-xs text-center">
                    <Check className="inline size-3.5 text-aurora mr-1" />Choisir
                  </span>
                </button>
              )
            })}
          </div>
          <p className="text-[11px] text-ash">Images fournies par TMDB. Votre choix est conservé par les mises à jour automatiques ; seule une nouvelle identification (autre œuvre) le remplace.</p>
        </div>
      </Modal>
    </>
  )
}
