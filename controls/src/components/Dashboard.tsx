import { Show, createSignal, onCleanup, onMount } from 'solid-js'
import {
  PREVIEW_URL,
  connectSupervisionStream,
  sendControlAction,
  type SupervisionStatus,
} from '../services/api.ts'
import { ControlButton } from './ControlButton.tsx'

const DEFAULT_STATUS: SupervisionStatus = {
  httpOnline: false,
  connectedClients: 0,
  currentTrack: null,
}

const PREVIEW_WIDTH = 1920
const PREVIEW_HEIGHT = 1080

const actionList = [
  {
    id: 'blackout',
    label: 'Blackout',
    background: '#b110e7',
    color: '#ffffff',
  },
  {
    id: 'freeze_tracking',
    label: 'Freeze Tracking',
    background: '#177fea',
    color: '#ffffff',
  },
] as const

export const Dashboard = () => {
  const [status, setStatus] = createSignal<SupervisionStatus>(DEFAULT_STATUS)
  const [isSseConnected, setIsSseConnected] = createSignal(false)
  const [isPreviewOpen, setIsPreviewOpen] = createSignal(true)
  const [previewScale, setPreviewScale] = createSignal(1)
  const [infoMessage, setInfoMessage] = createSignal('')
  const currentTrack = () => status().currentTrack
  let previewShellRef: HTMLDivElement | undefined
  let previewResizeObserver: ResizeObserver | null = null

  let stopSupervision: (() => void) | null = null

  const startSupervision = () => {
    stopSupervision?.()
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
        setInfoMessage('Connexion supervision SSE interrompue. Reconnexion en cours...')
      },
    })
  }

  const triggerAction = async (action: string) => {
    try {
      await sendControlAction(action)
      setInfoMessage(`Action envoyee: ${action}`)
    } catch (error) {
      console.error(error)
      setInfoMessage(`Echec de l'action: ${action}`)
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

  onMount(() => {
    startSupervision()

    if (previewShellRef) {
      previewResizeObserver = new ResizeObserver(() => {
        updatePreviewScale()
      })
      previewResizeObserver.observe(previewShellRef)
      updatePreviewScale()
    }

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
            <span>Serveur HTTP</span>
            <strong class={status().httpOnline ? 'status-ok' : 'status-ko'}>
              {status().httpOnline ? 'Online' : 'Offline'}
            </strong>
          </div>
          <div class="status-item">
            <span>Clients connectes</span>
            <strong>{status().connectedClients}</strong>
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
        <h2>Streamdeck</h2>
        <p>Commandes admin configurees dans le code.</p>
        <div class="deck-grid">
          {actionList.map((action) => (
            <ControlButton
              label={action.label}
              background={action.background}
              color={action.color}
              onClick={() => void triggerAction(action.id)}
            />
          ))}
        </div>
      </article>

      <article class="panel preview-panel">
        <div class="panel-head">
          <h2>Retour diffusion</h2>
          <button class="ghost-button" onClick={() => setIsPreviewOpen((open) => !open)}>
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

