const ACCESS_SESSION_KEY = 'trackker-control:access-granted'
const FAILED_ATTEMPTS_KEY = 'trackker-control:failed-attempts'
const LOCKOUT_UNTIL_KEY = 'trackker-control:lockout-until'

const MAX_ATTEMPTS = 3
const ONE_WEEK_IN_MS = 7 * 24 * 60 * 60 * 1000

const safeGetNumber = (value: string | null): number | null => {
  if (value === null) {
    return null
  }

  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

export const hasSessionAccess = (): boolean => sessionStorage.getItem(ACCESS_SESSION_KEY) === '1'

export const grantSessionAccess = (): void => {
  sessionStorage.setItem(ACCESS_SESSION_KEY, '1')
  localStorage.removeItem(FAILED_ATTEMPTS_KEY)
  localStorage.removeItem(LOCKOUT_UNTIL_KEY)
}

export const getLockoutUntil = (): number | null => {
  const lockout = safeGetNumber(localStorage.getItem(LOCKOUT_UNTIL_KEY))

  if (lockout === null) {
    return null
  }

  if (lockout <= Date.now()) {
    localStorage.removeItem(LOCKOUT_UNTIL_KEY)
    localStorage.removeItem(FAILED_ATTEMPTS_KEY)
    return null
  }

  return lockout
}

export const getRemainingAttempts = (): number => {
  const attempts = safeGetNumber(localStorage.getItem(FAILED_ATTEMPTS_KEY)) ?? 0
  return Math.max(0, MAX_ATTEMPTS - attempts)
}

export const registerFailedAttempt = (): { lockoutUntil: number | null; remainingAttempts: number } => {
  const currentAttempts = safeGetNumber(localStorage.getItem(FAILED_ATTEMPTS_KEY)) ?? 0
  const nextAttempts = currentAttempts + 1
  localStorage.setItem(FAILED_ATTEMPTS_KEY, String(nextAttempts))

  if (nextAttempts >= MAX_ATTEMPTS) {
    const lockoutUntil = Date.now() + ONE_WEEK_IN_MS
    localStorage.setItem(LOCKOUT_UNTIL_KEY, String(lockoutUntil))
    return { lockoutUntil, remainingAttempts: 0 }
  }

  return {
    lockoutUntil: null,
    remainingAttempts: Math.max(0, MAX_ATTEMPTS - nextAttempts),
  }
}

