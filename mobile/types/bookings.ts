/**
 * Booking types for the mobile client.
 *
 * These match the shape the Go API will return once the booking endpoints
 * exist (Phase 6 of the slot model). Until then the service layer returns
 * an empty list and every screen that renders bookings handles empty state
 * gracefully.
 */

/** Where a booking sits in its lifecycle. */
export type BookingStatus = 'CONFIRMED' | 'COMPLETED' | 'CANCELLED'

/** Enough turf context to render a booking card without a second fetch. */
export interface BookingTurf {
  id: string
  name: string
  address: string
  city: string
  /** First image URL, if available. */
  image_url?: string
}

/** A single booking as returned by `GET /players/me/bookings`. */
export interface Booking {
  id: string
  turf: BookingTurf
  sport_name: string
  date: string
  start_time: string
  end_time: string
  /** Price at time of booking, not the turf's current price. */
  price: number
  status: BookingStatus
  created_at: string
}
