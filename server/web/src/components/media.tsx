// Cartes, rangées et bandeau : le cœur visuel de la médiathèque.
import { useRef, useState, type ReactNode } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Check, ChevronLeft, ChevronRight, Heart, Info, Play, Plus, X } from 'lucide-react'
import type { Card } from '../lib/types'
import { episodeCode, remaining, runtime } from '../lib/format'
import { Badge, Button, ProgressBar, cx } from './ui'

export function itemLink(c: Card): string {
  if (c.kind === 'collection') return `/collections/${c.id}`
  if (c.kind === 'series') return `/series/${c.id}`
  if (c.kind === 'episode') return `/series/${c.seriesId}?season=${c.season}&episode=${c.id}`
  return `/movies/${c.id}`
}

export function playLink(c: Card, opts?: { restart?: boolean; file?: number }): string {
  const q = new URLSearchParams()
  if (opts?.restart) q.set('restart', '1')
  if (opts?.file) q.set('file', String(opts.file))
  const s = q.toString()
  return `/play/${c.id}${s ? '?' + s : ''}`
}

/**
 * Image avec repli. Sans affiche (pas de fiche TMDB), on compose une affiche « maison » : une image nette
 * tirée de la vidéo dans le haut, le titre en grand dessous, sur un fond de salle teinté d'aurore.
 */
export function Artwork({ src, fallbackSrc, title, subtitle, className, sizes = 'poster' }: {
  src?: string; fallbackSrc?: string; title: string; subtitle?: string; className?: string; sizes?: 'poster' | 'wide'
}) {
  const [failed, setFailed] = useState(false)
  const [fbFailed, setFbFailed] = useState(false)
  const url = !failed && src ? src : undefined
  if (url) {
    return <img src={url} alt="" loading="lazy" decoding="async" onError={() => setFailed(true)}
      className={cx('size-full object-cover', className)} />
  }
  if (sizes === 'wide') {
    return (
      <div className={cx('relative size-full overflow-hidden bg-coal', className)}>
        {fallbackSrc && !fbFailed && (
          <img src={fallbackSrc} alt="" loading="lazy" decoding="async" onError={() => setFbFailed(true)} className="absolute inset-0 size-full object-cover" />
        )}
      </div>
    )
  }
  const hasStill = fallbackSrc && !fbFailed
  return (
    <div className={cx('relative size-full overflow-hidden bg-[radial-gradient(120%_70%_at_20%_0%,rgb(126_232_198/0.10),transparent_60%),linear-gradient(180deg,#191c21,#0c0d10)]', className)}>
      {hasStill && (
        <div className="absolute inset-x-0 top-0 h-[56%]">
          <img src={fallbackSrc} alt="" loading="lazy" decoding="async" onError={() => setFbFailed(true)}
            className="size-full object-cover saturate-[0.9]" />
          <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-[#131519]" />
        </div>
      )}
      <div className={cx('absolute inset-x-0 bottom-0 flex flex-col p-3.5', hasStill ? 'top-[50%] justify-end' : 'top-0 justify-center')}>
        <span className="block h-px w-8 bg-aurora/70 mb-2.5" aria-hidden />
        <span className={cx('font-display leading-[1.08] text-paper', hasStill ? 'text-[16px] line-clamp-3' : 'text-[21px] line-clamp-5')}>{title}</span>
        {subtitle && <span className="mt-1.5 text-[10.5px] tracking-[0.18em] uppercase text-ash">{subtitle}</span>}
      </div>
    </div>
  )
}

function progressOf(c: Card): number {
  const d = c.duration || (c.runtime ? c.runtime * 60 : 0)
  return c.position && d ? c.position / d : 0
}

/** Affiche verticale (films, séries, collections). */
export function PosterCard({ card, width = 'w-[148px] sm:w-[172px]' }: { card: Card; width?: string }) {
  const p = progressOf(card)
  return (
    <Link to={itemLink(card)} className={cx('group block shrink-0 focus-visible:outline-none', width)} aria-label={card.title}>
      <div className="relative aspect-[2/3] rounded-[var(--radius-card)] overflow-hidden bg-coal ring-1 ring-white/[0.06] transition-[transform,box-shadow] duration-300 ease-[var(--ease-cinema)] group-hover:-translate-y-1 group-hover:shadow-[var(--shadow-lift)] group-focus-visible:ring-2 group-focus-visible:ring-aurora">
        <Artwork src={card.poster} fallbackSrc={card.thumb} title={card.title} subtitle={card.year ? String(card.year) : undefined} />
        <div className="absolute top-2 right-2 flex gap-1">
          {card.watched && <span className="grid size-6 place-items-center rounded-full bg-ink/80 text-aurora" title="Vu"><Check className="size-3.5" /></span>}
          {card.kind === 'series' && !!card.unwatched && card.unwatched > 0 && card.unwatched < (card.total ?? 0) && (
            <span className="h-6 min-w-6 px-1.5 grid place-items-center rounded-full bg-aurora text-ink text-[11px] font-bold" title={`${card.unwatched} épisodes non vus`}>{card.unwatched}</span>
          )}
        </div>
        {card.kind === 'collection' && (
          <div className="absolute bottom-0 inset-x-0 p-3 bg-gradient-to-t from-ink to-transparent">
            <span className="eyebrow !text-aurora">Collection · {card.total}</span>
          </div>
        )}
        {p > 0 && <ProgressBar value={p} className="absolute bottom-2 inset-x-2" />}
      </div>
      {card.kind !== 'collection' && (
        <div className="mt-2.5 px-0.5">
          <div className="text-[13.5px] font-medium text-paper truncate group-hover:text-white">{card.title}</div>
          <div className="text-xs text-ash truncate">
            {[card.year, card.kind === 'series' && card.total ? `${card.total} épisodes` : null].filter(Boolean).join(' · ')}
          </div>
        </div>
      )}
    </Link>
  )
}

/** Carte large (reprise, épisodes) avec bouton de lecture direct. */
export function WideCard({ card, width = 'w-[260px] sm:w-[320px]', onRemove }: { card: Card; width?: string; onRemove?: () => void }) {
  const p = progressOf(card)
  const isEp = card.kind === 'episode'
  const img = isEp ? card.thumb : card.backdrop || card.thumb
  return (
    <div className={cx('group relative shrink-0', width)}>
      {onRemove && (
        // « Retirer de Continuer à regarder » : visible au survol (souris), toujours sur écran tactile.
        <button onClick={onRemove} aria-label={`Retirer « ${card.seriesTitle ?? card.title} » de Continuer à regarder`}
          title="Retirer de Continuer à regarder (le fichier n'est pas supprimé)"
          className="absolute top-2 right-2 z-10 grid size-8 place-items-center rounded-full bg-ink/75 text-mist backdrop-blur ring-1 ring-white/10 hover:text-paper hover:bg-ink transition-opacity opacity-100 [@media(hover:hover)]:opacity-0 group-hover:opacity-100 focus-visible:opacity-100">
          <X className="size-4" />
        </button>
      )}
      <Link to={playLink(card)} className="relative block aspect-video rounded-[var(--radius-card)] overflow-hidden bg-coal ring-1 ring-white/[0.06] focus-visible:ring-2 focus-visible:ring-aurora" aria-label={`Lire ${card.title}`}>
        <Artwork src={img} fallbackSrc={card.thumb} title={card.title} sizes="wide" className="transition-transform duration-500 group-hover:scale-[1.03]" />
        <div className="absolute inset-0 bg-gradient-to-t from-ink/90 via-transparent to-transparent" />
        <span className="absolute left-3 bottom-3 grid size-10 place-items-center rounded-full bg-paper/95 text-ink opacity-90 transition-transform group-hover:scale-110">
          <Play className="size-4 fill-current ml-0.5" />
        </span>
        {p > 0 && <ProgressBar value={p} className="absolute bottom-0 inset-x-0 rounded-none h-[3px]" />}
      </Link>
      <Link to={itemLink(card)} className="block mt-2.5 px-0.5">
        {isEp && <div className="text-[11px] text-aurora/90 font-semibold tracking-wide">{card.seriesTitle} · {episodeCode(card.season, card.episode, card.episodeEnd)}</div>}
        <div className="text-[13.5px] font-medium truncate">{card.title}</div>
        <div className="text-xs text-ash">
          {p > 0 ? remaining(card.position ?? 0, card.duration || (card.runtime ?? 0) * 60) : runtime(card.runtime, card.duration)}
        </div>
      </Link>
    </div>
  )
}

/** Rangée horizontale avec flèches (souris) et défilement tactile natif. */
export function Rail({ title, children, action, wide }: { title: string; children: ReactNode; action?: ReactNode; wide?: boolean }) {
  const ref = useRef<HTMLDivElement>(null)
  const scroll = (dir: number) => ref.current?.scrollBy({ left: dir * ref.current.clientWidth * 0.85, behavior: 'smooth' })
  return (
    <section className="relative group/rail" aria-label={title}>
      <div className="flex items-end justify-between px-[var(--gutter)] mb-3">
        <h2 className="font-display text-[22px] sm:text-[26px] tracking-[-0.01em]">{title}</h2>
        <div className="flex items-center gap-1">
          {action}
          <button aria-label="Précédent" onClick={() => scroll(-1)} className="hidden md:grid size-8 place-items-center rounded-full text-ash hover:text-paper hover:bg-white/10"><ChevronLeft className="size-5" /></button>
          <button aria-label="Suivant" onClick={() => scroll(1)} className="hidden md:grid size-8 place-items-center rounded-full text-ash hover:text-paper hover:bg-white/10"><ChevronRight className="size-5" /></button>
        </div>
      </div>
      <div ref={ref} className={cx('rail', wide && 'gap-4')}>{children}</div>
    </section>
  )
}

/** Bandeau principal : fond immersif, bandes « scope » et appel à la lecture évident. */
export function Hero({ card, onToggleList }: { card: Card; onToggleList?: () => void }) {
  const nav = useNavigate()
  const p = progressOf(card)
  const bg = card.backdrop || card.thumb
  return (
    <section className="relative min-h-[68vh] sm:min-h-[78vh] flex items-end overflow-hidden" aria-label="À l'affiche">
      {bg && <img src={bg} alt="" className="absolute inset-0 size-full object-cover object-[center_25%]" />}
      <div className="absolute inset-0 hero-fade" />
      <div className="absolute -top-1/3 -left-1/4 w-[120%] h-full aurora-ribbon pointer-events-none" aria-hidden />
      <div className="relative w-full px-[var(--gutter)] pb-14 sm:pb-20 pt-32 max-w-[min(100%,980px)]">
        <p className="eyebrow mb-4 rise">{p > 0 ? 'Reprendre' : "Ce soir à l'affiche"}</p>
        <h1 className="title-xl text-[44px] sm:text-[72px] lg:text-[88px] mb-4 rise [animation-delay:60ms]">{card.title}</h1>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-mist mb-5 rise [animation-delay:120ms]">
          {card.year && <span>{card.year}</span>}
          {(card.runtime || card.duration) && <span>{runtime(card.runtime, card.duration)}</span>}
          {card.genres?.slice(0, 3).map((g) => <span key={g} className="text-ash">{g}</span>)}
          {card.resolution && <Badge>{card.resolution}</Badge>}
          {card.hdr && <Badge tone="accent">{card.hdr.includes('Dolby') ? 'DOLBY VISION' : card.hdr}</Badge>}
        </div>
        {card.overview && <p className="max-w-xl text-[15px] leading-relaxed text-mist line-clamp-3 mb-7 rise [animation-delay:180ms]">{card.overview}</p>}
        <div className="flex flex-wrap items-center gap-3 rise [animation-delay:240ms]">
          <Button variant="primary" size="lg" icon={<Play className="size-5 fill-current" />} onClick={() => nav(playLink(card))}>
            {p > 0 ? 'Reprendre' : 'Lecture'}
          </Button>
          <Button size="lg" icon={<Info className="size-5" />} onClick={() => nav(itemLink(card))}>Détails</Button>
          {onToggleList && (
            <Button size="lg" variant="ghost" icon={card.watchlist ? <Check className="size-5 text-aurora" /> : <Plus className="size-5" />} onClick={onToggleList}>
              {card.watchlist ? 'Dans ma liste' : 'Ma liste'}
            </Button>
          )}
        </div>
        {p > 0 && <ProgressBar value={p} className="mt-6 max-w-xs" />}
      </div>
    </section>
  )
}

export function FavoriteButton({ active, onClick }: { active: boolean; onClick: () => void }) {
  return (
    <Button size="lg" variant="ghost" onClick={onClick} aria-pressed={active}
      icon={<Heart className={cx('size-5', active && 'fill-aurora text-aurora')} />}>
      {active ? 'Favori' : 'Favoris'}
    </Button>
  )
}

export function PosterGrid({ cards }: { cards: Card[] }) {
  return (
    <div className="grid gap-x-4 gap-y-7 px-[var(--gutter)] grid-cols-[repeat(auto-fill,minmax(132px,1fr))] sm:grid-cols-[repeat(auto-fill,minmax(160px,1fr))]">
      {cards.map((c) => <div key={c.id} className="cv-auto"><PosterCard card={c} width="w-full" /></div>)}
    </div>
  )
}
