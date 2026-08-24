/** An owner's business profile. Never carries an email, role or account flag. */
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

/** A turf listing. The same shape serves the owner's own view and the public view. */
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
  /** Unset until the owner configures slot settings — both are set together. */
  slot_duration_minutes?: number
  slot_price?: number
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
