import { api } from '@/lib/api'
import type { Profile } from '@/lib/auth'
import type { Turf } from '@/lib/owners'

/** One page of GET /admin/users. */
export interface UserPage {
  users: Profile[]
  total: number
  limit: number
  offset: number
}

// --- turf moderation -----------------------------------------------------------

export async function fetchPendingTurfs(signal?: AbortSignal): Promise<Turf[]> {
  const body = await api.get<{ turfs: Turf[] }>('/admin/turfs/pending', signal ? { signal } : {})
  return body.turfs
}

export function fetchAdminTurf(turfId: string, signal?: AbortSignal): Promise<Turf> {
  return api.get<Turf>(`/admin/turfs/${encodeURIComponent(turfId)}`, signal ? { signal } : {})
}

export function approveTurf(turfId: string): Promise<Turf> {
  return api.post<Turf>(`/admin/turfs/${encodeURIComponent(turfId)}/approve`, {})
}

export function rejectTurf(turfId: string, reason: string): Promise<Turf> {
  return api.post<Turf>(`/admin/turfs/${encodeURIComponent(turfId)}/reject`, { body: { reason } })
}

export function suspendTurf(turfId: string, reason: string): Promise<Turf> {
  return api.post<Turf>(`/admin/turfs/${encodeURIComponent(turfId)}/suspend`, { body: { reason } })
}

export function restoreTurf(turfId: string): Promise<Turf> {
  return api.post<Turf>(`/admin/turfs/${encodeURIComponent(turfId)}/restore`, {})
}

// --- user management -------------------------------------------------------------

export function fetchUsers(
  params: { limit: number; offset: number },
  signal?: AbortSignal,
): Promise<UserPage> {
  return api.get<UserPage>('/admin/users', { query: params, ...(signal ? { signal } : {}) })
}

export function fetchAdminUser(userId: string, signal?: AbortSignal): Promise<Profile> {
  return api.get<Profile>(`/admin/users/${encodeURIComponent(userId)}`, signal ? { signal } : {})
}

export function deactivateUser(userId: string): Promise<Profile> {
  return api.post<Profile>(`/admin/users/${encodeURIComponent(userId)}/deactivate`, {})
}

export function reactivateUser(userId: string): Promise<Profile> {
  return api.post<Profile>(`/admin/users/${encodeURIComponent(userId)}/reactivate`, {})
}
