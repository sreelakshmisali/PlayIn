import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'

import { AuthContext, type AuthContextValue, type AuthStatus } from '@/auth/AuthContext'
import { onSessionEnded } from '@/lib/api'
import * as auth from '@/lib/auth'
import type { LoginPayload, Profile, RegisterPayload, Role } from '@/lib/auth'
import { clearTokens, onTokensChanged, readTokens, writeTokens } from '@/lib/tokens'

/**
 * Holds the session for the whole application.
 *
 * On mount it resolves the stored tokens against GET /auth/me rather than
 * trusting them. A token can be revoked, expired or issued for an account that
 * has since been deactivated, and only the server knows which.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<Profile | null>(null)

  const signOutLocally = useCallback(() => {
    clearTokens()
    setUser(null)
    setStatus('anonymous')
  }, [])

  // Resolve whatever is in storage once, at startup.
  useEffect(() => {
    const controller = new AbortController()

    if (!readTokens()) {
      setStatus('anonymous')
      return () => controller.abort()
    }

    auth
      .me(controller.signal)
      .then((profile) => {
        setUser(profile)
        setStatus('authenticated')
      })
      .catch(() => {
        if (controller.signal.aborted) return
        // The API client has already tried a refresh by this point, so a
        // failure here means the session is genuinely over.
        signOutLocally()
      })

    return () => controller.abort()
  }, [signOutLocally])

  // The API client signs out when a refresh fails mid-session. Without this the
  // UI would keep showing a signed-in shell over 401s.
  useEffect(() => onSessionEnded(signOutLocally), [signOutLocally])

  // Signing out in another tab signs out this one too.
  useEffect(
    () =>
      onTokensChanged(() => {
        if (!readTokens()) signOutLocally()
      }),
    [signOutLocally],
  )

  const adopt = useCallback((session: auth.Session): Profile => {
    writeTokens({
      accessToken: session.tokens.access_token,
      refreshToken: session.tokens.refresh_token,
    })
    setUser(session.user)
    setStatus('authenticated')
    return session.user
  }, [])

  const login = useCallback(
    async (payload: LoginPayload) => adopt(await auth.login(payload)),
    [adopt],
  )

  const register = useCallback(
    async (payload: RegisterPayload) => adopt(await auth.register(payload)),
    [adopt],
  )

  const logout = useCallback(async () => {
    const tokens = readTokens()

    // The local session is dropped whatever the server says. A failed revoke
    // must not leave someone looking signed in on a shared device.
    signOutLocally()

    if (tokens) {
      await auth.logout(tokens.refreshToken).catch(() => undefined)
    }
  }, [signOutLocally])

  const hasRole = useCallback(
    (...roles: Role[]) => user !== null && roles.includes(user.role),
    [user],
  )

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      ready: status !== 'loading',
      hasRole,
      login,
      register,
      logout,
    }),
    [status, user, hasRole, login, register, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
