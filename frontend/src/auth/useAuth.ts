import { useContext } from 'react'

import { AuthContext, type AuthContextValue } from '@/auth/AuthContext'

/**
 * Reads the session. Throws when used outside AuthProvider, so a missing
 * provider fails at the call site instead of silently reporting nobody is
 * signed in.
 */
export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext)

  if (value === undefined) {
    throw new Error('useAuth must be used inside an AuthProvider')
  }
  return value
}
