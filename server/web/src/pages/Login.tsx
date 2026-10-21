import { useState, type FormEvent } from 'react'
import { api, ApiError } from '../lib/api'
import { useSession } from '../lib/session'
import { useDocumentTitle } from '../lib/hooks'
import { Button, Field, Logo, Notice } from '../components/ui'

export function Login() {
  useDocumentTitle('Connexion')
  const { refresh } = useSession()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(undefined)
    try {
      await api.post('/auth/login', { username, password })
      await refresh()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err))
      setBusy(false)
    }
  }
  return (
    <Stage>
      <form onSubmit={submit} className="w-full max-w-sm space-y-5 rise">
        <div className="mb-10"><Logo /></div>
        <h1 className="title-xl text-5xl">Bonsoir.</h1>
        <p className="text-ash -mt-2">Connectez-vous pour retrouver votre médiathèque.</p>
        {error && <Notice tone="error">{error}</Notice>}
        <Field label="Nom d'utilisateur" autoComplete="username" autoFocus value={username} onChange={(e) => setUsername(e.target.value)} required />
        <Field label="Mot de passe" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        <Button variant="primary" size="lg" className="w-full" busy={busy} type="submit">Entrer</Button>
      </form>
    </Stage>
  )
}

/** Décor commun aux écrans d'accueil (connexion, assistant). */
export function Stage({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative min-h-dvh overflow-hidden grid place-items-center px-4 py-[calc(9vh+40px)]">
      <div className="absolute -top-1/2 -left-1/3 w-[160%] h-[120%] aurora-ribbon opacity-80 pointer-events-none" aria-hidden />
      <div className="absolute inset-x-0 top-0 h-[9vh] bg-black/70" aria-hidden />
      <div className="absolute inset-x-0 bottom-0 h-[9vh] bg-black/70" aria-hidden />
      <div className="relative w-full flex justify-center">{children}</div>
    </div>
  )
}
