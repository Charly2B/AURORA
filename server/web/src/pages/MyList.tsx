import { Bookmark } from 'lucide-react'
import { useApi, useDocumentTitle } from '../lib/hooks'
import type { Card } from '../lib/types'
import { PosterCard, Rail } from '../components/media'
import { EmptyState, ErrorState, PageLoader } from '../components/ui'
import { PageHeader } from '../components/Layout'

type List = { items: Card[]; total: number }

export function MyList() {
  useDocumentTitle('Ma liste')
  const movies = useApi<List>('/movies?watchlist=1&sort=added&order=desc&limit=200')
  const series = useApi<List>('/series?watchlist=1&sort=added&order=desc&limit=200')
  const favMovies = useApi<List>('/movies?favorite=1&sort=title&limit=200')
  const favSeries = useApi<List>('/series?favorite=1&sort=title&limit=200')
  const err = movies.error ?? series.error
  if (err) return <div className="pt-24"><ErrorState error={err} retry={() => { movies.reload(); series.reload() }} /></div>
  if (!movies.data || !series.data) return <PageLoader />
  const empty = !movies.data.total && !series.data.total && !favMovies.data?.total && !favSeries.data?.total
  return (
    <div>
      <PageHeader eyebrow="À voir et à revoir" title="Ma liste" />
      {empty && (
        <EmptyState icon={<Bookmark className="size-6" />} title="Votre liste est vide">
          Ajoutez un film ou une série avec le bouton « Ma liste » de sa fiche pour le retrouver ici.
        </EmptyState>
      )}
      <div className="space-y-10">
        {movies.data.total > 0 && <Rail title="Films à voir">{movies.data.items.map((c) => <PosterCard key={c.id} card={c} />)}</Rail>}
        {series.data.total > 0 && <Rail title="Séries à voir">{series.data.items.map((c) => <PosterCard key={c.id} card={c} />)}</Rail>}
        {!!favMovies.data?.total && <Rail title="Films favoris">{favMovies.data.items.map((c) => <PosterCard key={c.id} card={c} />)}</Rail>}
        {!!favSeries.data?.total && <Rail title="Séries favorites">{favSeries.data.items.map((c) => <PosterCard key={c.id} card={c} />)}</Rail>}
      </div>
    </div>
  )
}
