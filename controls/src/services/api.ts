export type CurrentTrack = {
  title: string
  artist: string
  filePath: string
  coverUrl: string | null
}

export type SupervisionStatus = {
  httpOnline: boolean
  connectedClients: number
  currentTrack: CurrentTrack | null
}

type SupervisionPayload = {
  httpOnline?: boolean
  connectedClients?: number
  currentTrack?: {
    title?: string
    artist?: string
    filePath?: string
    coverUrl?: string | null
  } | null
}

type SupervisionHandlers = {
  onStatus: (status: SupervisionStatus) => void
  onOpen?: () => void
  onError?: (error: Error) => void
}

export class ApiError extends Error {
  status: number

  constructor(status: number, path: string) {
    super(`HTTP ${status} on ${path}`)
    this.name = 'ApiError'
    this.status = status
  }
}

const API_BASE_URL = import.meta.env.VITE_TRACKKER_API_BASE ?? 'http://localhost:9000'
export const PREVIEW_URL = import.meta.env.VITE_TRACKKER_PREVIEW_URL ?? 'http://192.168.1.60:9000'

const toUrl = (path: string): string => `${API_BASE_URL}${path}`

const fetchJson = async <T>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(toUrl(path), {
    ...init,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })

  if (!response.ok) {
    throw new ApiError(response.status, path)
  }


  const body = await response.text()
  return body
      ? JSON.parse(body) as T
      : null as T
}

export const checkPinCode = async (pinCode: string): Promise<void> => {
  return fetchJson(`/api/pincode/${pinCode}`, { method: 'POST' })
}

export const getSessionStatus = async (): Promise<boolean> => {
  try {
    const payload = await fetchJson<{ authenticated?: boolean } | null>('/api/control/session', {
      method: 'GET',
    })

    if (payload && typeof payload.authenticated === 'boolean') {
      return payload.authenticated
    }

    return true
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 401 || error.status === 403 || error.status === 404) {
        return false
      }
    }
    throw error
  }
}

const toSupervisionStatus = (payload: SupervisionPayload): SupervisionStatus => {
  const hasTrack = payload.currentTrack !== null && payload.currentTrack !== undefined

  return {
    httpOnline: payload.httpOnline ?? false,
    connectedClients: payload.connectedClients ?? 0,
    currentTrack: hasTrack
      ? {
          title: payload.currentTrack?.title ?? 'Titre inconnu',
          artist: payload.currentTrack?.artist ?? 'Artiste inconnu',
          filePath: payload.currentTrack?.filePath ?? '-',
          coverUrl: payload.currentTrack?.coverUrl ?? null,
        }
      : null,
  }
}

export const connectSupervisionStream = (handlers: SupervisionHandlers): (() => void) => {
  const source = new EventSource(toUrl('/api/control/supervision/events'), {
    withCredentials: true,
  })

  source.onopen = () => {
    handlers.onOpen?.()
  }

  source.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data) as SupervisionPayload
      handlers.onStatus(toSupervisionStatus(payload))
    } catch {
      handlers.onError?.(new Error('Payload SSE supervision invalide.'))
    }
  }

  source.onerror = () => {
    handlers.onError?.(new Error('Connexion SSE supervision interrompue.'))
  }

  return () => {
    source.close()
  }
}

export const sendControlAction = async (action: string): Promise<void> => {
  await fetchJson('/api/control/actions/' + encodeURIComponent(action), {
    method: 'POST',
  })
}

// TODO(back): implementer GET /api/control/supervision/events (SSE) pour exposer httpOnline, connectedClients et currentTrack
// TODO(back): implementer POST /api/control/actions/{action} pour blackout/freeze_tracking



