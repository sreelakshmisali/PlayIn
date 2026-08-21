import type { OwnerProfile, SaveOwnerProfilePayload, SaveTurfPayload, Turf } from '../types/owners'
import { api } from './api'

// --- owner profile -----------------------------------------------------------

export function fetchMyOwnerProfile(): Promise<OwnerProfile> {
  return api.get<OwnerProfile>('/owners/me')
}

export function saveMyOwnerProfile(payload: SaveOwnerProfilePayload): Promise<OwnerProfile> {
  return api.put<OwnerProfile>('/owners/me', { body: payload })
}

// --- owner turf management ------------------------------------------------------

export async function fetchMyTurfs(): Promise<Turf[]> {
  const body = await api.get<{ turfs: Turf[] }>('/owners/me/turfs')
  return body.turfs
}

export function fetchMyTurf(turfId: string): Promise<Turf> {
  return api.get<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}`)
}

export function createTurf(payload: SaveTurfPayload): Promise<Turf> {
  return api.post<Turf>('/owners/me/turfs', { body: payload })
}

export function updateTurf(turfId: string, payload: SaveTurfPayload): Promise<Turf> {
  return api.put<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}`, { body: payload })
}

export function deleteTurf(turfId: string): Promise<void> {
  return api.delete<void>(`/owners/me/turfs/${encodeURIComponent(turfId)}`)
}

/** Submits a DRAFT or REJECTED turf for admin review. */
export function submitTurf(turfId: string): Promise<Turf> {
  return api.post<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}/submit`, {})
}

// --- public browsing -----------------------------------------------------------

export async function fetchPublicTurfs(): Promise<Turf[]> {
  const body = await api.get<{ turfs: Turf[] }>('/turfs')
  return body.turfs
}

export function fetchPublicTurf(turfId: string): Promise<Turf> {
  return api.get<Turf>(`/turfs/${encodeURIComponent(turfId)}`)
}
