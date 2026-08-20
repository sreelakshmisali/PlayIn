import { api } from '@/lib/api'

/**
 * An owner's business profile.
 *
 * Everything here comes from the profile table. No email address, role or
 * account flag is part of this shape, on the server or here.
 */
export interface OwnerProfile {
  user_id: string
  display_name: string
  phone?: string
  description?: string
  created_at: string
  updated_at: string
}

/** The body of PUT /owners/me. A full representation: omitted fields clear. */
export interface SaveOwnerProfilePayload {
  display_name: string
  phone: string
  description: string
}

/** A sport reference as it appears on a turf: no position, unlike a player's. */
export interface SportRef {
  id: string
  slug: string
  name: string
}

/** One entry in the amenities catalogue. */
export interface Amenity {
  id: string
  slug: string
  name: string
}

/** One photo attached to a turf. */
export interface TurfImage {
  id: string
  image_url: string
}

/** A turf's place in the approval workflow. Mirrors the backend Status type. */
export type TurfStatus = 'DRAFT' | 'PENDING_APPROVAL' | 'APPROVED' | 'REJECTED' | 'SUSPENDED'

/**
 * A turf listing.
 *
 * The same shape serves the owner's own view and the public view: nothing
 * here needs withholding from either.
 */
export interface Turf {
  id: string
  owner_display_name: string
  name: string
  description?: string
  address: string
  city: string
  latitude?: number
  longitude?: number
  capacity?: number
  opening_time: string
  closing_time: string
  status: TurfStatus
  /** Set only for REJECTED and SUSPENDED: why an admin moved it there. */
  moderation_reason?: string
  sports: SportRef[]
  amenities: Amenity[]
  images: TurfImage[]
  created_at: string
  updated_at: string
}

/** The body of POST /owners/me/turfs and PUT /owners/me/turfs/{id}. */
export interface SaveTurfPayload {
  name: string
  description: string
  address: string
  city: string
  latitude: number | null
  longitude: number | null
  capacity: number | null
  opening_time: string
  closing_time: string
}

// --- owner profile -----------------------------------------------------------

export function fetchMyOwnerProfile(signal?: AbortSignal): Promise<OwnerProfile> {
  return api.get<OwnerProfile>('/owners/me', signal ? { signal } : {})
}

export function saveMyOwnerProfile(payload: SaveOwnerProfilePayload): Promise<OwnerProfile> {
  return api.put<OwnerProfile>('/owners/me', { body: payload })
}

// --- amenities catalogue -------------------------------------------------------

export async function fetchAmenities(signal?: AbortSignal): Promise<Amenity[]> {
  const body = await api.get<{ amenities: Amenity[] }>('/amenities', signal ? { signal } : {})
  return body.amenities
}

// --- owner turf management ------------------------------------------------------

export async function fetchMyTurfs(signal?: AbortSignal): Promise<Turf[]> {
  const body = await api.get<{ turfs: Turf[] }>('/owners/me/turfs', signal ? { signal } : {})
  return body.turfs
}

export function fetchMyTurf(turfId: string, signal?: AbortSignal): Promise<Turf> {
  return api.get<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}`, signal ? { signal } : {})
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

/** Submits a DRAFT or REJECTED turf for review. */
export function submitTurf(turfId: string): Promise<Turf> {
  return api.post<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}/submit`, {})
}

/** Attaches a sport to a turf, or is a no-op if it is already attached. */
export function addTurfSport(turfId: string, sportId: string): Promise<Turf> {
  return api.post<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}/sports`, {
    body: { sport_id: sportId },
  })
}

export function removeTurfSport(turfId: string, sportId: string): Promise<Turf> {
  return api.delete<Turf>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/sports/${encodeURIComponent(sportId)}`,
  )
}

/** Attaches an amenity to a turf, or is a no-op if it is already attached. */
export function addTurfAmenity(turfId: string, amenityId: string): Promise<Turf> {
  return api.post<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}/amenities`, {
    body: { amenity_id: amenityId },
  })
}

export function removeTurfAmenity(turfId: string, amenityId: string): Promise<Turf> {
  return api.delete<Turf>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/amenities/${encodeURIComponent(amenityId)}`,
  )
}

export function addTurfImage(turfId: string, imageUrl: string): Promise<Turf> {
  return api.post<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}/images`, {
    body: { image_url: imageUrl },
  })
}

export function removeTurfImage(turfId: string, imageId: string): Promise<Turf> {
  return api.delete<Turf>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/images/${encodeURIComponent(imageId)}`,
  )
}

// --- public browsing -----------------------------------------------------------

export async function fetchPublicTurfs(signal?: AbortSignal): Promise<Turf[]> {
  const body = await api.get<{ turfs: Turf[] }>('/turfs', signal ? { signal } : {})
  return body.turfs
}

export function fetchPublicTurf(turfId: string, signal?: AbortSignal): Promise<Turf> {
  return api.get<Turf>(`/turfs/${encodeURIComponent(turfId)}`, signal ? { signal } : {})
}
