import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Check, Clapperboard, Eye, EyeOff, Plus, Play, RotateCcw } from 'lucide-react'
import { api } from '../lib/api'
import { useApi, useDocumentTitle } from '../lib/hooks'
import { useSession } from '../lib/session'
import type { Detail } from '../lib/types'
import { clock, date, remaining, runtime } from '../lib/format'
import { ArtworkPicker, Backdrop, Cast, FixMatch, MetaLine, TechInfo } from '../components/detail'
import { Artwork, FavoriteButton, PosterCard, Rail, playLink } from '../components/media'
import { Badge, Button, ErrorState, PageLoader, ProgressBar, Select, useToast } from '../components/ui'

export function MovieDetail() {
  const { id } = useParams()
  const nav = useNavigate()
  const { user } = useSession()
  const toast = useToast()
  const res = useApi<Detail>(`/items/${id}`)
  const d = res.data
  useDocumentTitle(d?.title)
  const [fileId, setFileId] = useState<number>()
  useEffect(() => {
    if (d) setFileId(d.state.lastFileId && d.files.some((f) => f.id === d.state.lastFileId) ? d.state.lastFileId : d.files[0]?.id)
  }, [d])

  if (res.error) return <div className="pt-24"><ErrorState error={res.error} retry={res.reload} /></div>
  if (!d) return <PageLoader />
  const file = d.files.find((f) => f.id === fileId) ?? d.files[0]
  const duration = file?.duration || (d.runtime ?? 0) * 60
  const resumable = d.state.position > 0 && !d.state.watched

  const setState = async (patch: { favorite?: boolean; watchlist?: boolean; watched?: boolean }, msg: string) => {
    try {
      await api.post(`/items/${d.id}/state`, patch)
      res.reload()
      toast('ok', msg)
    } catch (e) {
      toast('error', (e as Error).message)
    }
  }

  return (
    <div>
      <Backdrop src={d.backdrop || d.thumb}>
        <div className="flex flex-col md:flex-row gap-8 md:gap-12 items-start max-w-6xl">
          <div className="w-40 sm:w-56 md:w-64 shrink-0 aspect-[2/3] rounded-xl overflow-hidden ring-1 ring-white/10 shadow-[var(--shadow-lift)] rise">
            <Artwork src={d.poster} fallbackSrc={d.thumb} title={d.title} subtitle={d.year ? String(d.year) : undefined} />
          </div>
          <div className="min-w-0 flex-1 rise [animation-delay:80ms]">
            {d.tagline && <p className="eyebrow mb-3 !normal-case !tracking-normal italic font-display !text-base !text-mist">{d.tagline}</p>}
            <h1 className="title-xl text-[42px] sm:text-[64px] mb-2">{d.title}</h1>
            {d.originalTitle && d.originalTitle !== d.title && <p className="text-ash mb-3">{d.originalTitle}</p>}
            <MetaLine d={d} />
            <div className="flex flex-wrap gap-1.5 mt-3">
              {duration > 0 && <span className="text-sm text-mist mr-2">{runtime(d.runtime, file?.duration)}</span>}
              {file?.resolution && <Badge>{file.resolution}</Badge>}
              {file?.hdr && <Badge tone="accent">{file.hdr}</Badge>}
              {file?.audio.some((a) => a.channels >= 6) && <Badge>5.1+</Badge>}
              {file?.audio.some((a) => a.atmos) && <Badge>ATMOS</Badge>}
            </div>

            <div className="flex flex-wrap items-center gap-3 mt-7">
              <Button variant="primary" size="lg" icon={<Play className="size-5 fill-current" />} onClick={() => nav(playLink(d, { file: file?.id }))} disabled={!file}>
                {resumable ? `Reprendre à ${clock(d.state.position)}` : 'Lecture'}
              </Button>
              {resumable && <Button size="lg" icon={<RotateCcw className="size-4" />} onClick={() => nav(playLink(d, { restart: true, file: file?.id }))}>Depuis le début</Button>}
              <Button size="lg" variant="ghost" icon={d.watchlist ? <Check className="size-5 text-aurora" /> : <Plus className="size-5" />}
                onClick={() => setState({ watchlist: !d.watchlist }, d.watchlist ? 'Retiré de ma liste' : 'Ajouté à ma liste')}>
                Ma liste
              </Button>
              <FavoriteButton active={d.favorite} onClick={() => setState({ favorite: !d.favorite }, d.favorite ? 'Retiré des favoris' : 'Ajouté aux favoris')} />
              {d.trailer && (
                <a href={`https://www.youtube.com/watch?v=${encodeURIComponent(d.trailer)}`} target="_blank" rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 h-12 px-5 rounded-full text-mist hover:text-paper hover:bg-white/[0.06] text-[15px] font-semibold"
                  title="Ouvre YouTube (nécessite Internet)">
                  <Clapperboard className="size-5" /> Bande-annonce
                </a>
              )}
            </div>
            {resumable && (
              <div className="mt-5 max-w-sm">
                <ProgressBar value={d.state.position / (duration || 1)} />
                <p className="text-xs text-ash mt-1.5">{remaining(d.state.position, duration)}</p>
              </div>
            )}
            {d.overview && <p className="mt-7 max-w-2xl text-[15.5px] leading-relaxed text-mist">{d.overview}</p>}
            <div className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-ash">
              <button className="inline-flex items-center gap-1.5 hover:text-paper" onClick={() => setState({ watched: !d.watched }, d.watched ? 'Marqué comme non vu' : 'Marqué comme vu')}>
                {d.watched ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />} {d.watched ? 'Marquer comme non vu' : 'Marquer comme vu'}
              </button>
              {d.state.lastPlayedAt ? <span>Dernière lecture : {date(d.state.lastPlayedAt)}</span> : null}
              {user?.role === 'admin' && <FixMatch item={d} onDone={(nid) => (nid !== d.id ? nav(`/movies/${nid}`, { replace: true }) : res.reload())} />}
              {user?.role === 'admin' && !!d.tmdbId && <ArtworkPicker item={d} onDone={res.reload} />}
            </div>
          </div>
        </div>
      </Backdrop>

      <div className="space-y-12 mt-4">
        {d.files.length > 0 && (
          <section className="px-[var(--gutter)]" aria-label="Informations techniques">
            <div className="flex flex-wrap items-end justify-between gap-4 mb-5">
              <h2 className="font-display text-[24px]">{d.files.length > 1 ? 'Versions' : 'Informations techniques'}</h2>
              {d.files.length > 1 && (
                <Select aria-label="Version à lire" value={file?.id} onChange={(e) => setFileId(Number(e.target.value))}>
                  {d.files.map((f) => <option key={f.id} value={f.id}>{f.label || f.name}</option>)}
                </Select>
              )}
            </div>
            {file && <div className="rounded-2xl bg-night/70 ring-1 ring-white/[0.05] p-6"><TechInfo file={file} /></div>}
          </section>
        )}
        <Cast people={d.people} />
        {d.collections && d.collections.length > 1 && (
          <Rail title="Dans la même collection">{d.collections.map((c) => <PosterCard key={c.id} card={c} />)}</Rail>
        )}
        {d.similar && d.similar.length > 0 && (
          <Rail title="Dans le même esprit">{d.similar.map((c) => <PosterCard key={c.id} card={c} />)}</Rail>
        )}
      </div>
    </div>
  )
}
