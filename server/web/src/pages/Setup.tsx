import { useEffect, useState, type FormEvent } from 'react'
import { ArrowRight, Check, Clapperboard, Film, KeyRound, Languages, Tv } from 'lucide-react'
import { api, ApiError } from '../lib/api'
import { useApi, useDocumentTitle, useEvents } from '../lib/hooks'
import { useSession } from '../lib/session'
import type { ScanProgress } from '../lib/types'
import { Button, Field, Logo, Notice, ProgressBar, Spinner, cx } from '../components/ui'
import { FolderPicker } from '../components/FolderPicker'
import { Stage } from './Login'

// Assistant de premier démarrage. Les étapes après la création du compte sont enregistrées côté serveur :
// l'assistant reprend où il s'était arrêté après une interruption.

const STEPS = [
  { id: 'welcome', label: 'Bienvenue' },
  { id: 'admin', label: 'Administrateur' },
  { id: 'language', label: 'Langue' },
  { id: 'libraries', label: 'Bibliothèques' },
  { id: 'metadata', label: 'Métadonnées' },
  { id: 'scan', label: 'Indexation' },
  { id: 'done', label: 'Terminé' },
] as const
type StepId = (typeof STEPS)[number]['id']

export function Setup() {
  useDocumentTitle('Installation')
  const { status, refresh } = useSession()
  const [local, setLocal] = useState<StepId>('welcome')
  const server = useApi<{ step: StepId; language: string }>(status?.hasAdmin ? '/admin/setup' : null)
  const step: StepId = status?.hasAdmin ? (server.data?.step ?? 'language') : local
  const go = async (next: StepId, extra?: Record<string, string>) => {
    await api.put('/admin/setup', { step: next, ...extra })
    if (next === 'done') await refresh()
    else server.reload()
  }
  const idx = STEPS.findIndex((s) => s.id === step)

  return (
    <Stage>
      <div className="w-full max-w-2xl">
        <div className="flex items-center justify-between mb-10"><Logo /><span className="text-xs text-ash">Étape {idx + 1} / {STEPS.length}</span></div>
        <ol className="flex gap-1.5 mb-10" aria-label="Progression de l'installation">
          {STEPS.map((s, i) => (
            <li key={s.id} className={cx('h-1 flex-1 rounded-full transition-colors', i <= idx ? 'bg-aurora' : 'bg-white/10')} title={s.label} />
          ))}
        </ol>
        <div key={step} className="rise">
          {step === 'welcome' && <Welcome onNext={() => setLocal('admin')} />}
          {step === 'admin' && <AdminStep onDone={refresh} />}
          {step === 'language' && <LanguageStep current={server.data?.language} onNext={(l) => go('libraries', { language: l })} />}
          {step === 'libraries' && <LibrariesStep onNext={() => go('metadata')} />}
          {step === 'metadata' && <MetadataStep onNext={() => go('scan')} />}
          {step === 'scan' && <ScanStep onNext={() => go('done')} />}
        </div>
      </div>
    </Stage>
  )
}

function Welcome({ onNext }: { onNext: () => void }) {
  return (
    <div>
      <h1 className="title-xl text-[52px] sm:text-[68px] mb-6">Votre cinéma,<br /><span className="text-aurora">chez vous.</span></h1>
      <p className="text-mist text-lg leading-relaxed max-w-lg mb-4">
        AURORA organise les films et séries stockés sur votre NAS et les lit directement dans votre navigateur.
      </p>
      <ul className="text-ash space-y-2 mb-10 text-[15px]">
        <li>· Tout reste sur votre réseau : aucun compte en ligne, aucun abonnement.</li>
        <li>· Vos fichiers sont lus, jamais modifiés, déplacés ni supprimés.</li>
        <li>· Quelques minutes suffisent pour commencer.</li>
      </ul>
      <Button variant="primary" size="lg" onClick={onNext} icon={<ArrowRight className="size-5" />}>Commencer</Button>
    </div>
  )
}

function AdminStep({ onDone }: { onDone: () => Promise<void> }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (password !== confirm) return setError('Les deux mots de passe ne correspondent pas.')
    setBusy(true)
    setError(undefined)
    try {
      await api.post('/setup/admin', { username, password })
      await onDone()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
      setBusy(false)
    }
  }
  return (
    <form onSubmit={submit} className="space-y-5 max-w-md">
      <h1 className="title-xl text-5xl">Compte administrateur</h1>
      <p className="text-ash">Ce compte gère les bibliothèques et les profils. Il est propre à AURORA (indépendant de DSM).</p>
      {error && <Notice tone="error">{error}</Notice>}
      <Field label="Nom d'utilisateur" autoComplete="username" value={username} onChange={(e) => setUsername(e.target.value)} required autoFocus />
      <Field label="Mot de passe" type="password" autoComplete="new-password" minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} required hint="8 caractères minimum." />
      <Field label="Confirmation" type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
      <Button variant="primary" size="lg" busy={busy} type="submit" icon={<ArrowRight className="size-5" />}>Créer le compte</Button>
    </form>
  )
}

function LanguageStep({ current, onNext }: { current?: string; onNext: (l: string) => void }) {
  const [lang, setLang] = useState(current ?? 'fr-FR')
  const opts = [
    { id: 'fr-FR', label: 'Français', hint: 'Titres, résumés et genres en français, anglais en repli.' },
    { id: 'en-US', label: 'English', hint: 'Titles, overviews and genres in English.' },
  ]
  return (
    <div className="space-y-6">
      <h1 className="title-xl text-5xl flex items-center gap-4"><Languages className="size-9 text-aurora" /> Langue</h1>
      <p className="text-ash">Langue des informations sur les films et séries (titres, résumés). L'interface est en français.</p>
      <div className="grid sm:grid-cols-2 gap-3">
        {opts.map((o) => (
          <button key={o.id} onClick={() => setLang(o.id)} aria-pressed={lang === o.id}
            className={cx('text-left rounded-xl p-5 ring-1 transition-colors', lang === o.id ? 'ring-aurora bg-aurora/[0.06]' : 'ring-graphite hover:ring-smoke')}>
            <div className="font-display text-2xl mb-1">{o.label}</div>
            <div className="text-sm text-ash">{o.hint}</div>
          </button>
        ))}
      </div>
      <Button variant="primary" size="lg" onClick={() => onNext(lang)} icon={<ArrowRight className="size-5" />}>Continuer</Button>
    </div>
  )
}

interface AdminLibrary { id: number; name: string; kind: 'movies' | 'series'; path: string; files: number }

function LibrariesStep({ onNext }: { onNext: () => void }) {
  const libs = useApi<AdminLibrary[]>('/admin/libraries')
  const [kind, setKind] = useState<'movies' | 'series'>('movies')
  const [path, setPath] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState<string>()
  const [suggest, setSuggest] = useState<'movies' | 'series'>()
  const [busy, setBusy] = useState(false)
  useEffect(() => { if (path) setName(path.split('/').pop() ?? '') }, [path])
  const add = async (k = kind, force = false) => {
    setBusy(true)
    setError(undefined)
    setSuggest(undefined)
    try {
      await api.post('/admin/libraries', { kind: k, path, name, force })
      setPath('')
      setName('')
      libs.reload()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
      if (err instanceof ApiError && err.status === 409 && err.data?.suggestKind) setSuggest(err.data.suggestKind as 'movies' | 'series')
    }
    setBusy(false)
  }
  return (
    <div className="space-y-6">
      <h1 className="title-xl text-5xl">Vos bibliothèques</h1>
      <p className="text-ash">Choisissez les dossiers partagés contenant vos films et vos séries. Ils sont lus en lecture seule.</p>
      <Notice>
        Un dossier grisé signale qu'AURORA n'a pas le droit de le lire. Dans DSM : <b>Panneau de configuration › Dossier partagé ›
        Modifier › Permissions</b>, choisissez <b>Utilisateurs internes du système</b>, puis donnez à <b>aurora</b> la <b>lecture seule</b>.
      </Notice>
      {libs.data && libs.data.length > 0 && (
        <ul className="space-y-2">
          {libs.data.map((l) => (
            <li key={l.id} className="flex items-center gap-3 rounded-xl bg-coal px-4 py-3 ring-1 ring-white/5">
              {l.kind === 'movies' ? <Film className="size-5 text-aurora" /> : <Tv className="size-5 text-aurora" />}
              <div className="min-w-0">
                <div className="font-medium">{l.name} <span className="text-xs text-ash font-normal ml-1">{l.kind === 'movies' ? 'Films' : 'Séries'}</span></div>
                <div className="text-xs text-ash truncate font-mono">{l.path}</div>
              </div>
              <Check className="size-4 text-aurora ml-auto" />
            </li>
          ))}
        </ul>
      )}
      <div className="rounded-2xl ring-1 ring-white/[0.07] p-5 space-y-4 bg-night/60">
        <div className="flex gap-2" role="radiogroup" aria-label="Type de contenu">
          {(['movies', 'series'] as const).map((k) => (
            <button key={k} role="radio" aria-checked={kind === k} onClick={() => setKind(k)}
              className={cx('flex items-center gap-2 h-10 px-4 rounded-full text-sm ring-1', kind === k ? 'ring-aurora text-paper bg-aurora/[0.08]' : 'ring-graphite text-ash')}>
              {k === 'movies' ? <Film className="size-4" /> : <Tv className="size-4" />}{k === 'movies' ? 'Films' : 'Séries'}
            </button>
          ))}
        </div>
        <FolderPicker value={path} onChange={setPath} />
        {path && <Field label="Nom affiché" value={name} onChange={(e) => setName(e.target.value)} />}
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
        <Button onClick={() => add()} disabled={!path} busy={busy}>Ajouter ce dossier</Button>
      </div>
      <div className="flex items-center gap-4">
        <Button variant="primary" size="lg" onClick={onNext} disabled={!libs.data?.length} icon={<ArrowRight className="size-5" />}>Continuer</Button>
        {!libs.data?.length && <button onClick={onNext} className="text-sm text-ash hover:text-paper underline underline-offset-4">Plus tard</button>}
      </div>
    </div>
  )
}

function MetadataStep({ onNext }: { onNext: () => void }) {
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await api.put('/admin/settings', { tmdbApiKey: key })
      onNext()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
      setBusy(false)
    }
  }
  return (
    <div className="space-y-6">
      <h1 className="title-xl text-5xl flex items-center gap-4"><KeyRound className="size-9 text-aurora" /> Affiches et résumés</h1>
      <p className="text-mist leading-relaxed">
        AURORA peut récupérer affiches, résumés, distribution et classifications depuis <b>The Movie Database (TMDB)</b>.
        Il faut une clé d'API gratuite, créée sur votre compte TMDB (<span className="font-mono text-sm">themoviedb.org › Paramètres › API</span>).
      </p>
      <p className="text-ash text-sm leading-relaxed">
        Seuls les titres et années de vos fichiers sont envoyés à TMDB. Images et informations sont ensuite gardées sur le NAS
        et restent consultables sans Internet. Sans clé, AURORA fonctionne avec les noms de fichiers et des images extraites des vidéos.
      </p>
      {error && <Notice tone="error">{error}</Notice>}
      <Field label="Clé d'API TMDB (v3) ou jeton de lecture (v4)" value={key} onChange={(e) => setKey(e.target.value)} autoComplete="off" spellCheck={false} />
      <div className="flex items-center gap-4">
        <Button variant="primary" size="lg" busy={busy} disabled={!key.trim()} onClick={save} icon={<ArrowRight className="size-5" />}>Valider la clé</Button>
        <button onClick={onNext} className="text-sm text-ash hover:text-paper underline underline-offset-4">Continuer sans métadonnées</button>
      </div>
      <p className="text-[11px] text-ash/80">This product uses the TMDB API but is not endorsed or certified by TMDB.</p>
    </div>
  )
}

function ScanStep({ onNext }: { onNext: () => void }) {
  const [progress, setProgress] = useState<ScanProgress | null>(null)
  const tasks = useApi<{ current: ScanProgress | null; queued: number[] | null; tasks: { status: string; library: string; done: number; errors: number }[] }>('/admin/tasks')
  useEffect(() => { if (tasks.data?.current) setProgress(tasks.data.current) }, [tasks.data])
  useEvents((type, data) => { if (type === 'scan') setProgress(data as ScanProgress) })
  useEffect(() => {
    // Lance une analyse si aucune n'a encore eu lieu (bibliothèques ajoutées avant une interruption).
    if (tasks.data && !tasks.data.current && tasks.data.tasks.length === 0) api.post('/admin/scan', {}).catch(() => {})
  }, [tasks.data])
  const phases: Record<string, string> = {
    walk: 'Parcours des dossiers', identify: 'Identification des titres', probe: 'Analyse des pistes vidéo et audio',
    metadata: 'Récupération des affiches et résumés', done: 'Terminé',
  }
  const running = progress?.status === 'running'
  return (
    <div className="space-y-6">
      <h1 className="title-xl text-5xl flex items-center gap-4"><Clapperboard className="size-9 text-aurora" /> Indexation</h1>
      <p className="text-ash">AURORA parcourt vos dossiers puis analyse chaque fichier. Vous pouvez commencer à naviguer pendant ce temps : l'analyse continue en arrière-plan.</p>
      <div className="rounded-2xl bg-night/70 ring-1 ring-white/[0.07] p-6 space-y-4 min-h-36">
        {!progress && tasks.data && !tasks.data.current && !tasks.data.queued?.length && tasks.data.tasks.length > 0 ? (
          // Analyse déjà terminée avant l'affichage de cette étape (petites bibliothèques).
          <div className="space-y-3">
            <ProgressBar value={1} />
            <ul className="text-sm text-mist space-y-1">
              {tasks.data.tasks.slice(0, 6).map((t, i) => (
                <li key={i}>{t.library} — {t.status === 'done' ? `${t.done} fichier(s) analysé(s)` : t.status}{t.errors ? `, ${t.errors} erreur(s)` : ''}</li>
              ))}
            </ul>
          </div>
        ) : !progress && <Spinner label="Préparation" />}
        {progress && (
          <>
            <div className="flex items-baseline justify-between gap-4">
              <span className="font-medium">{progress.library} — {phases[progress.phase] ?? progress.phase}</span>
              {progress.total > 0 && <span className="text-sm text-ash tabular-nums">{progress.done} / {progress.total}</span>}
            </div>
            <ProgressBar value={progress.total ? progress.done / progress.total : progress.status === 'done' ? 1 : 0} />
            <p className="text-xs text-ash truncate h-4">{running ? progress.current : progress.message || (progress.status === 'done' ? 'Analyse terminée.' : '')}</p>
            {progress.errors > 0 && <Notice tone="warn">{progress.errors} fichier(s) n'ont pas pu être analysés. Le détail est dans Administration › Maintenance.</Notice>}
          </>
        )}
      </div>
      <Button variant="primary" size="lg" onClick={onNext} icon={<ArrowRight className="size-5" />}>
        {running ? 'Ouvrir AURORA (analyse en cours)' : 'Ouvrir AURORA'}
      </Button>
    </div>
  )
}
