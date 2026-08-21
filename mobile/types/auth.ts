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
