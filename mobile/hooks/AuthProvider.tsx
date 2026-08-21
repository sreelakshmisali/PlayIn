import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'

import * as authApi from '../services/auth'
import { onSessionEnded } from '../services/api'
import { clearTokens, readTokens, writeTokens } from '../storage/tokens'
import type { LoginPayload, Profile, RegisterPayload, Role, Session } from '../types/auth'
import { AuthContext, type AuthContextValue, type AuthStatus } from './AuthContext'

/**
 * Holds the session for the whole app.
 *
 * On mount it resolves the stored tokens against GET /auth/me rather than
 * trusting them: a token can be revoked, expired or issued for an account
 * that has since been deactivated, and only the server knows which.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<Profile | null>(null)

  const signOutLocally = useCallback(() => {
    void clearTokens()
    setUser(null)
    setStatus('anonymous')
  }, [])

  // Resolve whatever is in storage once, at startup.
  useEffect(() => {
    let cancelled = false

    readTokens().then((tokens) => {
      if (cancelled) return
      if (!tokens) {
        setStatus('anonymous')
        return
      }

      authApi
        .me()
        .then((profile) => {
          if (cancelled) return
          setUser(profile)
          setStatus('authenticated')
        })
        .catch(() => {
          if (cancelled) return
          // The API client has already tried a refresh by this point, so a
          // failure here means the session is genuinely over.
          signOutLocally()
        })
    })

    return () => {
      cancelled = true
    }
  }, [signOutLocally])

  // The API client signs out when a refresh fails mid-session. Without this
  // the app would keep showing a signed-in shell over 401s.
  useEffect(() => onSessionEnded(signOutLocally), [signOutLocally])

  const adopt = useCallback(async (session: Session): Promise<Profile> => {
    await writeTokens({
      accessToken: session.tokens.access_token,
      refreshToken: session.tokens.refresh_token,
    })
    setUser(session.user)
    setStatus('authenticated')
    return session.user
  }, [])

  const login = useCallback(async (payload: LoginPayload) => adopt(await authApi.login(payload)), [adopt])

  const register = useCallback(
    async (payload: RegisterPayload) => adopt(await authApi.register(payload)),
    [adopt],
  )

  const logout = useCallback(async () => {
    const tokens = await readTokens()

    // The local session is dropped whatever the server says. A failed revoke
    // must not leave someone looking signed in on a shared device.
    signOutLocally()

    if (tokens) {
      await authApi.logout(tokens.refreshToken).catch(() => undefined)
    }
  }, [signOutLocally])

  const hasRole = useCallback((...roles: Role[]) => user !== null && roles.includes(user.role), [user])

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, ready: status !== 'loading', hasRole, login, register, logout }),
    [status, user, hasRole, login, register, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
