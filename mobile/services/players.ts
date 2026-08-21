import type { PlayerProfile, SavePlayerProfilePayload, Sport } from '../types/players'
import { api } from './api'

/** Fetches the sports catalogue. Public, so it needs no session. */
export async function fetchSports(): Promise<Sport[]> {
  const body = await api.get<{ sports: Sport[] }>('/sports')
  return body.sports
}

/** Reads the signed-in player's profile. Rejects with a 404 before one exists. */
export function fetchMyPlayerProfile(): Promise<PlayerProfile> {
  return api.get<PlayerProfile>('/players/me')
}

/** Reads any player's public profile. */
export function fetchPlayerProfile(userId: string): Promise<PlayerProfile> {
  return api.get<PlayerProfile>(`/players/${encodeURIComponent(userId)}`)
}

/** Creates or replaces the signed-in player's profile. */
export function saveMyPlayerProfile(payload: SavePlayerProfilePayload): Promise<PlayerProfile> {
  return api.put<PlayerProfile>('/players/me', { body: payload })
}

/** Adds a preferred sport, or changes the position on one already chosen. */
export function setMySport(sportId: string, position = ''): Promise<PlayerProfile> {
  return api.post<PlayerProfile>('/players/me/sports', { body: { sport_id: sportId, position } })
}

/** Drops a preferred sport. */
export function removeMySport(sportId: string): Promise<PlayerProfile> {
  return api.delete<PlayerProfile>(`/players/me/sports/${encodeURIComponent(sportId)}`)
}
