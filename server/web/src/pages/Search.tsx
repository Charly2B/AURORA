import { useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Search as SearchIcon, UserRound, X } from 'lucide-react'
import { useApi, useDebounced, useDocumentTitle } from '../lib/hooks'
import type { Card } from '../lib/types'
import { PosterCard, WideCard } from '../components/media'
import { EmptyState, ErrorState, Spinner } from '../components/ui'

interface Results { movies: Card[]; series: Card[]; episodes: Card[]; people: string[]; fuzzy: boolean }

/** Recherche instantanée, locale (index plein texte sur le NAS, aucun accès Internet ni disque vidéo). */
export function Search() {
  useDocumentTitle('Recherche')
  const [params, setParams] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const dq = useDebounced(q.trim(), 180)
  const input = useRef<HTMLInputElement>(null)
  useEffect(() => input.current?.focus(), [])
  useEffect(() => { setParams(dq ? { q: dq } : {}, { replace: true }) }, [dq, setParams])
  const res = useApi<Results>(dq ? `/search?q=${encodeURIComponent(dq)}&limit=24` : null)
  const r = dq ? res.data : undefined
  const empty = r && !r.movies.length && !r.series.length && !r.episodes.length && !r.people.length

  return (
    <div className="pt-24">
      <div className="px-[var(--gutter)] mb-10">
        <label className="relative block max-w-3xl">
          <span className="sr-only">Rechercher</span>
          <SearchIcon className="absolute left-0 top-1/2 -translate-y-1/2 size-7 text-ash" aria-hidden />
          <input ref={input} value={q} onChange={(e) => setQ(e.target.value)} placeholder="Un film, une série, un acteur…"
            className="w-full bg-transparent border-b border-graphite focus:border-aurora outline-none pl-12 pr-10 py-4 font-display text-3xl sm:text-5xl placeholder:text-smoke transition-colors"
            autoComplete="off" spellCheck={false} />
          {q && <button aria-label="Effacer" onClick={() => setQ('')} className="absolute right-0 top-1/2 -translate-y-1/2 text-ash hover:text-paper"><X className="size-6" /></button>}
        </label>
        {r?.fuzzy && <p className="mt-3 text-sm text-ash">Aucun résultat exact : voici les titres les plus proches.</p>}
      </div>
      {res.loading && dq && !r && <div className="px-[var(--gutter)]"><Spinner label="Recherche" /></div>}
      {res.error && <ErrorState error={res.error} retry={res.reload} />}
      {empty && <EmptyState title="Rien trouvé">Aucun film, série, épisode ou personne ne correspond à « {dq} ».</EmptyState>}
      {r && (
        <div className="space-y-12">
          {r.people.length > 0 && (
            <Group title="Personnes">
              <div className="flex flex-wrap gap-2 px-[var(--gutter)]">
                {r.people.map((p) => (
                  <Link key={p} to={`/movies?person=${encodeURIComponent(p)}`} className="inline-flex items-center gap-2 h-10 px-4 rounded-full bg-coal ring-1 ring-white/[0.06] hover:ring-aurora/50 text-sm">
                    <UserRound className="size-4 text-aurora" /> {p}
                  </Link>
                ))}
              </div>
            </Group>
          )}
          {r.movies.length > 0 && <Group title="Films"><div className="rail">{r.movies.map((c) => <PosterCard key={c.id} card={c} />)}</div></Group>}
          {r.series.length > 0 && <Group title="Séries"><div className="rail">{r.series.map((c) => <PosterCard key={c.id} card={c} />)}</div></Group>}
          {r.episodes.length > 0 && <Group title="Épisodes"><div className="rail">{r.episodes.map((c) => <WideCard key={c.id} card={c} />)}</div></Group>}
        </div>
      )}
    </div>
  )
}

function Group({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section aria-label={title}>
      <h2 className="eyebrow px-[var(--gutter)] mb-4">{title}</h2>
      {children}
    </section>
  )
}
