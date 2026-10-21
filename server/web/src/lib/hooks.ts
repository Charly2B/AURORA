import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError } from './api'

export interface Resource<T> {
  data: T | undefined
  error: ApiError | undefined
  loading: boolean
  reload: () => void
  setData: (d: T) => void
}

/** Chargement d'une ressource GET, annulé si le composant change de cible. */
export function useApi<T>(path: string | null): Resource<T> {
  const [data, setData] = useState<T>()
  const [error, setError] = useState<ApiError>()
  const [loading, setLoading] = useState(!!path)
  const [tick, setTick] = useState(0)
  useEffect(() => {
    if (!path) return
    const ctrl = new AbortController()
    setLoading(true)
    setError(undefined)
    api
      .get<T>(path, { signal: ctrl.signal })
      .then((d) => {
        setData(d)
        setLoading(false)
      })
      .catch((e) => {
        if (ctrl.signal.aborted) return
        setError(e instanceof ApiError ? e : new ApiError(0, String(e)))
        setLoading(false)
      })
    return () => ctrl.abort()
  }, [path, tick])
  const reload = useCallback(() => setTick((t) => t + 1), [])
  return { data, error, loading, reload, setData }
}

/** Flux d'événements serveur (SSE), reconnecté automatiquement par le navigateur. */
export function useEvents(onEvent: (type: string, data: unknown) => void) {
  const ref = useRef(onEvent)
  ref.current = onEvent
  useEffect(() => {
    const es = new EventSource('/api/v1/events')
    es.onmessage = (m) => {
      try {
        const e = JSON.parse(m.data) as { type: string; data: unknown }
        ref.current(e.type, e.data)
      } catch {
        /* message ignoré */
      }
    }
    return () => es.close()
  }, [])
}

export function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms)
    return () => clearTimeout(t)
  }, [value, ms])
  return v
}

export function useDocumentTitle(title?: string) {
  useEffect(() => {
    document.title = title ? `${title} — AURORA` : 'AURORA'
  }, [title])
}
