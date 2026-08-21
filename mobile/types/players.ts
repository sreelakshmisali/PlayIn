/** One entry in the sports catalogue, GET /sports. Positions is the closed
 * set of positions this sport offers; empty means it has none. */
export interface Sport {
  id: string
  slug: string
  name: string
  positions: string[]
}

/** One sport a player prefers, with the position they play in it. */
export interface PlayerSport {
  sport: Sport
  position?: string
}

/** A player's profile. Public and self views share this shape; it never
 * carries an email address, role or account flag. */
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
export interface SavePlayerProfilePayload {
  display_name: string
  image_url: string
  bio: string
  location: string
}

/** The body of POST /players/me/sports. */
export interface SetPreferredSportPayload {
  sport_id: string
  position: string
}
