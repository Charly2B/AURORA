import { useParams } from 'react-router-dom'
import { Library } from 'lucide-react'
import { useApi, useDocumentTitle } from '../lib/hooks'
import type { Card } from '../lib/types'
import { PosterGrid } from '../components/media'
import { EmptyState, ErrorState, PageLoader } from '../components/ui'
import { PageHeader } from '../components/Layout'

export function Collections() {
  useDocumentTitle('Collections')
  const res = useApi<Card[]>('/collections')
  if (res.error) return <div className="pt-24"><ErrorState error={res.error} retry={res.reload} /></div>
  if (!res.data) return <PageLoader />
  return (
    <div>
      <PageHeader eyebrow="Sagas et ensembles" title="Collections" />
      {res.data.length === 0 ? (
        <EmptyState icon={<Library className="size-6" />} title="Aucune collection">
          Les sagas (par exemple une trilogie) apparaissent ici dès que deux films d'une même collection TMDB sont présents.
        </EmptyState>
      ) : <PosterGrid cards={res.data} />}
    </div>
  )
}

export function CollectionPage() {
  const { id } = useParams()
  const res = useApi<{ name: string; items: Card[] }>(`/collections/${id}`)
  useDocumentTitle(res.data?.name)
  if (res.error) return <div className="pt-24"><ErrorState error={res.error} retry={res.reload} /></div>
  if (!res.data) return <PageLoader />
  const bg = res.data.items.find((c) => c.backdrop)?.backdrop
  return (
    <div>
      <div className="relative">
        {bg && <img src={bg} alt="" className="absolute inset-0 size-full object-cover opacity-40" />}
        <div className="absolute inset-0 bg-gradient-to-t from-ink to-ink/40" />
        <div className="relative"><PageHeader eyebrow={`Collection · ${res.data.items.length} films`} title={res.data.name} /></div>
      </div>
      <PosterGrid cards={res.data.items} />
    </div>
  )
}
