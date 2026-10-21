import { useState, type FormEvent } from 'react'
import { ArrowDown, ArrowUp, Eye, EyeOff } from 'lucide-react'
import { api, ApiError } from '../lib/api'
import { useDocumentTitle } from '../lib/hooks'
import { useSession } from '../lib/session'
import type { Prefs } from '../lib/types'
import { Button, Field, Notice, Select, Toggle, cx, useToast } from '../components/ui'
import { PageHeader } from '../components/Layout'

const SECTION_LABELS: Record<string, string> = {
  continue: 'Continuer à regarder', nextup: 'Prochain épisode', recent_movies: 'Films récemment ajoutés',
  recent_series: 'Séries récemment ajoutées', favorites: 'Favoris', watchlist: 'Ma liste', collections: 'Collections',
  unwatched_movies: 'Films non visionnés', series_in_progress: 'Séries en cours', suggestions: 'Suggestions de ma bibliothèque',
}

const LANGS = [['fre', 'Français'], ['eng', 'Anglais'], ['ger', 'Allemand'], ['spa', 'Espagnol'], ['ita', 'Italien'], ['jpn', 'Japonais']]

export function Profile() {
  useDocumentTitle('Mon profil')
  const { user, setPrefs } = useSession()
  const toast = useToast()
  const [p, setP] = useState<Prefs>(user!.prefs)
  const [busy, setBusy] = useState(false)
  const save = async (next: Prefs) => {
    setP(next)
    setBusy(true)
    try {
      const saved = await api.put<Prefs>('/me/prefs', next)
      setPrefs(saved)
      toast('ok', 'Préférences enregistrées')
    } catch (e) {
      toast('error', (e as Error).message)
    }
    setBusy(false)
  }
  const move = (i: number, dir: number) => {
    const h = [...p.home]
    const j = i + dir
    if (j < 0 || j >= h.length) return
    ;[h[i], h[j]] = [h[j], h[i]]
    save({ ...p, home: h })
  }
  return (
    <div className="pb-10">
      <PageHeader eyebrow={user!.role === 'child' ? 'Profil enfant' : user!.role === 'admin' ? 'Administrateur' : 'Utilisateur'} title={user!.username} />
      <div className="px-[var(--gutter)] grid gap-10 lg:grid-cols-2 max-w-6xl">
        <Panel title="Lecture">
          <Toggle checked={p.autoplayNext} onChange={(v) => save({ ...p, autoplayNext: v })} label="Enchaîner les épisodes"
            hint="Propose l'épisode suivant à la fin d'un épisode, avec un compte à rebours annulable." />
          <div className="grid sm:grid-cols-2 gap-4">
            <Select label="Langue audio préférée" value={p.audioLang} onChange={(e) => save({ ...p, audioLang: e.target.value })}>
              <option value="">Piste par défaut du fichier</option>
              {LANGS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </Select>
            <Select label="Sous-titres" value={p.subMode} onChange={(e) => save({ ...p, subMode: e.target.value as Prefs['subMode'] })}>
              <option value="auto">Automatiques (forcés, ou si l'audio n'est pas dans ma langue)</option>
              <option value="always">Toujours</option>
              <option value="off">Jamais</option>
            </Select>
            <Select label="Langue des sous-titres" value={p.subLang} onChange={(e) => save({ ...p, subLang: e.target.value })}>
              <option value="">Comme la langue audio préférée</option>
              {LANGS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </Select>
            <Select label="Taille des sous-titres" value={p.subSize} onChange={(e) => save({ ...p, subSize: e.target.value as Prefs['subSize'] })}>
              <option value="s">Petite</option><option value="m">Moyenne</option><option value="l">Grande</option>
            </Select>
          </div>
        </Panel>

        <Panel title="Page d'accueil" id="home">
          <p className="text-sm text-ash -mt-2">Choisissez les rangées affichées et leur ordre. Une rangée vide n'apparaît jamais.</p>
          <ul className="space-y-1">
            {p.home.map((s, i) => (
              <li key={s.id} className={cx('flex items-center gap-2 rounded-lg px-3 py-2 ring-1 ring-white/[0.05]', !s.visible && 'opacity-55')}>
                <span className="flex-1 text-sm">{SECTION_LABELS[s.id] ?? s.id}</span>
                <button aria-label="Monter" disabled={busy || i === 0} onClick={() => move(i, -1)} className="size-8 grid place-items-center rounded text-ash hover:text-paper disabled:opacity-30"><ArrowUp className="size-4" /></button>
                <button aria-label="Descendre" disabled={busy || i === p.home.length - 1} onClick={() => move(i, 1)} className="size-8 grid place-items-center rounded text-ash hover:text-paper disabled:opacity-30"><ArrowDown className="size-4" /></button>
                <button aria-label={s.visible ? 'Masquer' : 'Afficher'} aria-pressed={s.visible} disabled={busy}
                  onClick={() => save({ ...p, home: p.home.map((x) => (x.id === s.id ? { ...x, visible: !x.visible } : x)) })}
                  className="size-8 grid place-items-center rounded text-ash hover:text-paper">
                  {s.visible ? <Eye className="size-4" /> : <EyeOff className="size-4" />}
                </button>
              </li>
            ))}
          </ul>
        </Panel>

        <Panel title="Mot de passe"><PasswordForm /></Panel>
        {user!.role === 'child' && (
          <Panel title="Contrôle parental">
            <Notice>
              Ce profil n'affiche que les contenus classés {user!.maxAge != null ? `jusqu'à ${user!.maxAge} ans` : ''}{user!.hideUnrated ? ', et masque ceux sans classification' : ''}.
              La classification provient de TMDB : un contenu mal renseigné peut échapper au filtre. Seul un administrateur peut modifier ces réglages.
            </Notice>
          </Panel>
        )}
      </div>
    </div>
  )
}

function Panel({ title, id, children }: { title: string; id?: string; children: React.ReactNode }) {
  return (
    <section id={id} className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-6 space-y-5 scroll-mt-24">
      <h2 className="font-display text-2xl">{title}</h2>
      {children}
    </section>
  )
}

function PasswordForm() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const toast = useToast()
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(undefined)
    try {
      await api.put('/me/password', { current, new: next })
      setCurrent('')
      setNext('')
      toast('ok', 'Mot de passe modifié. Les autres appareils ont été déconnectés.')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
    }
    setBusy(false)
  }
  return (
    <form onSubmit={submit} className="space-y-4">
      {error && <Notice tone="error">{error}</Notice>}
      <Field label="Mot de passe actuel" type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} required />
      <Field label="Nouveau mot de passe" type="password" autoComplete="new-password" minLength={8} value={next} onChange={(e) => setNext(e.target.value)} required />
      <Button type="submit" busy={busy}>Modifier</Button>
    </form>
  )
}
