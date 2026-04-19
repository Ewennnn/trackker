import { Show, createSignal } from 'solid-js'
import { checkPinCode } from '../services/api.ts'
import {
  getRemainingAttempts,
  grantSessionAccess,
  registerFailedAttempt,
} from '../services/security.ts'

type PinGateProps = {
  canAttempt: boolean
  lockoutUntil: number | null
  onAccessGranted: () => void
  onLockoutChanged: () => void
}

export const PinGate = (props: PinGateProps) => {
  const [pin, setPin] = createSignal('')
  const [errorMessage, setErrorMessage] = createSignal<string | null>(null)
  const [remainingAttempts, setRemainingAttempts] = createSignal(getRemainingAttempts())

  const submitPin = async (event: SubmitEvent) => {
    event.preventDefault()

    if (!props.canAttempt) {
      setErrorMessage('Acces bloqué')
      return
    }

    if (!/^\d{6}$/.test(pin())) {
      setErrorMessage('Le code doit contenir exactement 6 chiffres.')
      return
    }

    checkPinCode(pin())
        .then(() => {
            console.log("SUCCESS")
            grantSessionAccess()
            props.onAccessGranted()
        })
        .catch(err => {
            console.log(err)
          const nextState = registerFailedAttempt()
          setRemainingAttempts(nextState.remainingAttempts)
          props.onLockoutChanged()

          if (nextState.lockoutUntil) {
            setErrorMessage('Acces bloque pendant une semaine apres 3 erreurs.')
            return
          }

          setErrorMessage(`Code invalide. Tentatives restantes: ${nextState.remainingAttempts}.`)
        })
  }

  return (
    <section class="panel pin-gate">
      <h2>Acces au panneau</h2>

      <Show when={props.canAttempt} fallback={<p class="lockout">Acces bloqué.</p>}>
        <form class="pin-form" onSubmit={submitPin}>
          <input
            class="pin-input"
            type="password"
            inputMode="numeric"
            autocomplete="one-time-code"
            maxlength={6}
            placeholder="000000"
            value={pin()}
            onInput={(event) => setPin(event.currentTarget.value.replace(/\D/g, '').slice(0, 6))}
          />
          <button class="primary-button" type="submit">
            Entrer
          </button>
        </form>

        <Show when={errorMessage()}>
          {(message) => <p class="error-text">{message()}</p>}
        </Show>

        <p class="attempt-hint">Tentatives restantes: {remainingAttempts()}</p>
      </Show>
    </section>
  )
}

