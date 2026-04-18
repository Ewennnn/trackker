import { Show, createEffect, createMemo, createSignal } from 'solid-js'
import { Dashboard } from './components/Dashboard.tsx'
import { PinGate } from './components/PinGate.tsx'
import { getLockoutUntil, hasSessionAccess } from './services/security.ts'
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
  const [isAuthorized, setIsAuthorized] = createSignal(hasSessionAccess())
  const [lockoutUntilValue, setLockoutUntilValue] = createSignal(getLockoutUntil())
  const [themeMode, setThemeMode] = createSignal<ThemeMode>(getInitialTheme())

  const canAttemptAccess = createMemo(() => {
    const until = lockoutUntilValue()
    return until === null || until <= Date.now()
  })

  const handleAccessGranted = () => {
    setIsAuthorized(true)
    setLockoutUntilValue(getLockoutUntil())
  }

  const handleLockoutChanged = () => {
    setLockoutUntilValue(getLockoutUntil())
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

      <Show when={isAuthorized()} fallback={<PinGate canAttempt={canAttemptAccess()} lockoutUntil={lockoutUntilValue()} onAccessGranted={handleAccessGranted} onLockoutChanged={handleLockoutChanged} />}>
        <Dashboard />
      </Show>
    </main>
  )
}

export default App
