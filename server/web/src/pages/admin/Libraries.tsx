import { useState } from 'react'
import { ArrowLeftRight, Film, FolderX, Pencil, Plus, RefreshCw, Trash2, Tv } from 'lucide-react'
import { api, ApiError } from '../../lib/api'
import { useApi } from '../../lib/hooks'
import { relative } from '../../lib/format'
import { FolderPicker } from '../../components/FolderPicker'
import { Button, ErrorState, Field, Modal, Notice, PageLoader, cx, useToast } from '../../components/ui'

interface Lib {
  id: number; name: string; kind: 'movies' | 'series'; path: string; lastScanAt: number; lastScanStatus: string
  files: number; missing: number; unidentified: number; pending: number; accessible: boolean
  suggestKind?: 'movies' | 'series'
}

const STATUS: Record<string, string> = { done: 'terminée', failed: 'en échec', cancelled: 'interrompue', interrupted: 'interrompue' }

export function Libraries() {
  const libs = useApi<Lib[]>('/admin/libraries')
  const toast = useToast()
  const [adding, setAdding] = useState(false)
  const [kind, setKind] = useState<'movies' | 'series'>('movies')
  const [path, setPath] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const [removing, setRemoving] = useState<Lib>()
  const [renaming, setRenaming] = useState<Lib>()
  const [retyping, setRetyping] = useState<Lib>()
  const [suggest, setSuggest] = useState<'movies' | 'series'>()

  if (libs.error) return <ErrorState error={libs.error} retry={libs.reload} />
  if (!libs.data) return <PageLoader />

  const add = async (k = kind, force = false) => {
    setBusy(true)
    setError(undefined)
    setSuggest(undefined)
    try {
      await api.post('/admin/libraries', { kind: k, path, name: name || path.split('/').pop(), force })
      toast('ok', 'Bibliothèque ajoutée, analyse lancée')
      setAdding(false); setPath(''); setName('')
      libs.reload()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
      // Le contenu ne correspond pas au type choisi : on propose le bon type plutôt que de deviner.
      if (e instanceof ApiError && e.status === 409 && e.data?.suggestKind) setSuggest(e.data.suggestKind as 'movies' | 'series')
    }
    setBusy(false)
  }
  const retype = async () => {
    if (!retyping) return
    try {
      await api.put(`/admin/libraries/${retyping.id}`, { kind: retyping.kind === 'movies' ? 'series' : 'movies' })
      toast('ok', 'Type modifié (sauvegarde faite avant), nouvelle identification en cours')
      setRetyping(undefined)
      libs.reload()
    } catch (e) { toast('error', (e as Error).message) }
  }
  const scan = async (id: number) => {
    try { await api.post('/admin/scan', { libraryId: id }); toast('ok', 'Analyse lancée') } catch (e) { toast('error', (e as Error).message) }
  }
  const remove = async () => {
    if (!removing) return
    try {
      await api.del(`/admin/libraries/${removing.id}`)
      toast('ok', 'Bibliothèque retirée du catalogue (aucun fichier touché)')
      setRemoving(undefined)
      libs.reload()
    } catch (e) { toast('error', (e as Error).message) }
  }
  const rename = async () => {
    if (!renaming) return
    try { await api.put(`/admin/libraries/${renaming.id}`, { name: renaming.name }); setRenaming(undefined); libs.reload() } catch (e) { toast('error', (e as Error).message) }
  }

  return (
    <div className="max-w-5xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <p className="text-sm text-ash max-w-2xl">Les dossiers sont lus en lecture seule. AURORA ne renomme, ne déplace et ne supprime jamais vos vidéos.</p>
        <Button variant="primary" icon={<Plus className="size-4" />} onClick={() => setAdding(true)}>Ajouter un dossier</Button>
      </div>
      {libs.data.length === 0 && <Notice>Aucune bibliothèque. Ajoutez un dossier de films ou de séries pour commencer.</Notice>}
      <ul className="space-y-3">
        {libs.data.map((l) => (
          <li key={l.id} className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-5 flex flex-wrap items-center gap-5">
            <div className="grid size-11 place-items-center rounded-xl bg-white/[0.05] text-aurora">{l.kind === 'movies' ? <Film className="size-5" /> : <Tv className="size-5" />}</div>
            <div className="min-w-0 flex-1">
              <div className="font-medium text-lg">{l.name} <span className="text-xs text-ash font-normal ml-1">{l.kind === 'movies' ? 'Films' : 'Séries'}</span></div>
              <div className="text-xs text-ash font-mono truncate">{l.path}</div>
              <div className="text-xs text-ash mt-1.5 flex flex-wrap gap-x-4 gap-y-1">
                <span>{l.files} fichier(s)</span>
                {l.missing > 0 && <span className="text-ember">{l.missing} introuvable(s)</span>}
                {l.unidentified > 0 && <span className="text-ember">{l.unidentified} non identifié(s)</span>}
                {l.pending > 0 && <span>{l.pending} en attente d'analyse</span>}
                <span>Analyse {relative(l.lastScanAt)}{l.lastScanStatus ? ` (${STATUS[l.lastScanStatus] ?? l.lastScanStatus})` : ''}</span>
              </div>
              {l.suggestKind && (
                <div className="mt-3 flex flex-wrap items-center gap-3 rounded-lg bg-ember/[0.08] ring-1 ring-ember/25 px-3 py-2 text-sm">
                  <span className="text-mist">
                    {l.suggestKind === 'series'
                      ? 'Ces fichiers sont nommés comme des épisodes : cette bibliothèque devrait être de type Séries.'
                      : 'Ces fichiers ne ressemblent pas à des épisodes : cette bibliothèque devrait être de type Films.'}
                  </span>
                  <Button size="sm" variant="primary" onClick={() => setRetyping(l)}>Passer en {l.suggestKind === 'series' ? 'Séries' : 'Films'}</Button>
                </div>
              )}
              {!l.accessible && (
                <p className="mt-2 text-xs text-danger flex items-center gap-1.5"><FolderX className="size-3.5" /> Dossier inaccessible : disque démonté ou permission retirée à l'utilisateur « aurora ».</p>
              )}
            </div>
            <div className="flex gap-1">
              <Button size="sm" icon={<RefreshCw className="size-4" />} onClick={() => scan(l.id)}>Analyser</Button>
              <Button size="sm" variant="ghost" aria-label={l.kind === 'movies' ? 'Changer en bibliothèque de séries' : 'Changer en bibliothèque de films'}
                title={l.kind === 'movies' ? 'Changer en Séries' : 'Changer en Films'} onClick={() => setRetyping(l)}><ArrowLeftRight className="size-4" /></Button>
              <Button size="sm" variant="ghost" aria-label="Renommer" onClick={() => setRenaming(l)}><Pencil className="size-4" /></Button>
              <Button size="sm" variant="ghost" aria-label="Retirer" onClick={() => setRemoving(l)}><Trash2 className="size-4" /></Button>
            </div>
          </li>
        ))}
      </ul>

      <Modal open={adding} onClose={() => setAdding(false)} title="Ajouter un dossier" wide>
        <div className="space-y-4">
          <div className="flex gap-2">
            {(['movies', 'series'] as const).map((k) => (
              <button key={k} onClick={() => setKind(k)} aria-pressed={kind === k}
                className={cx('flex items-center gap-2 h-10 px-4 rounded-full text-sm ring-1', kind === k ? 'ring-aurora bg-aurora/[0.08]' : 'ring-graphite text-ash')}>
                {k === 'movies' ? <Film className="size-4" /> : <Tv className="size-4" />}{k === 'movies' ? 'Films' : 'Séries'}
              </button>
            ))}
          </div>
          <FolderPicker value={path} onChange={(p) => { setPath(p); setName(p.split('/').pop() ?? '') }} />
          {path && <Field label="Nom affiché" value={name} onChange={(e) => setName(e.target.value)} />}
          <Notice>Un dossier grisé n'est pas lisible par AURORA. Dans DSM : Panneau de configuration › Dossier partagé › Modifier › Permissions › « Utilisateurs internes du système » › <b>aurora</b> : lecture seule.</Notice>
          {error && !suggest && <Notice tone="error">{error}</Notice>}
          {suggest && (
            <Notice tone="warn">
              <p className="mb-3">{error}</p>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant="primary" busy={busy} onClick={() => { setKind(suggest); add(suggest) }}>
                  Ajouter comme {suggest === 'series' ? 'Séries' : 'Films'}
                </Button>
                <Button size="sm" variant="ghost" onClick={() => add(kind, true)}>Garder « {kind === 'series' ? 'Séries' : 'Films'} »</Button>
              </div>
            </Notice>
          )}
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setAdding(false)}>Annuler</Button>
            <Button variant="primary" disabled={!path} busy={busy} onClick={() => add()}>Ajouter et analyser</Button>
          </div>
        </div>
      </Modal>

      <Modal open={!!removing} onClose={() => setRemoving(undefined)} title="Retirer la bibliothèque ?">
        <p className="text-mist leading-relaxed mb-2">« {removing?.name} » sera retirée du catalogue d'AURORA, avec l'historique de lecture associé.</p>
        <p className="text-sm text-ash mb-6">Vos fichiers vidéo restent intacts sur le NAS : rien n'est supprimé du disque.</p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setRemoving(undefined)}>Annuler</Button>
          <Button variant="danger" onClick={remove}>Retirer du catalogue</Button>
        </div>
      </Modal>

      <Modal open={!!retyping} onClose={() => setRetyping(undefined)} title={retyping?.kind === 'movies' ? 'Transformer en bibliothèque de séries ?' : 'Transformer en bibliothèque de films ?'}>
        <p className="text-mist leading-relaxed mb-2">
          « {retyping?.name} » sera réidentifiée comme {retyping?.kind === 'movies' ? 'des séries (série › saison › épisode)' : 'des films'}.
        </p>
        <p className="text-sm text-ash mb-6">
          Les vidéos ne sont ni déplacées ni renommées. L'identification actuelle de cette bibliothèque (fiches, progression,
          favoris associés) est remplacée ; l'analyse technique des fichiers est conservée, la nouvelle analyse est donc rapide.
          Une sauvegarde complète de la base est faite juste avant (restaurable dans Maintenance).
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setRetyping(undefined)}>Annuler</Button>
          <Button variant="primary" onClick={retype}>Changer le type</Button>
        </div>
      </Modal>

      <Modal open={!!renaming} onClose={() => setRenaming(undefined)} title="Renommer">
        <Field label="Nom" value={renaming?.name ?? ''} onChange={(e) => renaming && setRenaming({ ...renaming, name: e.target.value })} />
        <div className="flex justify-end gap-2 mt-6">
          <Button variant="ghost" onClick={() => setRenaming(undefined)}>Annuler</Button>
          <Button variant="primary" onClick={rename}>Enregistrer</Button>
        </div>
      </Modal>
    </div>
  )
}
