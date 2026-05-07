export type CurrentTrack = {
  title: string
  artist: string
  filePath: string
  coverUrl: string | null
}

export type DisplayMode = 'live' | 'blackout' | 'freeze_tracking'

export type StreamDeckButton = {
  id: number
  label: string
  textColor: string
  backgroundColor: string
  icon?: string | null
  displayMode: DisplayMode
  position: number
  isDeletable: boolean
}

export type TrackkerIPs = {
  display: string[]
  controls: string[]
}

export type SupervisionStatus = {
  displayServerOnline: boolean
  connectedClients: {
    display: number
    controls: number
  }
  displayMode: DisplayMode
  currentTrack: CurrentTrack | null
}

type SupervisionPayload = {
  httpOnline?: boolean
  connectedClients?: Record<string, number>
  displayMode?: DisplayMode
  currentTrack?: {
    title?: string
    artist?: string
    filePath?: string
    coverUrlPath?: string | null
    coverURLPath?: string | null
    coverUrl?: string | null
  } | null
  display?: number
  controls?: number
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

const API_BASE_URL = import.meta.env.VITE_TRACKKER_API_BASE ?? 'http://localhost:8080'
export const PREVIEW_URL = import.meta.env.VITE_TRACKKER_PREVIEW_URL ?? 'http://localhost:9000'

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

const toSupervisionStatus = (payload: SupervisionPayload, lastStatus: SupervisionStatus): SupervisionStatus => {
  const hasTrack = payload.currentTrack !== null && payload.currentTrack !== undefined
  const nextConnected = {
    ...lastStatus.connectedClients,
  }

  if (payload.connectedClients) {
    if (typeof payload.connectedClients.display === 'number') {
      nextConnected.display = payload.connectedClients.display
    }
    if (typeof payload.connectedClients.controls === 'number') {
      nextConnected.controls = payload.connectedClients.controls
    }
  }

  if (typeof payload.display === 'number') {
    nextConnected.display = payload.display
  }

  if (typeof payload.controls === 'number') {
    nextConnected.controls = payload.controls
  }

  const rawCoverPath =
    payload.currentTrack?.coverUrlPath ??
    payload.currentTrack?.coverURLPath ??
    payload.currentTrack?.coverUrl ??
    null
  const coverUrl = rawCoverPath
    ? rawCoverPath.startsWith('http')
      ? rawCoverPath
      : `${PREVIEW_URL}${rawCoverPath}`
    : null

  return {
    displayServerOnline: payload.httpOnline ?? lastStatus.displayServerOnline,
    connectedClients: nextConnected,
    displayMode: payload.displayMode ?? lastStatus.displayMode,
    currentTrack: hasTrack
      ? {
          title: payload.currentTrack?.title ?? 'Titre inconnu',
          artist: payload.currentTrack?.artist ?? 'Artiste inconnu',
          filePath: payload.currentTrack?.filePath ?? '-',
          coverUrl,
        }
      : payload.currentTrack === null
        ? null
        : lastStatus.currentTrack,
  }
}

export const connectSupervisionStream = (handlers: SupervisionHandlers): (() => void) => {
  const source = new EventSource(toUrl('/api/control/supervision/events'), {
    withCredentials: true,
  })
  let lastStatus: SupervisionStatus = {
    displayServerOnline: false,
    connectedClients: {
      display: 0,
      controls: 0,
    },
    displayMode: 'live',
    currentTrack: null,
  }

  const emitNextStatus = (payload: SupervisionPayload) => {
    lastStatus = toSupervisionStatus(payload, lastStatus)
    handlers.onStatus(lastStatus)
  }

  const parsePayload = (raw: string): SupervisionPayload => JSON.parse(raw) as SupervisionPayload

  source.onopen = () => {
    handlers.onOpen?.()
  }

  source.onmessage = (event) => {
    try {
      emitNextStatus(parsePayload(event.data))
    } catch {
      handlers.onError?.(new Error('Payload SSE supervision invalide.'))
    }
  }

  source.addEventListener('http_online', (event) => {
    try {
      emitNextStatus(parsePayload((event as MessageEvent).data))
    } catch {
      handlers.onError?.(new Error('Payload SSE http_online invalide.'))
    }
  })

  source.addEventListener('connected_clients', (event) => {
    try {
      emitNextStatus(parsePayload((event as MessageEvent).data))
    } catch {
      handlers.onError?.(new Error('Payload SSE connected_clients invalide.'))
    }
  })

  source.addEventListener('current_track', (event) => {
    try {
      emitNextStatus(parsePayload((event as MessageEvent).data))
    } catch {
      handlers.onError?.(new Error('Payload SSE current_track invalide.'))
    }
  })

  source.addEventListener('display_mode', (event) => {
    try {
      emitNextStatus(parsePayload((event as MessageEvent).data))
    } catch {
      handlers.onError?.(new Error('Payload SSE display_mode invalide.'))
    }
  })

  source.onerror = () => {
    handlers.onError?.(new Error('Connexion SSE supervision interrompue.'))
  }

  return () => {
    source.close()
  }
}

export const getStreamDeckButtons = async (): Promise<StreamDeckButton[]> => {
  return fetchJson('/api/control/ui/streamdeck', {
    method: 'GET',
  })
}

export const toggleStreamDeckButton = async (buttonId: number): Promise<{ displayMode: DisplayMode }> => {
  return fetchJson(`/api/control/actions/button/${buttonId}`, {
    method: 'POST',
  })
}

export const startDisplayServer = async (): Promise<void> => {
  await fetchJson('/api/control/display/start', {
    method: 'POST',
  })
}

export const stopDisplayServer = async (): Promise<void> => {
  await fetchJson('/api/control/display/stop', {
    method: 'POST',
  })
}

export const getLocalIPs = async (): Promise<TrackkerIPs> => {
  return fetchJson('/api/control/ip', {
    method: 'GET',
  })
}

