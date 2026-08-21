import type { LoginPayload, Profile, RegisterPayload, Session } from '../types/auth'
import { api } from './api'

export function register(payload: RegisterPayload): Promise<Session> {
  return api.post<Session>('/auth/register', { body: payload, skipAuth: true })
}

export function login(payload: LoginPayload): Promise<Session> {
  return api.post<Session>('/auth/login', { body: payload, skipAuth: true })
}

/**
 * Exchanges a refresh token for a new pair. The old token is spent: the
 * server revokes it, so this cannot be retried with the same input.
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
export function me(): Promise<Profile> {
  return api.get<Profile>('/auth/me')
}
