import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { Check, Eye, EyeOff, MoreHorizontal, Play, Plus } from 'lucide-react'
import { api } from '../lib/api'
import { useApi, useDocumentTitle } from '../lib/hooks'
import { useSession } from '../lib/session'
import type { Card, Detail } from '../lib/types'
import { clock, episodeCode, runtime } from '../lib/format'
import { ArtworkPicker, Backdrop, Cast, FixMatch, MetaLine } from '../components/detail'
import { Artwork, FavoriteButton, playLink } from '../components/media'
import { Button, ErrorState, PageLoader, ProgressBar, Skeleton, cx, useToast } from '../components/ui'

export function SeriesDetail() {
  const { id } = useParams()
  const [params, setParams] = useSearchParams()
  const nav = useNavigate()
  const { user } = useSession()
  const toast = useToast()
  const res = useApi<Detail>(`/items/${id}`)
  const d = res.data
  useDocumentTitle(d?.title)
  const seasonParam = params.get('season')
  const season = seasonParam !== null ? Number(seasonParam) : (d?.nextUp?.season ?? d?.seasons?.[0]?.number)
  const eps = useApi<Card[]>(d && season !== undefined ? `/items/${d.id}/seasons/${season}` : null)
  const focusEp = Number(params.get('episode')) || undefined

  if (res.error) return <div className="pt-24"><ErrorState error={res.error} retry={res.reload} /></div>
  if (!d) return <PageLoader />
  const next = d.nextUp

  const setState = async (itemId: number, patch: Record<string, unknown>, msg: string) => {
    try {
      await api.post(`/items/${itemId}/state`, patch)
      res.reload()
      eps.reload()
      toast('ok', msg)
    } catch (e) {
      toast('error', (e as Error).message)
    }
  }
  const allWatched = d.total !== undefined && d.unwatched === 0

  return (
    <div>
      <Backdrop src={d.backdrop}>
        <div className="flex flex-col md:flex-row gap-8 md:gap-12 items-start max-w-6xl">
          <div className="w-40 sm:w-56 shrink-0 aspect-[2/3] rounded-xl overflow-hidden ring-1 ring-white/10 shadow-[var(--shadow-lift)] rise">
            <Artwork src={d.poster} title={d.title} subtitle={d.year ? String(d.year) : undefined} />
          </div>
          <div className="min-w-0 flex-1 rise [animation-delay:80ms]">
            <p className="eyebrow mb-3">Série{d.status ? ` · ${d.status}` : ''}</p>
            <h1 className="title-xl text-[42px] sm:text-[64px] mb-3">{d.title}</h1>
            <MetaLine d={d} />
            <p className="text-sm text-ash mt-2">{d.seasons?.filter((s) => s.number > 0).length ?? 0} saison(s) · {d.total} épisode(s) disponibles{d.unwatched ? ` · ${d.unwatched} non vus` : ''}</p>
            <div className="flex flex-wrap items-center gap-3 mt-7">
              {next && (
                <Button variant="primary" size="lg" icon={<Play className="size-5 fill-current" />} onClick={() => nav(playLink(next))}>
                  {next.position ? 'Reprendre' : 'Lecture'} {episodeCode(next.season, next.episode, next.episodeEnd)}
                </Button>
              )}
              <Button size="lg" variant="ghost" icon={d.watchlist ? <Check className="size-5 text-aurora" /> : <Plus className="size-5" />}
                onClick={() => setState(d.id, { watchlist: !d.watchlist }, d.watchlist ? 'Retirée de ma liste' : 'Ajoutée à ma liste')}>Ma liste</Button>
              <FavoriteButton active={d.favorite} onClick={() => setState(d.id, { favorite: !d.favorite }, d.favorite ? 'Retirée des favoris' : 'Ajoutée aux favoris')} />
            </div>
            {next && <p className="mt-3 text-sm text-ash">Prochain épisode : <span className="text-mist">{next.title}</span>{next.position ? ` — reprise à ${clock(next.position)}` : ''}</p>}
            {d.overview && <p className="mt-6 max-w-2xl text-[15.5px] leading-relaxed text-mist">{d.overview}</p>}
            <div className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-ash">
              <button className="inline-flex items-center gap-1.5 hover:text-paper"
                onClick={() => setState(d.id, { watched: !allWatched }, allWatched ? 'Série marquée comme non vue' : 'Série marquée comme vue')}>
                {allWatched ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />} {allWatched ? 'Tout marquer comme non vu' : 'Tout marquer comme vu'}
              </button>
              {user?.role === 'admin' && <FixMatch item={d} onDone={(nid) => (nid !== d.id ? nav(`/series/${nid}`, { replace: true }) : res.reload())} />}
              {user?.role === 'admin' && !!d.tmdbId && <ArtworkPicker item={d} onDone={res.reload} />}
            </div>
          </div>
        </div>
      </Backdrop>

      <section className="mt-2" aria-label="Épisodes">
        {d.seasons && d.seasons.length > 0 && (
          <div className="px-[var(--gutter)] flex items-center gap-2 overflow-x-auto [scrollbar-width:none] mb-6" role="tablist" aria-label="Saisons">
            {d.seasons.map((s) => (
              <button key={s.number} role="tab" aria-selected={s.number === season}
                onClick={() => setParams({ season: String(s.number) }, { replace: true })}
                className={cx('h-10 px-4 rounded-full text-sm whitespace-nowrap ring-1 transition-colors',
                  s.number === season ? 'bg-paper text-ink ring-paper font-semibold' : 'ring-graphite text-mist hover:text-paper')}>
                {s.title}{s.unwatched > 0 && s.unwatched < s.episodes ? <span className="ml-1.5 text-aurora-deep">·{s.unwatched}</span> : null}
              </button>
            ))}
            {season !== undefined && (
              <button className="ml-auto text-xs text-ash hover:text-paper whitespace-nowrap px-2"
                onClick={() => setState(d.id, { watched: true, season }, 'Saison marquée comme vue')}>Marquer la saison comme vue</button>
            )}
          </div>
        )}
        <div className="px-[var(--gutter)] space-y-2 max-w-5xl">
          {eps.loading && Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-28" />)}
          {eps.error && <ErrorState error={eps.error} retry={eps.reload} />}
          {eps.data?.map((e) => (
            <EpisodeRow key={e.id} ep={e} focused={focusEp === e.id}
              onToggleWatched={() => setState(e.id, { watched: !e.watched }, e.watched ? 'Épisode marqué comme non vu' : 'Épisode marqué comme vu')} />
          ))}
        </div>
      </section>
      <div className="mt-12"><Cast people={d.people} /></div>
    </div>
  )
}

function EpisodeRow({ ep, focused, onToggleWatched }: { ep: Card; focused: boolean; onToggleWatched: () => void }) {
  const ref = useRef<HTMLDivElement>(null)
  const [menu, setMenu] = useState(false)
  useEffect(() => { if (focused) ref.current?.scrollIntoView({ block: 'center' }) }, [focused])
  const d = ep.duration || (ep.runtime ?? 0) * 60
  const p = ep.position && d ? ep.position / d : 0
  return (
    <div ref={ref} className={cx('group flex gap-4 sm:gap-6 rounded-xl p-2.5 sm:p-3 transition-colors hover:bg-white/[0.03]', focused && 'ring-1 ring-aurora/40 bg-aurora/[0.03]')}>
      <Link to={playLink(ep)} className="relative w-36 sm:w-56 shrink-0 aspect-video rounded-lg overflow-hidden bg-coal" aria-label={`Lire ${ep.title}`}>
        <Artwork src={ep.thumb} title={ep.title} sizes="wide" />
        <span className="absolute inset-0 grid place-items-center bg-ink/40 opacity-0 group-hover:opacity-100 transition-opacity">
          <span className="grid size-11 place-items-center rounded-full bg-paper text-ink"><Play className="size-5 fill-current ml-0.5" /></span>
        </span>
        {p > 0 && <ProgressBar value={p} className="absolute bottom-0 inset-x-0 rounded-none h-[3px]" />}
      </Link>
      <div className="min-w-0 flex-1 py-1">
        <div className="flex items-start gap-3">
          <div className="min-w-0 flex-1">
            <div className="text-[11px] font-semibold tracking-wide text-ash">
              {episodeCode(ep.season, ep.episode, ep.episodeEnd)}{ep.episodeEnd ? ' · épisode double' : ''}
            </div>
            <h3 className="font-medium text-[15px] leading-snug mt-0.5">{ep.title}</h3>
          </div>
          {ep.watched && <Check className="size-4 text-aurora shrink-0 mt-1" aria-label="Vu" />}
          <div className="relative">
            <button aria-label="Options de l'épisode" onClick={() => setMenu((m) => !m)} className="grid size-8 place-items-center rounded-full text-ash hover:text-paper hover:bg-white/10">
              <MoreHorizontal className="size-4" />
            </button>
            {menu && (
              <div className="absolute right-0 mt-1 z-10 w-52 rounded-lg bg-coal ring-1 ring-white/10 p-1 text-sm shadow-[var(--shadow-lift)]" onMouseLeave={() => setMenu(false)}>
                <button className="w-full text-left px-3 py-2 rounded hover:bg-white/[0.06]" onClick={() => { setMenu(false); onToggleWatched() }}>
                  {ep.watched ? 'Marquer comme non vu' : 'Marquer comme vu'}
                </button>
                {p > 0 && <Link className="block px-3 py-2 rounded hover:bg-white/[0.06]" to={playLink(ep, { restart: true })}>Lire depuis le début</Link>}
              </div>
            )}
          </div>
        </div>
        {ep.overview && <p className="text-sm text-ash leading-relaxed mt-1.5 line-clamp-2">{ep.overview}</p>}
        <div className="text-xs text-ash mt-2 flex gap-3">
          {d > 0 && <span>{runtime(ep.runtime, ep.duration)}</span>}
          {p > 0 && <span className="text-aurora/90">Reprise à {clock(ep.position ?? 0)}</span>}
          {ep.versions && ep.versions > 1 && <span>{ep.versions} versions</span>}
        </div>
      </div>
    </div>
  )
}
