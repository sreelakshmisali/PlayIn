/**
 * Token storage.
 *
 * Tokens live in localStorage. That trades some XSS exposure for surviving a
 * page reload and for working with the token-based API the backend exposes.
 * The alternative, an httpOnly cookie, would need the API to set cookies and a
 * CSRF defence on every write, which is a larger change than Phase 1 calls for.
 *
 * Everything that touches storage goes through this module, so moving to
 * cookies later is one file.
 */

const ACCESS_KEY = 'playhub.access_token'
const REFRESH_KEY = 'playhub.refresh_token'

export interface StoredTokens {
  accessToken: string
  refreshToken: string
}

/** Reads the stored pair, or null when the user is signed out. */
export function readTokens(): StoredTokens | null {
  const accessToken = safeRead(ACCESS_KEY)
  const refreshToken = safeRead(REFRESH_KEY)

  if (!accessToken || !refreshToken) return null
  return { accessToken, refreshToken }
}

/** Replaces the stored pair. */
export function writeTokens(tokens: StoredTokens): void {
  safeWrite(ACCESS_KEY, tokens.accessToken)
  safeWrite(REFRESH_KEY, tokens.refreshToken)
}

/** Removes the stored pair. */
export function clearTokens(): void {
  safeRemove(ACCESS_KEY)
  safeRemove(REFRESH_KEY)
}

/**
 * Subscribes to sign-in and sign-out in other tabs.
 *
 * The storage event only fires in the tabs that did not make the change, which
 * is exactly what is wanted: logging out in one tab logs out the rest.
 */
export function onTokensChanged(listener: () => void): () => void {
  const handler = (event: StorageEvent) => {
    if (event.key === ACCESS_KEY || event.key === REFRESH_KEY || event.key === null) {
      listener()
    }
  }

  window.addEventListener('storage', handler)
  return () => window.removeEventListener('storage', handler)
}

// Storage throws in private browsing modes and when the quota is exhausted. A
// failure there must not take the application down, so it degrades to an
// in-memory session instead.
const memory = new Map<string, string>()

function safeRead(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return memory.get(key) ?? null
  }
}

function safeWrite(key: string, value: string): void {
  try {
    window.localStorage.setItem(key, value)
  } catch {
    memory.set(key, value)
  }
}

function safeRemove(key: string): void {
  try {
    window.localStorage.removeItem(key)
  } catch {
    memory.delete(key)
  }
}
