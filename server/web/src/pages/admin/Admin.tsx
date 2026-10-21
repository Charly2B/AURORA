import { useState } from 'react'
import { Link, NavLink, Route, Routes } from 'react-router-dom'
import { AlertTriangle, Cpu, Database, Film, HardDrive, Info, Library, RefreshCw, Users, Wand2, Wrench } from 'lucide-react'
import { api } from '../../lib/api'
import { useApi, useDocumentTitle, useEvents } from '../../lib/hooks'
import { date, relative } from '../../lib/format'
import type { ScanProgress } from '../../lib/types'
import { Button, ErrorState, Notice, PageLoader, ProgressBar, cx, useToast } from '../../components/ui'
import { Libraries } from './Libraries'
import { Corrections } from './Corrections'
import { UsersPage } from './Users'
import { SettingsPage } from './Settings'
import { Maintenance } from './Maintenance'
import { About } from './About'

const TABS = [
  { to: '', label: 'Tableau de bord', icon: Cpu, end: true },
  { to: 'libraries', label: 'Bibliothèques', icon: Library },
  { to: 'corrections', label: 'Identification', icon: Wand2 },
  { to: 'users', label: 'Utilisateurs', icon: Users },
  { to: 'settings', label: 'Réglages', icon: Wrench },
  { to: 'maintenance', label: 'Maintenance', icon: Database },
  { to: 'about', label: 'À propos', icon: Info },
]

export default function Admin() {
  useDocumentTitle('Administration')
  return (
    <div className="pt-24 px-[var(--gutter)] pb-16">
      <p className="eyebrow mb-3">Administration</p>
      <nav className="flex gap-1 overflow-x-auto [scrollbar-width:none] border-b border-white/[0.06] mb-8" aria-label="Sections d'administration">
        {TABS.map((t) => (
          <NavLink key={t.to} to={`/admin/${t.to}`} end={t.end}
            className={({ isActive }) => cx('flex items-center gap-2 px-3.5 py-3 text-sm whitespace-nowrap border-b-2 -mb-px transition-colors',
              isActive ? 'border-aurora text-paper' : 'border-transparent text-ash hover:text-paper')}>
            <t.icon className="size-4" /> {t.label}
          </NavLink>
        ))}
      </nav>
      <Routes>
        <Route index element={<Dashboard />} />
        <Route path="libraries" element={<Libraries />} />
        <Route path="corrections" element={<Corrections />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="maintenance" element={<Maintenance />} />
        <Route path="about" element={<About />} />
      </Routes>
    </div>
  )
}

interface Dash {
  version: string; startedAt: number; movies: number; series: number; episodes: number; files: number; missing: number
  unidentified: number; probeErrors: number; pending: number; libraries: number; users: number; lastScanAt: number
  scan: ScanProgress | null; transcodes: number; remuxes: number; ffmpeg: boolean; ffprobe: boolean; tmdb: boolean
  errors: { taskId: number; library: string; status: string; message: string; errors: number; at: number }[] | null
}

export const PHASES: Record<string, string> = {
  walk: 'Parcours des dossiers', identify: 'Identification', probe: 'Analyse technique', metadata: 'Métadonnées', done: 'Terminé',
}

function Dashboard() {
  const d = useApi<Dash>('/admin/dashboard')
  const libs = useApi<{ id: number; name: string; suggestKind?: string }[]>('/admin/libraries')
  const toast = useToast()
  const [scan, setScan] = useState<ScanProgress | null>(null)
  useEvents((type, data) => {
    if (type !== 'scan') return
    const p = data as ScanProgress
    setScan(p.status === 'running' ? p : null)
    if (p.status !== 'running') d.reload()
  })
  if (d.error) return <ErrorState error={d.error} retry={d.reload} />
  if (!d.data) return <PageLoader />
  const x = d.data
  const current = scan ?? x.scan
  const rescan = async () => {
    try { await api.post('/admin/scan', {}); toast('ok', 'Analyse lancée') } catch (e) { toast('error', (e as Error).message) }
  }
  return (
    <div className="space-y-8 max-w-6xl">
      {(!x.ffprobe || !x.ffmpeg) && (
        <Notice tone="error">
          {!x.ffprobe && 'ffprobe est introuvable : les pistes et codecs ne peuvent pas être analysés. '}
          {!x.ffmpeg && 'ffmpeg est introuvable : seule la lecture directe est possible (pas de remux ni de conversion). '}
          Réinstallez le paquet AURORA, qui embarque ces outils.
        </Notice>
      )}
      {libs.data?.filter((l) => l.suggestKind).map((l) => (
        <Notice key={l.id} tone="warn">
          La bibliothèque <b>{l.name}</b> semble contenir des {l.suggestKind === 'series' ? 'séries' : 'films'} mais est déclarée
          comme {l.suggestKind === 'series' ? 'Films' : 'Séries'} : les affiches et les fiches ne peuvent pas être trouvées.{' '}
          <Link to="/admin/libraries" className="text-aurora underline underline-offset-4">Corriger le type</Link>
        </Notice>
      ))}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Stat label="Films" value={x.movies} icon={<Film className="size-4" />} />
        <Stat label="Séries" value={x.series} sub={`${x.episodes} épisodes`} />
        <Stat label="Fichiers" value={x.files} sub={x.missing ? `${x.missing} introuvables` : 'tous présents'} warn={x.missing > 0} icon={<HardDrive className="size-4" />} />
        <Stat label="À identifier" value={x.unidentified} warn={x.unidentified > 0} link={x.unidentified ? '/admin/corrections' : undefined} />
      </div>
      <section className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-6">
        <div className="flex flex-wrap items-center justify-between gap-4 mb-4">
          <div>
            <h2 className="font-display text-2xl">Analyse</h2>
            <p className="text-sm text-ash">Dernière analyse : {relative(x.lastScanAt)}{x.pending ? ` · ${x.pending} fichier(s) en attente d'analyse technique` : ''}</p>
          </div>
          <Button icon={<RefreshCw className="size-4" />} onClick={rescan} disabled={!!current || !x.libraries}>Analyser maintenant</Button>
        </div>
        {current ? (
          <div className="space-y-2">
            <div className="flex justify-between text-sm"><span>{current.library} — {PHASES[current.phase] ?? current.phase}</span><span className="tabular-nums text-ash">{current.done} / {current.total || '…'}</span></div>
            <ProgressBar value={current.total ? current.done / current.total : 0} />
            <p className="text-xs text-ash truncate">{current.current}</p>
            <Button size="sm" variant="ghost" onClick={() => api.post('/admin/scan/cancel').then(() => toast('info', 'Analyse interrompue ; elle reprendra au prochain lancement'))}>Interrompre</Button>
          </div>
        ) : <p className="text-sm text-mist">Aucune analyse en cours. Les dossiers sont revérifiés automatiquement (réglable dans Réglages).</p>}
      </section>
      <div className="grid md:grid-cols-2 gap-4">
        <section className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-6 space-y-2 text-sm">
          <h2 className="font-display text-2xl mb-2">Serveur</h2>
          <Line k="Version" v={`AURORA ${x.version}`} />
          <Line k="Démarré" v={date(x.startedAt)} />
          <Line k="Lectures en cours" v={`${x.remuxes} flux · ${x.transcodes} conversion(s)`} />
          <Line k="Métadonnées TMDB" v={x.tmdb ? 'configurées' : 'non configurées (mode local)'} />
          <Line k="Profils" v={String(x.users)} />
        </section>
        <section className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-6 text-sm">
          <h2 className="font-display text-2xl mb-3">Erreurs importantes</h2>
          {!x.errors?.length && <p className="text-mist">Aucune erreur récente.</p>}
          <ul className="space-y-2">
            {x.errors?.map((e) => (
              <li key={e.taskId} className="flex gap-3">
                <AlertTriangle className="size-4 text-ember shrink-0 mt-0.5" />
                <div><span className="text-paper">{e.library}</span> <span className="text-ash">— {relative(e.at)}</span><div className="text-ash">{e.message || `${e.errors} erreur(s)`}</div></div>
              </li>
            ))}
          </ul>
          {!!x.errors?.length && <Link to="/admin/maintenance" className="inline-block mt-3 text-aurora text-sm">Voir le détail</Link>}
        </section>
      </div>
    </div>
  )
}

function Stat({ label, value, sub, icon, warn, link }: { label: string; value: number; sub?: string; icon?: React.ReactNode; warn?: boolean; link?: string }) {
  const body = (
    <div className={cx('rounded-2xl bg-night/70 ring-1 p-5 h-full', warn ? 'ring-ember/30' : 'ring-white/[0.06]', link && 'hover:ring-aurora/40')}>
      <div className="flex items-center gap-2 eyebrow">{icon}{label}</div>
      <div className={cx('font-display text-4xl mt-2 tabular-nums', warn && 'text-ember')}>{value.toLocaleString('fr-FR')}</div>
      {sub && <div className="text-xs text-ash mt-1">{sub}</div>}
    </div>
  )
  return link ? <Link to={link}>{body}</Link> : body
}

export function Line({ k, v }: { k: string; v: React.ReactNode }) {
  return <div className="flex justify-between gap-4 border-b border-white/[0.04] pb-2"><span className="text-ash">{k}</span><span className="text-right text-mist break-all">{v}</span></div>
}
