import { useState } from 'react'
import { Link } from 'react-router-dom'
import { FileQuestion, HelpCircle } from 'lucide-react'
import { api, ApiError } from '../../lib/api'
import { useApi } from '../../lib/hooks'
import { Button, ErrorState, Field, Modal, Notice, PageLoader, useToast } from '../../components/ui'

interface Unident {
  files: { id: number; path: string; library: string; kind: 'movies' | 'series' }[]
  items: { id: number; kind: 'movie' | 'series'; title: string; year: number; state: string; library: string }[]
}

export function Corrections() {
  const res = useApi<Unident>('/admin/unidentified')
  const [file, setFile] = useState<Unident['files'][number]>()
  if (res.error) return <ErrorState error={res.error} retry={res.reload} />
  if (!res.data) return <PageLoader />
  const { files, items } = res.data
  return (
    <div className="max-w-5xl space-y-10">
      <section className="space-y-4">
        <h2 className="font-display text-2xl">Correspondances incertaines</h2>
        <p className="text-sm text-ash">AURORA n'associe jamais une fiche au hasard : quand plusieurs résultats TMDB correspondent, ou aucun, l'œuvre attend votre choix.</p>
        {items.length === 0 ? <Notice>Aucune œuvre en attente.</Notice> : (
          <ul className="divide-y divide-white/[0.05] rounded-2xl bg-night/70 ring-1 ring-white/[0.06]">
            {items.map((i) => (
              <li key={i.id} className="flex items-center gap-4 px-5 py-3.5">
                <HelpCircle className="size-5 text-ember shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="font-medium truncate">{i.title} {i.year ? <span className="text-ash font-normal">({i.year})</span> : null}</div>
                  <div className="text-xs text-ash">{i.library} · {i.state === 'ambiguous' ? 'plusieurs correspondances possibles' : 'aucune correspondance trouvée'}</div>
                </div>
                <Link to={`/${i.kind === 'movie' ? 'movies' : 'series'}/${i.id}`}><Button size="sm">Corriger</Button></Link>
              </li>
            ))}
          </ul>
        )}
      </section>
      <section className="space-y-4">
        <h2 className="font-display text-2xl">Fichiers non identifiés</h2>
        <p className="text-sm text-ash">Fichiers de séries dont le numéro de saison/épisode n'a pas pu être lu dans le nom. Indiquez-le ici : le fichier n'est pas renommé.</p>
        {files.length === 0 ? <Notice>Tous les fichiers sont identifiés.</Notice> : (
          <ul className="divide-y divide-white/[0.05] rounded-2xl bg-night/70 ring-1 ring-white/[0.06]">
            {files.map((f) => (
              <li key={f.id} className="flex items-center gap-4 px-5 py-3.5">
                <FileQuestion className="size-5 text-ember shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="font-mono text-[13px] truncate">{f.path}</div>
                  <div className="text-xs text-ash">{f.library}</div>
                </div>
                <Button size="sm" onClick={() => setFile(f)}>Identifier</Button>
              </li>
            ))}
          </ul>
        )}
      </section>
      {file && <AssignModal file={file} onClose={() => setFile(undefined)} onDone={() => { setFile(undefined); res.reload() }} />}
    </div>
  )
}

function AssignModal({ file, onClose, onDone }: { file: Unident['files'][number]; onClose: () => void; onDone: () => void }) {
  const guess = file.path.split('/')[0] ?? ''
  const [title, setTitle] = useState(file.kind === 'series' ? guess : '')
  const [year, setYear] = useState('')
  const [season, setSeason] = useState('1')
  const [episode, setEpisode] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const toast = useToast()
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await api.post(`/admin/files/${file.id}/assign`, file.kind === 'series'
        ? { title, season: Number(season), episode: Number(episode) }
        : { title, year: Number(year) || 0 })
      toast('ok', 'Fichier identifié')
      onDone()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    }
    setBusy(false)
  }
  return (
    <Modal open onClose={onClose} title="Identifier le fichier">
      <div className="space-y-4">
        <p className="font-mono text-xs text-ash break-all">{file.path}</p>
        <Field label={file.kind === 'series' ? 'Série' : 'Titre du film'} value={title} onChange={(e) => setTitle(e.target.value)} />
        {file.kind === 'series' ? (
          <div className="grid grid-cols-2 gap-3">
            <Field label="Saison (0 = spéciaux)" inputMode="numeric" value={season} onChange={(e) => setSeason(e.target.value.replace(/\D/g, ''))} />
            <Field label="Épisode" inputMode="numeric" value={episode} onChange={(e) => setEpisode(e.target.value.replace(/\D/g, ''))} />
          </div>
        ) : <Field label="Année" inputMode="numeric" value={year} onChange={(e) => setYear(e.target.value.replace(/\D/g, ''))} />}
        {error && <Notice tone="error">{error}</Notice>}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>Annuler</Button>
          <Button variant="primary" busy={busy} disabled={!title || (file.kind === 'series' && !episode)} onClick={save}>Enregistrer</Button>
        </div>
      </div>
    </Modal>
  )
}
