import type { Booking } from '../types/bookings'
import { api } from './api'

/**
 * Fetch the signed-in player's bookings. The endpoint does not exist yet
 * (booking is Phase 6 of the slot model), so this returns an empty list
 * until the backend ships `GET /players/me/bookings`.
 *
 * When the API is ready, remove the early return and uncomment the real
 * call — the response shape (`{ bookings: Booking[] }`) is already typed.
 */
export async function fetchMyBookings(): Promise<Booking[]> {
  // TODO: uncomment when GET /players/me/bookings exists
  // const body = await api.get<{ bookings: Booking[] }>('/players/me/bookings')
  // return body.bookings
  return []
}
