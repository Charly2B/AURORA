import { useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api, ApiError } from '../lib/api'
import { useDocumentTitle } from '../lib/hooks'
import { useSession } from '../lib/session'
import { Button, Field, Notice } from '../components/ui'
import { PageHeader } from '../components/Layout'

interface Request { name: string; ip: string; age: number; ok: boolean }

function ago(seconds: number): string {
  if (seconds < 60) return "à l'instant"
  const m = Math.round(seconds / 60)
  return `il y a ${m} min`
}

/**
 * Associer une TV : on saisit le code affiché par l'app, on vérifie l'appareil, puis on confirme.
 * Deux temps exprès : un lien /link?code=… reçu d'un tiers ne doit jamais associer en un clic.
 */
export function LinkDevice() {
  useDocumentTitle('Associer une TV')
  const { user } = useSession()
  const [params] = useSearchParams()
  const [code, setCode] = useState(params.get('code') ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [pending, setPending] = useState<Request>()
  const [done, setDone] = useState<string>()

  const send = async (confirm: boolean) => {
    setBusy(true)
    setError(undefined)
    try {
      const r = await api.post<Request>('/auth/link/approve', { code, confirm })
      if (confirm) {
        setDone(r.name)
        setPending(undefined)
        setCode('')
      } else {
        setPending(r)
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
      setPending(undefined)
    }
    setBusy(false)
  }
  const check = (e: FormEvent) => {
    e.preventDefault()
    setDone(undefined)
    send(false)
  }

  return (
    <div className="pb-10">
      <PageHeader eyebrow="Appareils" title="Associer une TV" />
      <div className="px-[var(--gutter)] max-w-md space-y-5">
        <p className="text-mist leading-relaxed">
          Saisissez le code affiché par l'app AURORA de la TV. Elle se connectera sur le profil
          <strong className="text-paper"> {user?.username}</strong>, sans mot de passe.
        </p>
        {done && <Notice>« {done} » est connectée. Vous pouvez reprendre la télécommande.</Notice>}
        {error && <Notice tone="error">{error}</Notice>}
        {pending ? (
          <div className="space-y-5">
            <Notice tone="warn">
              Associer <strong className="text-paper">« {pending.name} »</strong> (adresse {pending.ip}, demande {ago(pending.age)})
              à votre profil ? Ne confirmez que si ce code est affiché sur <strong className="text-paper">votre</strong> TV,
              devant vous.
            </Notice>
            <div className="flex gap-3">
              <Button variant="primary" size="lg" busy={busy} onClick={() => send(true)}>Confirmer</Button>
              <Button size="lg" onClick={() => { setPending(undefined); setCode('') }}>Annuler</Button>
            </div>
          </div>
        ) : (
          <form onSubmit={check} className="space-y-5">
            <Field label="Code de la TV" placeholder="K7M-2PQ" autoFocus autoComplete="off" autoCapitalize="characters"
              spellCheck={false} maxLength={9} value={code} onChange={(e) => setCode(e.target.value.toUpperCase())}
              className="[&_input]:text-2xl [&_input]:tracking-[0.3em] [&_input]:h-14 [&_input]:font-semibold" required />
            <Button variant="primary" size="lg" busy={busy} type="submit">Continuer</Button>
          </form>
        )}
      </div>
    </div>
  )
}
