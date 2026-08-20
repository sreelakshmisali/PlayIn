import { createContext } from 'react'

import type { LoginPayload, Profile, RegisterPayload, Role } from '@/lib/auth'

/**
 * Session state.
 *
 * `loading` is its own state rather than `user === null`, because the two mean
 * different things on first paint: "we have not asked yet" must not render the
 * signed-out UI or bounce a signed-in user to the login screen.
 */
export type AuthStatus = 'loading' | 'authenticated' | 'anonymous'

export interface AuthContextValue {
  status: AuthStatus
  user: Profile | null
  /** True once the session has been resolved, either way. */
  ready: boolean
  hasRole: (...roles: Role[]) => boolean
  login: (payload: LoginPayload) => Promise<Profile>
  register: (payload: RegisterPayload) => Promise<Profile>
  logout: () => Promise<void>
}

// Undefined is the "no provider above me" signal, which useAuth turns into a
// thrown error rather than a confusing undefined user.
export const AuthContext = createContext<AuthContextValue | undefined>(undefined)
