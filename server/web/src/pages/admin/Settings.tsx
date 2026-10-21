import { useState } from 'react'
import { api, ApiError } from '../../lib/api'
import { useApi } from '../../lib/hooks'
import { Button, ErrorState, Field, Notice, PageLoader, Select, Toggle, useToast } from '../../components/ui'

interface S { tmdbConfigured: boolean; language: string; scanIntervalHours: number; probeJobs: number; maxTranscodes: number; allow4kTranscode: boolean; watchEnabled: boolean }

export function SettingsPage() {
  const s = useApi<S>('/admin/settings')
  const toast = useToast()
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  if (s.error) return <ErrorState error={s.error} retry={s.reload} />
  if (!s.data) return <PageLoader />
  const put = async (patch: Partial<S> & { tmdbApiKey?: string }) => {
    setBusy(true)
    setError(undefined)
    try {
      s.setData(await api.put<S>('/admin/settings', patch))
      toast('ok', 'Réglage enregistré')
      if (patch.tmdbApiKey !== undefined) setKey('')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    }
    setBusy(false)
  }
  const x = s.data
  return (
    <div className="max-w-3xl space-y-8">
      {error && <Notice tone="error">{error}</Notice>}
      <Panel title="Métadonnées">
        <p className="text-sm text-ash">État : {x.tmdbConfigured ? <span className="text-aurora">clé TMDB configurée</span> : 'mode local (noms de fichiers et images extraites des vidéos)'}.</p>
        <div className="flex flex-wrap items-end gap-3">
          <Field label={x.tmdbConfigured ? 'Remplacer la clé TMDB' : 'Clé TMDB'} value={key} onChange={(e) => setKey(e.target.value)} autoComplete="off" className="flex-1 min-w-64" />
          <Button busy={busy} disabled={!key.trim()} onClick={() => put({ tmdbApiKey: key })}>Valider</Button>
          {x.tmdbConfigured && <Button variant="ghost" onClick={() => put({ tmdbApiKey: '' })}>Retirer la clé</Button>}
        </div>
        <Select label="Langue des métadonnées" value={x.language} onChange={(e) => put({ language: e.target.value })}>
          <option value="fr-FR">Français (repli anglais)</option><option value="en-US">Anglais</option>
        </Select>
        <p className="text-[11px] text-ash">This product uses the TMDB API but is not endorsed or certified by TMDB.</p>
      </Panel>
      <Panel title="Analyse des dossiers">
        <Toggle checked={x.watchEnabled} onChange={(v) => put({ watchEnabled: v })} label="Détecter immédiatement les ajouts"
          hint="Comme Plex : un film ou un épisode copié dans un dossier est indexé environ 20 secondes après la fin de la copie. Ne fonctionne que pour les disques du NAS (pas les partages réseau montés)." />
        <Select label="Vérification complète périodique (filet de sécurité)" value={String(x.scanIntervalHours)} onChange={(e) => put({ scanIntervalHours: Number(e.target.value) })}>
          <option value="0">Désactivée (manuelle uniquement)</option><option value="1">Toutes les heures</option><option value="6">Toutes les 6 heures</option>
          <option value="12">Toutes les 12 heures</option><option value="24">Une fois par jour</option>
        </Select>
        <p className="text-xs text-ash -mt-2">Analyse incrémentale : seuls les fichiers nouveaux ou modifiés sont lus. Aucune analyse au démarrage du NAS.</p>
        <Select label="Analyses techniques simultanées" value={String(x.probeJobs)} onChange={(e) => put({ probeJobs: Number(e.target.value) })}>
          <option value="1">1 (disques les plus sollicités au minimum)</option><option value="2">2 (recommandé)</option><option value="3">3</option><option value="4">4</option>
        </Select>
      </Panel>
      <Panel title="Lecture et conversion">
        <Select label="Conversions vidéo simultanées" value={String(x.maxTranscodes)} onChange={(e) => put({ maxTranscodes: Number(e.target.value) })}>
          <option value="0">Aucune (lecture directe et remux uniquement)</option><option value="1">1 (recommandé sur DS1825+)</option><option value="2">2</option><option value="3">3</option>
        </Select>
        <Toggle checked={x.allow4kTranscode} onChange={(v) => put({ allow4kTranscode: v })} label="Autoriser la conversion des vidéos 4K"
          hint="Déconseillé : sans accélération matérielle, le processeur du NAS ne tient en général pas le temps réel en 4K. La lecture risque de saccader et le NAS d'être saturé." />
        <Notice>La lecture directe est toujours préférée. Le remux (changement de conteneur, audio converti si besoin) est peu coûteux ; seule la conversion vidéo sollicite fortement le processeur.</Notice>
      </Panel>
    </div>
  )
}

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return <section className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-6 space-y-4"><h2 className="font-display text-2xl">{title}</h2>{children}</section>
}
