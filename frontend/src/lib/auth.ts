import { api } from '@/lib/api'

/** A user's authorisation level. Mirrors the backend Role type. */
export type Role = 'PLAYER' | 'OWNER' | 'ADMIN'

/** Roles a person can choose for themselves at signup. ADMIN is granted out of band. */
export const SELF_ASSIGNABLE_ROLES: readonly Role[] = ['PLAYER', 'OWNER']

/** The client-facing view of a user. Never carries a password hash. */
export interface Profile {
  id: string
  email: string
  full_name: string
  role: Role
  is_active: boolean
  created_at: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: string
  /** Access token lifetime in seconds. */
  expires_in: number
}

/** The body of a successful register, login or refresh. */
export interface Session {
  user: Profile
  tokens: TokenPair
}

export interface RegisterPayload {
  email: string
  password: string
  full_name: string
  role: Role
}

export interface LoginPayload {
  email: string
  password: string
}

export function register(payload: RegisterPayload): Promise<Session> {
  return api.post<Session>('/auth/register', { body: payload, skipAuth: true })
}

export function login(payload: LoginPayload): Promise<Session> {
  return api.post<Session>('/auth/login', { body: payload, skipAuth: true })
}

/**
 * Exchanges a refresh token for a new pair. The old token is spent: the server
 * revokes it, so this cannot be retried with the same input.
 */
export function refresh(refreshToken: string): Promise<Session> {
  return api.post<Session>('/auth/refresh', {
    body: { refresh_token: refreshToken },
    skipAuth: true,
    skipRefresh: true,
  })
}

/** Revokes a refresh token. The server answers 204 even if it was already dead. */
export function logout(refreshToken: string): Promise<void> {
  return api.post<void>('/auth/logout', {
    body: { refresh_token: refreshToken },
    skipAuth: true,
    skipRefresh: true,
  })
}

/** Reads the signed-in user. Requires a valid access token. */
export function me(signal?: AbortSignal): Promise<Profile> {
  return api.get<Profile>('/auth/me', signal ? { signal } : {})
}
