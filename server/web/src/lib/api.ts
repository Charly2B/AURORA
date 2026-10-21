// Client de l'API AURORA. Toutes les requêtes modifiantes portent l'en-tête X-Aurora (protection CSRF).

export class ApiError extends Error {
  constructor(public status: number, message: string, public data?: Record<string, unknown>) {
    super(message)
  }
}

async function request<T>(method: string, path: string, body?: unknown, init?: RequestInit): Promise<T> {
  let resp: Response
  try {
    resp = await fetch('/api/v1' + path, {
      method,
      credentials: 'same-origin',
      headers: { 'X-Aurora': '1', ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}) },
      body: body !== undefined ? JSON.stringify(body) : undefined,
      ...init,
    })
  } catch {
    throw new ApiError(0, 'Le serveur AURORA ne répond pas. Vérifiez que le NAS est allumé et accessible.')
  }
  if (resp.status === 401 && !path.startsWith('/auth/') && !path.startsWith('/status')) {
    window.dispatchEvent(new Event('aurora:unauthorized'))
  }
  const text = await resp.text()
  let data: unknown = undefined
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = undefined
    }
  }
  if (!resp.ok) {
    const msg = (data as { error?: string } | undefined)?.error ?? `Erreur ${resp.status}`
    throw new ApiError(resp.status, msg.charAt(0).toUpperCase() + msg.slice(1), data as Record<string, unknown> | undefined)
  }
  return data as T
}

export const api = {
  get: <T>(path: string, init?: RequestInit) => request<T>('GET', path, undefined, init),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body ?? {}),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body ?? {}),
  del: <T>(path: string) => request<T>('DELETE', path),
  /** Envoi qui survit à la fermeture de l'onglet (sauvegarde de position). */
  beacon(path: string, body: unknown) {
    fetch('/api/v1' + path, {
      method: 'POST', keepalive: true, credentials: 'same-origin',
      headers: { 'X-Aurora': '1', 'Content-Type': 'application/json' }, body: JSON.stringify(body),
    }).catch(() => {})
  },
}

export function qs(params: Record<string, string | number | boolean | undefined | null>): string {
  const u = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '' || v === false) continue
    u.set(k, v === true ? '1' : String(v))
  }
  const s = u.toString()
  return s ? '?' + s : ''
}
