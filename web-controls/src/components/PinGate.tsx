import { Show, createSignal } from 'solid-js'
import { ApiError, checkPinCode } from '../services/api.ts'

type PinGateProps = {
  onAccessGranted: () => void
}

export const PinGate = (props: PinGateProps) => {
  const [pin, setPin] = createSignal('')
  const [errorMessage, setErrorMessage] = createSignal<string | null>(null)
  const [isSubmitting, setIsSubmitting] = createSignal(false)

  const submitPin = async (event: SubmitEvent) => {
    event.preventDefault()
    setErrorMessage(null)

    if (!/^\d{6}$/.test(pin())) {
      setErrorMessage('Le code doit contenir exactement 6 chiffres.')
      return
    }

    setIsSubmitting(true)

    try {
      await checkPinCode(pin())
      props.onAccessGranted()
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.status === 401) {
          setErrorMessage('Code invalide.')
          return
        }

        if (error.status === 429) {
          setErrorMessage('Acces temporairement bloque par le serveur. Reessaye plus tard.')
          return
        }
      }

      setErrorMessage('Impossible de verifier le code pour le moment.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section class="panel pin-gate">
      <h2>Acces au panneau</h2>

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
        <button class="primary-button" type="submit" disabled={isSubmitting()}>
          {isSubmitting() ? 'Verification...' : 'Entrer'}
        </button>
      </form>

      <Show when={errorMessage()}>
        {(message) => <p class="error-text">{message()}</p>}
      </Show>
    </section>
  )
}

