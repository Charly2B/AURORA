import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { ArrowDownWideNarrow, ArrowUpNarrowWide, Check, LayoutGrid, List, Rows3, SlidersHorizontal, X } from 'lucide-react'
import { api, qs } from '../lib/api'
import { useApi, useDocumentTitle } from '../lib/hooks'
import type { Card, Facets, LibraryRef } from '../lib/types'
import { itemLink, PosterCard, PosterGrid } from '../components/media'
import { Badge, Button, EmptyState, ErrorState, Select, Skeleton, Spinner, cx } from '../components/ui'
import { PageHeader } from '../components/Layout'
import { language, runtime } from '../lib/format'

const PAGE = 120
const SORTS = [
  { id: 'title', label: 'Titre' }, { id: 'added', label: "Date d'ajout" }, { id: 'year', label: 'Année' },
  { id: 'runtime', label: 'Durée' }, { id: 'rating', label: 'Note' }, { id: 'quality', label: 'Qualité vidéo' },
]
type View = 'grid' | 'compact' | 'list'

/** Bibliothèque films ou séries : tri, filtres, trois vues, chargement progressif. */
export function Library({ kind }: { kind: 'movie' | 'series' }) {
  const isMovie = kind === 'movie'
  useDocumentTitle(isMovie ? 'Films' : 'Séries')
  const [params, setParams] = useSearchParams()
  const [view, setView] = useState<View>(() => (localStorage.getItem('aurora.view') as View) || 'grid')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const facetsBase = `/facets/${isMovie ? 'movies' : 'series'}`
  const allLibs = useApi<LibraryRef[]>('/libraries')
  // Un onglet par dossier contenant ce type d'œuvre : les bibliothèques ne sont pas mélangées.
  const libs = (allLibs.data ?? []).filter((l) => (isMovie ? l.movies : l.series) > 0)
  const storeKey = `aurora.library.${kind}`
  const stored = (() => { try { return localStorage.getItem(storeKey) ?? undefined } catch { return undefined } })()
  const libParam = params.get('library') ?? stored
  const activeLib = libs.length <= 1 ? 'all'
    : libParam === 'all' ? 'all'
    : libs.some((l) => String(l.id) === libParam) ? libParam! : String(libs[0].id)
  const filters = useMemo(() => {
    const f = Object.fromEntries(params.entries())
    delete f.library
    return f
  }, [params])
  const sort = filters.sort ?? (isMovie ? 'added' : 'title')
  const order = filters.order ?? (sort === 'title' ? 'asc' : 'desc')

  const [items, setItems] = useState<Card[]>([])
  const [total, setTotal] = useState<number>()
  const [error, setError] = useState<Error>()
  const [loadingMore, setLoadingMore] = useState(false)
  const ready = !allLibs.loading
  const base = `/${isMovie ? 'movies' : 'series'}` + qs({ ...filters, library: activeLib === 'all' ? undefined : activeLib, sort, order, limit: PAGE })

  // « Tous les dossiers » : un bloc par dossier (titre du dossier puis ses œuvres), jamais un mélange.
  const grouped = activeLib === 'all' && libs.length > 1
  const [folderTotals, setFolderTotals] = useState<Record<number, number>>({})
  const allFoldersEmpty = grouped && libs.length > 0 && libs.every((l) => folderTotals[l.id] === 0)
  const groupKey = JSON.stringify([filters, sort, order])
  useEffect(() => setFolderTotals({}), [groupKey]) // nouveaux filtres : totaux recalculés
  useEffect(() => {
    if (!ready || grouped) return
    const ctrl = new AbortController()
    setItems([])
    setTotal(undefined)
    setError(undefined)
    api.get<{ items: Card[]; total: number }>(base + '&offset=0', { signal: ctrl.signal })
      .then((r) => { setItems(r.items); setTotal(r.total) })
      .catch((e) => !ctrl.signal.aborted && setError(e))
    return () => ctrl.abort()
  }, [base, ready, grouped])

  const sentinel = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const el = sentinel.current
    if (!el || total === undefined || items.length >= total) return
    const io = new IntersectionObserver(async ([e]) => {
      if (!e.isIntersecting || loadingMore) return
      setLoadingMore(true)
      try {
        const r = await api.get<{ items: Card[]; total: number }>(base + `&offset=${items.length}`)
        setItems((prev) => [...prev, ...r.items])
      } catch (err) {
        setError(err as Error)
      }
      setLoadingMore(false)
    }, { rootMargin: '800px' })
    io.observe(el)
    return () => io.disconnect()
  }, [items.length, total, base, loadingMore])

  const set = (k: string, v?: string) => {
    const p = new URLSearchParams(params)
    if (v) p.set(k, v)
    else p.delete(k)
    setParams(p, { replace: true })
  }
  const setViewSaved = (v: View) => { setView(v); try { localStorage.setItem('aurora.view', v) } catch { /* stockage indisponible */ } }
  const active = ['genre', 'year', 'resolution', 'hdr', 'watched', 'favorite', 'person', 'language'].filter((k) => filters[k])
  const chooseLib = (id: string) => {
    try { localStorage.setItem(storeKey, id) } catch { /* stockage indisponible */ }
    set('library', id)
  }
  const libName = libs.find((l) => String(l.id) === activeLib)?.name
  const facets = useApi<Facets>(ready ? facetsBase + (activeLib === 'all' ? '' : `?library=${activeLib}`) : null)

  return (
    <div>
      <PageHeader eyebrow={grouped ? `${libs.length} dossiers` : total !== undefined ? `${total} ${isMovie ? 'film' : 'série'}${total > 1 ? 's' : ''}` : ' '}
        title={libName ?? (isMovie ? 'Films' : 'Séries')}>
        <div className="flex flex-wrap items-end gap-2">
          <Select aria-label="Trier par" value={sort} onChange={(e) => set('sort', e.target.value)}>
            {SORTS.filter((s) => isMovie || s.id !== 'runtime').map((s) => <option key={s.id} value={s.id}>{s.label}</option>)}
          </Select>
          <button aria-label={order === 'asc' ? 'Ordre croissant' : 'Ordre décroissant'} onClick={() => set('order', order === 'asc' ? 'desc' : 'asc')}
            className="grid size-10 place-items-center rounded-lg bg-coal ring-1 ring-graphite text-mist hover:text-paper">
            {order === 'asc' ? <ArrowUpNarrowWide className="size-4" /> : <ArrowDownWideNarrow className="size-4" />}
          </button>
          <div className="flex rounded-lg bg-coal ring-1 ring-graphite p-0.5" role="group" aria-label="Affichage">
            {([['grid', LayoutGrid, 'Grille'], ['compact', Rows3, 'Compacte'], ['list', List, 'Liste']] as const).map(([v, Icon, label]) => (
              <button key={v} aria-label={label} aria-pressed={view === v} onClick={() => setViewSaved(v)}
                className={cx('grid size-9 place-items-center rounded-md', view === v ? 'bg-white/10 text-paper' : 'text-ash hover:text-paper')}>
                <Icon className="size-4" />
              </button>
            ))}
          </div>
          <Button icon={<SlidersHorizontal className="size-4" />} onClick={() => setFiltersOpen((o) => !o)} aria-expanded={filtersOpen}>
            Filtres{active.length ? ` (${active.length})` : ''}
          </Button>
        </div>
      </PageHeader>

      {libs.length > 1 && (
        <div className="px-[var(--gutter)] -mt-3 mb-8 flex gap-2 overflow-x-auto [scrollbar-width:none]" role="tablist" aria-label="Dossiers">
          {[...libs.map((l) => ({ id: String(l.id), name: l.name, n: isMovie ? l.movies : l.series })), { id: 'all', name: 'Tous les dossiers', n: undefined }].map((l) => (
            <button key={l.id} role="tab" aria-selected={activeLib === l.id} onClick={() => chooseLib(l.id)}
              className={cx('h-10 px-4 rounded-full text-sm whitespace-nowrap ring-1 transition-colors',
                activeLib === l.id ? 'bg-paper text-ink ring-paper font-semibold' : 'ring-graphite text-mist hover:text-paper')}>
              {l.name}{l.n !== undefined && <span className={cx('ml-2 text-xs tabular-nums', activeLib === l.id ? 'text-ink/60' : 'text-ash')}>{l.n}</span>}
            </button>
          ))}
        </div>
      )}

      {filtersOpen && facets.data && (
        <div className="mx-[var(--gutter)] mb-8 rounded-2xl bg-night ring-1 ring-white/[0.06] p-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-4 rise">
          {facets.data.genres.length > 0 && (
            <Select label="Genre" value={filters.genre ?? ''} onChange={(e) => set('genre', e.target.value)}>
              <option value="">Tous</option>{facets.data.genres.map((g) => <option key={g}>{g}</option>)}
            </Select>
          )}
          {facets.data.years.length > 0 && (
            <Select label="Année" value={filters.year ?? ''} onChange={(e) => set('year', e.target.value)}>
              <option value="">Toutes</option>{facets.data.years.map((y) => <option key={y}>{y}</option>)}
            </Select>
          )}
          {facets.data.resolutions.length > 0 && (
            <Select label="Résolution" value={filters.resolution ?? ''} onChange={(e) => set('resolution', e.target.value)}>
              <option value="">Toutes</option>{facets.data.resolutions.map((r) => <option key={r}>{r}</option>)}
            </Select>
          )}
          {facets.data.languages.length > 0 && (
            <Select label="Langue audio" value={filters.language ?? ''} onChange={(e) => set('language', e.target.value)}>
              <option value="">Toutes</option>{facets.data.languages.map((l) => <option key={l} value={l}>{language(l)}</option>)}
            </Select>
          )}
          {facets.data.people.length > 0 && (
            <Select label={isMovie ? 'Réalisateur' : 'Créateur'} value={filters.person ?? ''} onChange={(e) => set('person', e.target.value)}>
              <option value="">Tous</option>{facets.data.people.map((p) => <option key={p}>{p}</option>)}
            </Select>
          )}
          <Select label="Visionnage" value={filters.watched ?? ''} onChange={(e) => set('watched', e.target.value)}>
            <option value="">Tous</option><option value="unwatched">Non vus</option><option value="watched">Vus</option>
          </Select>
          <div className="flex flex-wrap items-end gap-2">
            {facets.data.hasHdr && <Chip on={!!filters.hdr} onClick={() => set('hdr', filters.hdr ? '' : '1')}>HDR</Chip>}
            <Chip on={!!filters.favorite} onClick={() => set('favorite', filters.favorite ? '' : '1')}>Favoris</Chip>
          </div>
          {active.length > 0 && (
            <div className="flex items-end">
              <Button variant="ghost" icon={<X className="size-4" />} onClick={() => setParams(new URLSearchParams(filters.sort ? { sort: filters.sort } : {}), { replace: true })}>Effacer les filtres</Button>
            </div>
          )}
        </div>
      )}

      {filters.person && <p className="px-[var(--gutter)] -mt-4 mb-6 text-sm text-ash">Avec <span className="text-paper">{filters.person}</span> <button onClick={() => set('person')} className="ml-2 underline">retirer</button></p>}

      {grouped && (
        <div className="space-y-14">
          {libs.map((l) => (
            <FolderBlock key={l.id} lib={l} kind={kind} view={view}
              query={qs({ ...filters, library: l.id, sort, order, limit: 60 })} onOpen={() => chooseLib(String(l.id))}
              onTotal={(n) => setFolderTotals((t) => (t[l.id] === n ? t : { ...t, [l.id]: n }))} />
          ))}
          {allFoldersEmpty && (
            <EmptyState title={active.length ? 'Aucun résultat' : isMovie ? 'Aucun film' : 'Aucune série'}>
              {active.length ? 'Aucune œuvre ne correspond à ces filtres, dans aucun dossier.' : "Rien n'a encore été indexé dans cette catégorie."}
            </EmptyState>
          )}
        </div>
      )}
      {!grouped && error && <ErrorState error={error} />}
      {!grouped && !error && total === undefined && <GridSkeleton />}
      {!grouped && total === 0 && (
        <EmptyState title={active.length ? 'Aucun résultat' : isMovie ? 'Aucun film' : 'Aucune série'}>
          {active.length ? 'Aucune œuvre ne correspond à ces filtres.' : "Rien n'a encore été indexé dans cette catégorie."}
        </EmptyState>
      )}
      {!grouped && items.length > 0 && <Cards items={items} view={view} />}
      {!grouped && <div ref={sentinel} className="h-10" />}
      {loadingMore && <div className="flex justify-center py-6"><Spinner /></div>}
    </div>
  )
}

function Cards({ items, view }: { items: Card[]; view: View }) {
  if (view === 'list') return <ListView items={items} />
  if (view === 'compact') {
    return (
      <div className="grid gap-x-3 gap-y-5 px-[var(--gutter)] grid-cols-[repeat(auto-fill,minmax(104px,1fr))]">
        {items.map((c) => <div key={c.id} className="cv-auto"><PosterCard card={c} width="w-full" /></div>)}
      </div>
    )
  }
  return <PosterGrid cards={items} />
}

/** Bloc d'un dossier : son nom en titre, puis ses œuvres. */
function FolderBlock({ lib, kind, view, query, onOpen, onTotal }: {
  lib: LibraryRef; kind: 'movie' | 'series'; view: View; query: string; onOpen: () => void; onTotal: (n: number) => void
}) {
  const res = useApi<{ items: Card[]; total: number }>(`/${kind === 'movie' ? 'movies' : 'series'}${query}`)
  const fresh = !res.loading && !res.error ? res.data : undefined // jamais d'anciens résultats (rechargement, erreur)
  useEffect(() => { if (fresh) onTotal(fresh.total) }, [fresh, onTotal])
  if (fresh && fresh.total === 0) return null // dossier sans résultat pour ces filtres
  // Cadre façon fenêtre macOS : coins arrondis, liseré fin, barre de titre avec les trois pastilles.
  // Dedans, --gutter est réduit : les grilles gardent leur marge habituelle sans la doubler avec le cadre.
  return (
    <section aria-label={lib.name}
      className="mx-[var(--gutter)] rounded-2xl bg-white/[0.06] ring-1 ring-white/[0.12] shadow-[0_24px_60px_-28px_rgba(0,0,0,0.85)] backdrop-blur-sm overflow-hidden">
      <div className="flex items-center gap-4 px-5 h-14 bg-white/[0.09] border-b border-white/[0.10]">
        <span className="hidden sm:flex gap-2" aria-hidden>
          <i className="size-3 rounded-full bg-[#ff5f57]/85" /><i className="size-3 rounded-full bg-[#febc2e]/85" /><i className="size-3 rounded-full bg-[#28c840]/85" />
        </span>
        <h2 className="font-display text-[22px] sm:text-[26px] leading-none truncate">{lib.name}</h2>
        <span className="text-sm text-ash tabular-nums whitespace-nowrap">
          {fresh ? `${fresh.total} ${kind === 'movie' ? 'film' : 'série'}${fresh.total > 1 ? 's' : ''}` : ''}
        </span>
        <button onClick={onOpen} className="ml-auto text-sm text-ash hover:text-aurora whitespace-nowrap">Ouvrir le dossier</button>
      </div>
      <div className="py-6 [--gutter:1.25rem]">
        {res.error && <ErrorState error={res.error} retry={res.reload} />}
        {!fresh && !res.error && <GridSkeleton />}
        {fresh && <Cards items={fresh.items} view={view} />}
        {fresh && fresh.total > fresh.items.length && (
          <div className="px-[var(--gutter)] mt-6">
            <Button onClick={onOpen}>Voir les {fresh.total} {kind === 'movie' ? 'films' : 'séries'} de « {lib.name} »</Button>
          </div>
        )}
      </div>
    </section>
  )
}

function Chip({ on, onClick, children }: { on: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button onClick={onClick} aria-pressed={on}
      className={cx('h-10 px-4 rounded-full text-sm ring-1 inline-flex items-center gap-1.5', on ? 'ring-aurora text-paper bg-aurora/[0.08]' : 'ring-graphite text-ash hover:text-paper')}>
      {on && <Check className="size-3.5 text-aurora" />}{children}
    </button>
  )
}

function ListView({ items }: { items: Card[] }) {
  return (
    <ul className="px-[var(--gutter)] divide-y divide-white/[0.05]">
      {items.map((c) => (
        <li key={c.id} className="cv-auto [contain-intrinsic-size:auto_72px]">
          <Link to={itemLink(c)} className="flex items-center gap-4 py-2.5 hover:bg-white/[0.03] rounded-lg px-2 -mx-2">
            <div className="w-10 aspect-[2/3] rounded overflow-hidden bg-coal shrink-0">
              {(c.poster || c.thumb) && <img src={c.poster || c.thumb} alt="" loading="lazy" className="size-full object-cover" />}
            </div>
            <div className="min-w-0 flex-1">
              <div className="font-medium truncate">{c.title}</div>
              <div className="text-xs text-ash truncate">{c.genres?.slice(0, 3).join(', ')}</div>
            </div>
            <span className="hidden sm:block w-14 text-sm text-ash tabular-nums">{c.year ?? ''}</span>
            <span className="hidden md:block w-20 text-sm text-ash">{runtime(c.runtime, c.duration)}</span>
            <span className="hidden md:flex w-28 gap-1">{c.resolution && <Badge>{c.resolution}</Badge>}{c.hdr && <Badge tone="accent">HDR</Badge>}</span>
            <span className="w-12 text-sm text-mist tabular-nums text-right">{c.rating ? c.rating.toFixed(1) : ''}</span>
            <span className="w-6">{c.watched && <Check className="size-4 text-aurora" aria-label="Vu" />}</span>
          </Link>
        </li>
      ))}
    </ul>
  )
}

function GridSkeleton() {
  return (
    <div className="grid gap-x-4 gap-y-7 px-[var(--gutter)] grid-cols-[repeat(auto-fill,minmax(160px,1fr))]" aria-busy="true">
      {Array.from({ length: 18 }, (_, i) => <Skeleton key={i} className="aspect-[2/3]" />)}
    </div>
  )
}
