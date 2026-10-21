import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from './api'
import type { Prefs, Status, User } from './types'

interface SessionValue {
  status: Status | undefined
  user: User | undefined
  error: string | undefined
  refresh: () => Promise<void>
  logout: () => Promise<void>
  setPrefs: (p: Prefs) => void
}

const Ctx = createContext<SessionValue | null>(null)

export function SessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status>()
  const [error, setError] = useState<string>()
  const refresh = useCallback(async () => {
    try {
      setStatus(await api.get<Status>('/status'))
      setError(undefined)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }, [])
  useEffect(() => {
    refresh()
    const onUnauth = () => refresh()
    window.addEventListener('aurora:unauthorized', onUnauth)
    return () => window.removeEventListener('aurora:unauthorized', onUnauth)
  }, [refresh])
  const logout = useCallback(async () => {
    await api.post('/auth/logout').catch(() => {})
    await refresh()
  }, [refresh])
  const setPrefs = useCallback((p: Prefs) => {
    setStatus((s) => (s?.user ? { ...s, user: { ...s.user, prefs: p } } : s))
  }, [])
  return (
    <Ctx.Provider value={{ status, user: status?.user, error, refresh, logout, setPrefs }}>{children}</Ctx.Provider>
  )
}

export function useSession(): SessionValue {
  const v = useContext(Ctx)
  if (!v) throw new Error('SessionProvider manquant')
  return v
}
