/**
 * Booking types for the mobile client, matching the Go API's actual
 * response shapes (`backend/internal/bookings/model.go`).
 */

/** Where a booking sits in its lifecycle. The backend's set is closed at
 * two values — there is no third "completed" status; a CONFIRMED booking
 * whose date has passed is still, honestly, CONFIRMED. Screens derive
 * "upcoming vs past" from the date instead of a status the API doesn't
 * send. */
export type BookingStatus = 'CONFIRMED' | 'CANCELLED'

/** The slice of a turf a booking response carries, joined in so a booking
 * card never needs a second fetch. */
export interface BookingTurf {
  id: string
  name: string
  address: string
  city: string
}

/** A single booking, as returned by every `/players/me/bookings*` endpoint. */
export interface Booking {
  id: string
  turf: BookingTurf
  turf_slot_id: string
  date: string
  start_time: string
  end_time: string
  status: BookingStatus
  /** Price at time of booking, not the turf's current price. */
  price: number
  created_at: string
  updated_at: string
  cancelled_at?: string
}

/** One page of `GET /players/me/bookings`. */
export interface BookingPage {
  bookings: Booking[]
  total: number
  limit: number
  offset: number
}

/** The body of `POST /players/me/bookings`. */
export interface CreateBookingPayload {
  turf_slot_id: string
}
