import type { Booking, BookingPage } from '../types/bookings'
import { api } from './api'

/** Fetches one page of the signed-in player's own bookings, most recent
 * first — matches `GET /players/me/bookings?limit=&offset=`. Both
 * parameters are optional; omitting them takes the API's own defaults. */
export function fetchMyBookings(limit?: number, offset?: number): Promise<BookingPage> {
  return api.get<BookingPage>('/players/me/bookings', { query: { limit, offset } })
}

/** Reads one of the signed-in player's own bookings. */
export function fetchBooking(bookingId: string): Promise<Booking> {
  return api.get<Booking>(`/players/me/bookings/${encodeURIComponent(bookingId)}`)
}

/** Reserves one open slot for the signed-in player. */
export function createBooking(turfSlotId: string): Promise<Booking> {
  return api.post<Booking>('/players/me/bookings', { body: { turf_slot_id: turfSlotId } })
}

/** Cancels one of the signed-in player's own CONFIRMED bookings, releasing
 * its slot back to OPEN. */
export function cancelBooking(bookingId: string): Promise<Booking> {
  return api.post<Booking>(`/players/me/bookings/${encodeURIComponent(bookingId)}/cancel`, {})
}
