import { Show, createEffect, createSignal, onMount } from 'solid-js'
import { Dashboard } from './components/Dashboard.tsx'
import { PinGate } from './components/PinGate.tsx'
import { getSessionStatus } from './services/api.ts'
import './App.css'

type ThemeMode = 'light' | 'dark'

const THEME_STORAGE_KEY = 'trackker-control:theme'

const getInitialTheme = (): ThemeMode => {
  const storedTheme = localStorage.getItem(THEME_STORAGE_KEY)
  if (storedTheme === 'light' || storedTheme === 'dark') {
    return storedTheme
  }

  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function App() {
  const [isAuthorized, setIsAuthorized] = createSignal(false)
  const [isCheckingSession, setIsCheckingSession] = createSignal(true)
  const [sessionError, setSessionError] = createSignal<string | null>(null)
  const [kickNotice, setKickNotice] = createSignal<string | null>(null)
  const [themeMode, setThemeMode] = createSignal<ThemeMode>(getInitialTheme())

  const refreshSession = async () => {
    setSessionError(null)
    try {
      const authenticated = await getSessionStatus()
      setIsAuthorized(authenticated)
    } catch {
      setIsAuthorized(false)
      setSessionError('Impossible de verifier la session avec le backend.')
    } finally {
      setIsCheckingSession(false)
    }
  }

  onMount(() => {
    void refreshSession()
  })

  const handleAccessGranted = () => {
    setKickNotice(null)
    setIsAuthorized(true)
  }

  const handleConnectionClosed = () => {
    setKickNotice('Trackker Core n\'est plus accessible.')
    setIsAuthorized(false)
  }

  const dismissKickNotice = () => {
    setKickNotice(null)
  }

  const toggleTheme = () => {
    setThemeMode((mode) => (mode === 'dark' ? 'light' : 'dark'))
  }

  createEffect(() => {
    const mode = themeMode()
    document.documentElement.setAttribute('data-theme', mode)
    localStorage.setItem(THEME_STORAGE_KEY, mode)
  })

  return (
    <main class="app-shell">
      <header class="app-header">
        <div class="app-header-row">
          <h1>Trackker Control</h1>
          <button class="theme-toggle" type="button" onClick={toggleTheme}>
            {themeMode() === 'dark' ? 'Mode clair' : 'Mode sombre'}
          </button>
        </div>
        <p>Panneau de supervision et commandes du backend Trackker</p>
      </header>

      <Show
        when={!isCheckingSession()}
        fallback={
          <section class="panel pin-gate">
            <h2>Verification de la session...</h2>
          </section>
        }
      >
        <Show when={isAuthorized()} fallback={<PinGate onAccessGranted={handleAccessGranted} />}>
          <Dashboard onConnectionClosed={handleConnectionClosed} />
        </Show>
      </Show>

      <Show when={sessionError()}>
        {(message) => <p class="error-text">{message()}</p>}
      </Show>

      <Show when={kickNotice()}>
        {(message) => (
          <aside class="toast toast-error" role="status" aria-live="polite">
            <p>{message()}</p>
            <button class="toast-close" type="button" onClick={dismissKickNotice} aria-label="Fermer la notification">
              ×
            </button>
          </aside>
        )}
      </Show>
    </main>
  )
}

export default App
