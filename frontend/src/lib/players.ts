import { api } from '@/lib/api'

/** One entry in the sports catalogue. */
export interface Sport {
  id: string
  slug: string
  name: string
  /**
   * The closed set of positions this sport offers. Empty for sports that have
   * none, which is how the UI knows whether to show a position picker at all.
   */
  positions: string[]
}

/** A sport a player prefers, with the position they play in it. */
export interface PlayerSport {
  sport: Sport
  position?: string
}

/**
 * A player's public profile.
 *
 * Everything here comes from the profile tables. No email address, role or
 * account flag is part of this shape, on the server or here.
 */
export interface PlayerProfile {
  user_id: string
  display_name: string
  image_url?: string
  bio?: string
  location?: string
  sports: PlayerSport[]
  created_at: string
  updated_at: string
}

/** The body of PUT /players/me. A full representation: omitted fields clear. */
export interface SaveProfilePayload {
  display_name: string
  image_url: string
  bio: string
  location: string
}

/** Fetches the sports catalogue. Public, so it needs no session. */
export async function fetchSports(signal?: AbortSignal): Promise<Sport[]> {
  const body = await api.get<{ sports: Sport[] }>('/sports', signal ? { signal } : {})
  return body.sports
}

/** Reads the signed-in player's profile. Rejects with a 404 before one exists. */
export function fetchMyProfile(signal?: AbortSignal): Promise<PlayerProfile> {
  return api.get<PlayerProfile>('/players/me', signal ? { signal } : {})
}

/** Reads any player's public profile. */
export function fetchPlayerProfile(userId: string, signal?: AbortSignal): Promise<PlayerProfile> {
  return api.get<PlayerProfile>(`/players/${encodeURIComponent(userId)}`, signal ? { signal } : {})
}

/** Creates or replaces the signed-in player's profile. */
export function saveMyProfile(payload: SaveProfilePayload): Promise<PlayerProfile> {
  return api.put<PlayerProfile>('/players/me', { body: payload })
}

/**
 * Adds a preferred sport, or changes the position on one already chosen.
 * The server treats this as an upsert, so the caller does not have to know
 * which of the two it is doing.
 */
export function setMySport(sportId: string, position?: string): Promise<PlayerProfile> {
  return api.post<PlayerProfile>('/players/me/sports', {
    body: { sport_id: sportId, position: position ?? '' },
  })
}

/** Drops a preferred sport. */
export function removeMySport(sportId: string): Promise<PlayerProfile> {
  return api.delete<PlayerProfile>(`/players/me/sports/${encodeURIComponent(sportId)}`)
}
