import { Show, createSignal, createEffect, onCleanup, onMount } from 'solid-js'
import {
  PREVIEW_URL,
  connectSupervisionStream,
  getStreamDeckButtons,
  startDisplayServer,
  stopDisplayServer,
  toggleStreamDeckButton,
  type StreamDeckButton,
  type SupervisionStatus,
} from '../services/api.ts'
import { ControlButton } from './ControlButton.tsx'

const DEFAULT_STATUS: SupervisionStatus = {
  displayServerOnline: false,
  connectedClients: {
    display: 0,
    controls: 0,
  },
  displayMode: 'live',
  currentTrack: null,
}

const PREVIEW_WIDTH = 1920
const PREVIEW_HEIGHT = 1080
const STREAMDECK_COLUMNS = 4

const toDisplayModeLabel = (mode: SupervisionStatus['displayMode']): string => {
  if (mode === 'blackout') {
    return 'Blackout'
  }

  if (mode === 'freeze_tracking') {
    return 'Freeze tracking'
  }

  return 'Live'
}

type DashboardProps = {
  onConnectionClosed: () => void
}

export const Dashboard = (props: DashboardProps) => {
  const [status, setStatus] = createSignal<SupervisionStatus>(DEFAULT_STATUS)
  const [isSseConnected, setIsSseConnected] = createSignal(false)
  const [isPreviewOpen, setIsPreviewOpen] = createSignal(true)
  const [previewScale, setPreviewScale] = createSignal(1)
  const [infoMessage, setInfoMessage] = createSignal('')
  const [streamDeckButtons, setStreamDeckButtons] = createSignal<StreamDeckButton[]>([])
  const [isLoadingButtons, setIsLoadingButtons] = createSignal(false)
  const [pendingButtons, setPendingButtons] = createSignal(new Set<number>())
  const currentTrack = () => status().currentTrack
  const sortedButtons = () => [...streamDeckButtons()].sort((a, b) => a.position - b.position)
  let previewShellRef: HTMLDivElement | undefined
  let previewResizeObserver: ResizeObserver | null = null

  let stopSupervision: (() => void) | null = null
  let hasNotifiedConnectionClosed = false

  const fetchStreamDeck = async () => {
    setIsLoadingButtons(true)
    try {
      const buttons = await getStreamDeckButtons()
      setStreamDeckButtons(buttons)
    } catch (error) {
      console.error(error)
      setInfoMessage('Impossible de charger le streamdeck.')
    } finally {
      setIsLoadingButtons(false)
    }
  }

  const toGridStyle = (button: StreamDeckButton, index: number) => {
    const position = button.position > 0 ? button.position : index + 1
    const row = Math.floor((position - 1) / STREAMDECK_COLUMNS) + 1
    const column = ((position - 1) % STREAMDECK_COLUMNS) + 1
    return {
      'grid-column': String(column),
      'grid-row': String(row),
    }
  }

  const renderIcon = (icon: string | null | undefined) => {
    if (!icon) {
      return null
    }

    const trimmed = icon.trim()
    if (trimmed.startsWith('<svg')) {
      return <span class="control-button-icon" innerHTML={trimmed} />
    }

    if (trimmed.startsWith('http') || trimmed.startsWith('/')) {
      return <img class="control-button-icon" src={trimmed} alt="" />
    }

    return <span class="control-button-icon">{trimmed}</span>
  }

  const startSupervision = () => {
    stopSupervision?.()
    setIsSseConnected(false)
    hasNotifiedConnectionClosed = false

    stopSupervision = connectSupervisionStream({
      onStatus: (nextStatus: any) => {
        setStatus(nextStatus)
      },
      onOpen: () => {
        setIsSseConnected(true)
        setInfoMessage('')
      },
      onError: (error: Error) => {
        console.error(error)
        setIsSseConnected(false)

        if (hasNotifiedConnectionClosed) {
          return
        }

        hasNotifiedConnectionClosed = true
        setInfoMessage('Connexion supervision SSE interrompue.')
        stopSupervision?.()
        stopSupervision = null
        props.onConnectionClosed()
      },
    })
  }

  const triggerAction = async (button: StreamDeckButton) => {
    const pending = new Set(pendingButtons())
    pending.add(button.id)
    setPendingButtons(pending)

    try {
      const response = await toggleStreamDeckButton(button.id)
      setInfoMessage(`Action envoyee: ${button.label} (mode: ${toDisplayModeLabel(response.displayMode)})`)
    } catch (error) {
      console.error(error)
      setInfoMessage(`Echec de l'action: ${button.label}`)
    } finally {
      const nextPending = new Set(pendingButtons())
      nextPending.delete(button.id)
      setPendingButtons(nextPending)
    }
  }

  const toggleDisplayServer = async () => {
    const shouldStop = status().displayServerOnline
    if (shouldStop) {
      const confirmed = window.confirm('Couper le serveur de diffusion ?')
      if (!confirmed) {
        return
      }
    }

    try {
      if (shouldStop) {
        await stopDisplayServer()
        setInfoMessage('Arret du serveur de diffusion demande.')
      } else {
        await startDisplayServer()
        setInfoMessage('Demarrage du serveur de diffusion demande.')
      }
    } catch (error) {
      console.error(error)
      setInfoMessage('Impossible de changer l\'etat du serveur de diffusion.')
    }
  }

  const updatePreviewScale = () => {
    if (!previewShellRef) {
      return
    }

    const widthRatio = previewShellRef.clientWidth / PREVIEW_WIDTH
    const heightRatio = previewShellRef.clientHeight / PREVIEW_HEIGHT
    setPreviewScale(Math.min(widthRatio, heightRatio))
  }

  const togglePreview = () => {
    setIsPreviewOpen((open) => {
      return !open
    })
  }

  onMount(() => {
    startSupervision()
    void fetchStreamDeck()

    if (previewShellRef) {
      previewResizeObserver = new ResizeObserver(() => {
        updatePreviewScale()
      })
      previewResizeObserver.observe(previewShellRef)
      updatePreviewScale()
    }

    // Force scale recalculation when preview is reopened
    createEffect(() => {
      if (isPreviewOpen()) {
        // Use setTimeout to ensure DOM has been updated
        const timeoutId = setTimeout(() => {
          updatePreviewScale()
        }, 0)
        onCleanup(() => clearTimeout(timeoutId))
      }
    })

    onCleanup(() => {
      stopSupervision?.()
      stopSupervision = null
      previewResizeObserver?.disconnect()
      previewResizeObserver = null
    })
  })

  return (
    <section class="dashboard-grid">
      <article class="panel">
        <div class="panel-head">
          <h2>Supervision</h2>
          <button class="ghost-button" onClick={startSupervision}>
            Reconnecter SSE
          </button>
        </div>

        <div class="status-list">
          <div class="status-item">
            <span>Flux supervision</span>
            <strong class={isSseConnected() ? 'status-ok' : 'status-ko'}>
              {isSseConnected() ? 'Connecte' : 'Deconnecte'}
            </strong>
          </div>
          <div class="status-item">
            <div class="status-item-row">
              <span>Serveur display</span>
              <button class="ghost-button" onClick={() => void toggleDisplayServer()}>
                {status().displayServerOnline ? 'Eteindre' : 'Allumer'}
              </button>
            </div>
            <strong class={status().displayServerOnline ? 'status-ok' : 'status-ko'}>
              {status().displayServerOnline ? 'Online' : 'Offline'}
            </strong>
          </div>
          <div class="status-item">
            <span>Clients connectes (display)</span>
            <strong>{status().connectedClients.display}</strong>
          </div>
          <div class="status-item">
            <span>Clients connectes (controls)</span>
            <strong>{status().connectedClients.controls}</strong>
          </div>
          <div class="status-item">
            <span>Mode display</span>
            <strong>{toDisplayModeLabel(status().displayMode)}</strong>
          </div>
          <div class="status-item track-item">
            <span>Track en cours</span>
            <Show when={currentTrack() !== null} fallback={<strong>Aucune diffusion active</strong>}>
              <div class="track-card">
                {currentTrack()?.coverUrl ? (
                  <img class="track-cover" src={currentTrack()?.coverUrl ?? ''} alt={`Cover de ${currentTrack()?.title ?? 'track'}`} />
                ) : (
                  <div class="cover-placeholder">No cover</div>
                )}
                <div>
                  <strong>{currentTrack()?.title}</strong>
                  <p>{currentTrack()?.artist}</p>
                  <code>{currentTrack()?.filePath}</code>
                </div>
              </div>
            </Show>
          </div>
        </div>

        <Show when={infoMessage().length > 0}>
          <p class="info-text">{infoMessage()}</p>
        </Show>
      </article>

      <article class="panel">
        <div class="panel-head">
          <h2>Streamdeck</h2>
          <button class="ghost-button" onClick={() => void fetchStreamDeck()}>
            Recharger
          </button>
        </div>
        <p>Commandes admin configurees dans le backend.</p>
        <Show when={!isLoadingButtons()} fallback={<p class="meta-text">Chargement des boutons...</p>}>
          <Show when={sortedButtons().length > 0} fallback={<p class="meta-text">Aucun bouton configure.</p>}>
            <div class="deck-grid" style={{ '--deck-columns': String(STREAMDECK_COLUMNS) }}>
              {sortedButtons().map((button, index) => (
                <ControlButton
                  label={button.label}
                  background={button.backgroundColor}
                  color={button.textColor}
                  icon={renderIcon(button.icon)}
                  isActive={status().displayMode === button.displayMode}
                  disabled={pendingButtons().has(button.id)}
                  style={toGridStyle(button, index)}
                  onClick={() => void triggerAction(button)}
                />
              ))}
            </div>
          </Show>
        </Show>
      </article>

      <article class="panel preview-panel">
        <div class="panel-head">
          <h2>Retour diffusion</h2>
          <button class="ghost-button" onClick={() => togglePreview()}>
            {isPreviewOpen() ? 'Fermer' : 'Ouvrir'}
          </button>
        </div>
        <Show when={isPreviewOpen()}>
          <div class="preview-frame-shell" ref={previewShellRef}>
            <div class="preview-frame-canvas" style={{ transform: `scale(${previewScale()})` }}>
              <iframe
                title="Trackker Diffusion"
                src={PREVIEW_URL}
                class="preview-frame"
                width={1920}
                height={1080}
              />
            </div>
          </div>
        </Show>
      </article>
    </section>
  )
}

