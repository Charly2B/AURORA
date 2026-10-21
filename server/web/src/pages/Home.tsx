import { Link } from 'react-router-dom'
import { Film, FolderPlus, Tv } from 'lucide-react'
import { api } from '../lib/api'
import { useApi, useDocumentTitle } from '../lib/hooks'
import { useSession } from '../lib/session'
import type { Card, LibraryRef, Section } from '../lib/types'
import { Hero, PosterCard, Rail, WideCard } from '../components/media'
import { Button, EmptyState, ErrorState, Skeleton, useToast } from '../components/ui'

interface HomeResp { hero?: Card; sections: Section[] }

const WIDE = new Set(['continue', 'nextup'])

export function Home() {
  useDocumentTitle()
  const { user } = useSession()
  const home = useApi<HomeResp>('/home')
  const libs = useApi<LibraryRef[]>('/libraries')
  const toast = useToast()

  if (home.error) return <div className="pt-24"><ErrorState error={home.error} retry={home.reload} /></div>
  if (!home.data) return <HomeSkeleton />
  const { hero, sections } = home.data

  if (!hero && sections.length === 0) {
    return (
      <div className="pt-32">
        <EmptyState icon={<FolderPlus className="size-6" />} title="La salle est encore vide"
          action={user?.role === 'admin' ? <Link to="/admin/libraries"><Button variant="primary">Gérer les bibliothèques</Button></Link> : undefined}>
          {user?.role === 'admin'
            ? "Aucune vidéo n'a encore été indexée. Ajoutez un dossier de films ou de séries, ou patientez pendant l'analyse en cours."
            : "Aucune vidéo n'est disponible pour votre profil pour le moment."}
        </EmptyState>
      </div>
    )
  }

  const toggleList = async () => {
    if (!hero) return
    try {
      await api.post(`/items/${hero.id}/state`, { watchlist: !hero.watchlist })
      home.setData({ ...home.data!, hero: { ...hero, watchlist: !hero.watchlist } })
      toast('ok', hero.watchlist ? 'Retiré de ma liste' : 'Ajouté à ma liste')
    } catch (e) {
      toast('error', (e as Error).message)
    }
  }

  // Retire l'œuvre (ou la série de l'épisode) des rangées Continuer / Prochain épisode ; fichier et progression intacts.
  const hideContinue = async (c: Card) => {
    try {
      await api.post(`/items/${c.id}/continue/hide`)
      toast('ok', `« ${c.kind === 'episode' ? c.seriesTitle ?? c.title : c.title} » retiré de Continuer à regarder`)
      home.reload()
    } catch (e) {
      toast('error', (e as Error).message)
    }
  }

  return (
    <div>
      {hero ? <Hero card={hero} onToggleList={toggleList} /> : <div className="h-24" />}
      <div className={hero ? '-mt-10 relative space-y-10' : 'space-y-10'}>
        {sections.map((s) => (
          <Rail key={s.id} title={s.title} wide={WIDE.has(s.id)}
            action={s.libraryId ? <Link to={`/${s.kind === 'series' ? 'series' : 'movies'}?library=${s.libraryId}`}
              className="text-xs text-ash hover:text-aurora mr-2">Tout voir</Link> : undefined}>
            {s.items.map((c) => (WIDE.has(s.id) ? <WideCard key={c.id} card={c} onRemove={() => hideContinue(c)} /> : <PosterCard key={c.id} card={c} />))}
          </Rail>
        ))}
        {libs.data && libs.data.length > 1 && (
          <section className="px-[var(--gutter)]" aria-label="Bibliothèques">
            <h2 className="font-display text-[22px] sm:text-[26px] mb-3">Bibliothèques</h2>
            <div className="flex flex-wrap gap-3">
              {libs.data.flatMap((l) => [
                ...(l.movies > 0 ? [{ key: `m${l.id}`, to: `/movies?library=${l.id}`, icon: <Film className="size-5 text-aurora" />, n: l.movies, label: 'films' }] : []),
                ...(l.series > 0 ? [{ key: `s${l.id}`, to: `/series?library=${l.id}`, icon: <Tv className="size-5 text-aurora" />, n: l.series, label: 'séries' }] : []),
              ].map((x) => (
                <Link key={x.key} to={x.to} className="flex items-center gap-3 rounded-xl bg-coal px-5 py-4 ring-1 ring-white/5 hover:ring-aurora/40 transition-shadow">
                  {x.icon}
                  <span className="font-medium">{l.name}</span>
                  <span className="text-xs text-ash tabular-nums">{x.n} {x.label}</span>
                </Link>
              )))}
            </div>
          </section>
        )}
      </div>
    </div>
  )
}

function HomeSkeleton() {
  return (
    <div aria-busy="true">
      <Skeleton className="h-[68vh] !rounded-none" />
      <div className="px-[var(--gutter)] mt-8 flex gap-4 overflow-hidden">
        {Array.from({ length: 8 }, (_, i) => <Skeleton key={i} className="w-[172px] aspect-[2/3] shrink-0" />)}
      </div>
    </div>
  )
}
