import { useState } from 'react'
import { Download, RotateCcw, Save } from 'lucide-react'
import { api } from '../../lib/api'
import { useApi } from '../../lib/hooks'
import { bytes, date } from '../../lib/format'
import { Button, ErrorState, Modal, Notice, PageLoader, Spinner, useToast } from '../../components/ui'
import { Line, PHASES } from './Admin'

interface Task { id: number; kind: string; library: string; status: string; startedAt: number; endedAt: number; total: number; done: number; errors: number; message: string }
interface Backup { name: string; size: number; modTime: number }
interface Diag {
  version: string; go: string; os: string; cpus: number; goroutines: number; memoryMB: number; heapMB: number; dataDir: string; tmpDir: string
  browseRoots: string[]; listen: string; dbIntegrity: string; schemaVersion: number; dbSize: number
  tools: { ffmpeg: string; ffprobe: string; ffmpegVersion: string; ffprobeVersion: string }; tonemap: boolean
  watch: { active: boolean; dirs?: number; limited?: boolean; unavailable?: boolean }
  limits: { maxTranscodes: number; maxRemux: number; transcodeThreads: number }
}
const STATUS: Record<string, string> = { running: 'en cours', done: 'terminée', failed: 'échec', cancelled: 'interrompue', interrupted: 'interrompue (arrêt du serveur)' }

export function Maintenance() {
  return (
    <div className="max-w-5xl space-y-10">
      <Tasks />
      <Backups />
      <Diagnostics />
      <Logs />
    </div>
  )
}

function Tasks() {
  const t = useApi<{ tasks: Task[] }>('/admin/tasks')
  const [errs, setErrs] = useState<{ id: number; list?: { path: string; message: string }[] }>()
  const open = async (id: number) => {
    setErrs({ id })
    setErrs({ id, list: await api.get(`/admin/tasks/${id}/errors`) })
  }
  return (
    <section className="space-y-3">
      <h2 className="font-display text-2xl">Historique des analyses</h2>
      {t.error && <ErrorState error={t.error} retry={t.reload} />}
      {!t.data ? <Spinner /> : t.data.tasks.length === 0 ? <Notice>Aucune analyse pour l'instant.</Notice> : (
        <div className="overflow-x-auto rounded-2xl ring-1 ring-white/[0.06]">
          <table className="w-full text-sm">
            <thead className="text-left text-ash"><tr className="border-b border-white/[0.06]"><th className="p-3 font-medium">Bibliothèque</th><th className="p-3 font-medium">Début</th><th className="p-3 font-medium">État</th><th className="p-3 font-medium">Fichiers</th><th className="p-3 font-medium">Erreurs</th></tr></thead>
            <tbody>
              {t.data.tasks.map((x) => (
                <tr key={x.id} className="border-b border-white/[0.03] last:border-0">
                  <td className="p-3">{x.library || x.kind}</td>
                  <td className="p-3 text-ash whitespace-nowrap">{date(x.startedAt)}</td>
                  <td className="p-3">{STATUS[x.status] ?? x.status}{x.message && <div className="text-xs text-ash">{x.message}</div>}</td>
                  <td className="p-3 tabular-nums">{x.done}{x.total ? ` / ${x.total}` : ''}</td>
                  <td className="p-3">{x.errors > 0 ? <button className="text-ember underline" onClick={() => open(x.id)}>{x.errors}</button> : '0'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <Modal open={!!errs} onClose={() => setErrs(undefined)} title="Fichiers en erreur" wide>
        {!errs?.list ? <Spinner /> : (
          <ul className="space-y-3 max-h-[60vh] overflow-y-auto text-sm">
            {errs.list.map((e, i) => <li key={i}><div className="font-mono text-xs break-all">{e.path}</div><div className="text-ash">{e.message}</div></li>)}
          </ul>
        )}
      </Modal>
      <p className="text-xs text-ash">Phases : {Object.values(PHASES).join(' → ')}.</p>
    </section>
  )
}

function Backups() {
  const b = useApi<Backup[]>('/admin/backups')
  const toast = useToast()
  const [busy, setBusy] = useState(false)
  const [restore, setRestore] = useState<Backup>()
  const create = async () => {
    setBusy(true)
    try { await api.post('/admin/backups'); toast('ok', 'Sauvegarde créée'); b.reload() } catch (e) { toast('error', (e as Error).message) }
    setBusy(false)
  }
  const doRestore = async () => {
    try {
      const r = await api.post<{ message: string }>(`/admin/backups/${encodeURIComponent(restore!.name)}/restore`)
      toast('info', r.message)
      setRestore(undefined)
      b.reload()
    } catch (e) { toast('error', (e as Error).message) }
  }
  return (
    <section className="space-y-3">
      <div className="flex justify-between items-center gap-4">
        <h2 className="font-display text-2xl">Sauvegardes</h2>
        <Button icon={<Save className="size-4" />} busy={busy} onClick={create}>Sauvegarder maintenant</Button>
      </div>
      <p className="text-sm text-ash">La base (catalogue, profils, historique, réglages) est copiée de façon cohérente. Une sauvegarde est aussi faite automatiquement avant chaque mise à jour du schéma. Les images en cache se retéléchargent.</p>
      {b.error && <ErrorState error={b.error} retry={b.reload} />}
      {b.data && b.data.length === 0 && <Notice>Aucune sauvegarde.</Notice>}
      {b.data && b.data.length > 0 && (
        <ul className="divide-y divide-white/[0.05] rounded-2xl ring-1 ring-white/[0.06]">
          {b.data.map((x) => (
            <li key={x.name} className="flex flex-wrap items-center gap-4 px-4 py-3 text-sm">
              <span className="font-mono text-xs flex-1 min-w-48 break-all">{x.name}</span>
              <span className="text-ash">{bytes(x.size)}</span>
              <span className="text-ash">{date(x.modTime)}</span>
              <a href={`/api/v1/admin/backups/${encodeURIComponent(x.name)}`} className="inline-flex items-center gap-1.5 text-aurora" download><Download className="size-4" /> Télécharger</a>
              <Button size="sm" variant="ghost" icon={<RotateCcw className="size-4" />} onClick={() => setRestore(x)}>Restaurer</Button>
            </li>
          ))}
        </ul>
      )}
      <Modal open={!!restore} onClose={() => setRestore(undefined)} title="Restaurer cette sauvegarde ?">
        <p className="text-mist mb-3">La base sera remplacée par « {restore?.name} » au prochain démarrage d'AURORA.</p>
        <p className="text-sm text-ash mb-6">L'état actuel est d'abord sauvegardé, et l'ancienne base est conservée sur le disque : rien n'est effacé. Après validation, redémarrez AURORA depuis le Centre de paquets.</p>
        <div className="flex justify-end gap-2"><Button variant="ghost" onClick={() => setRestore(undefined)}>Annuler</Button><Button variant="primary" onClick={doRestore}>Programmer la restauration</Button></div>
      </Modal>
    </section>
  )
}

function Diagnostics() {
  const d = useApi<Diag>('/admin/diagnostics')
  if (d.error) return <ErrorState error={d.error} retry={d.reload} />
  if (!d.data) return <PageLoader />
  const x = d.data
  return (
    <section className="space-y-3">
      <h2 className="font-display text-2xl">Diagnostic</h2>
      <div className="grid md:grid-cols-2 gap-4 text-sm">
        <div className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-5 space-y-2">
          <Line k="Version" v={x.version} />
          <Line k="Système" v={`${x.os} · ${x.cpus} cœurs logiques · ${x.go}`} />
          <Line k="Mémoire du serveur" v={`${x.heapMB} Mo utilisés · ${x.memoryMB} Mo réservés`} />
          <Line k="Écoute" v={x.listen} />
          <Line k="Données" v={x.dataDir} />
          <Line k="Temporaire" v={x.tmpDir} />
          <Line k="Volumes explorables" v={x.browseRoots.join(', ')} />
        </div>
        <div className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-5 space-y-2">
          <Line k="Base de données" v={`${bytes(x.dbSize)} · schéma v${x.schemaVersion}`} />
          <Line k="Intégrité" v={<span className={x.dbIntegrity === 'ok' ? 'text-aurora' : 'text-danger'}>{x.dbIntegrity}</span>} />
          <Line k="ffmpeg" v={x.tools.ffmpeg ? x.tools.ffmpegVersion || x.tools.ffmpeg : <span className="text-danger">absent</span>} />
          <Line k="ffprobe" v={x.tools.ffprobe ? x.tools.ffprobe : <span className="text-danger">absent</span>} />
          <Line k="Conversion HDR → SDR" v={x.tonemap ? 'disponible' : 'indisponible'} />
          <Line k="Détection immédiate des ajouts" v={x.watch.unavailable ? <span className="text-danger">indisponible (analyse périodique seulement)</span>
            : !x.watch.active ? 'désactivée' : x.watch.limited ? <span className="text-ember">partielle : {x.watch.dirs} dossiers (limite du système atteinte)</span>
            : `active (${x.watch.dirs} dossiers surveillés)`} />
          <Line k="Limites" v={`${x.limits.maxTranscodes} conversion(s) · ${x.limits.maxRemux} remux · ${x.limits.transcodeThreads} fils`} />
        </div>
      </div>
    </section>
  )
}

function Logs() {
  const l = useApi<{ text: string }>('/admin/logs')
  return (
    <section className="space-y-3">
      <div className="flex justify-between items-center"><h2 className="font-display text-2xl">Journal</h2><Button size="sm" onClick={l.reload}>Actualiser</Button></div>
      {l.error && <ErrorState error={l.error} retry={l.reload} />}
      <pre className="max-h-96 overflow-auto rounded-2xl bg-black/60 ring-1 ring-white/[0.06] p-4 text-[11.5px] leading-relaxed text-mist whitespace-pre-wrap break-all">
        {l.data?.text.split('\n').slice(-400).join('\n') || 'Journal vide.'}
      </pre>
      <p className="text-xs text-ash">Le journal ne contient ni mot de passe ni clé d'API.</p>
    </section>
  )
}
