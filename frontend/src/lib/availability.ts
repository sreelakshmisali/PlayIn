import { api } from '@/lib/api'
import type { Turf } from '@/lib/owners'

/** A slot's own management state, set directly by the owner. */
export type SlotStatus = 'OPEN' | 'BLOCKED'

/**
 * One bookable window on one turf on one date.
 *
 * `available` is computed by the API on every read, from `status` plus
 * whatever blocked dates and blocked time ranges apply; it is never something
 * this client derives or caches itself.
 */
export interface Slot {
  id: string
  date: string
  start_time: string
  end_time: string
  price: number
  status: SlotStatus
  available: boolean
  created_at: string
  updated_at: string
}

/** A whole day an owner has taken off availability. */
export interface BlockedDate {
  id: string
  date: string
  reason?: string
}

/** Part of one day taken off availability, e.g. a maintenance window. */
export interface BlockedTimeRange {
  id: string
  date: string
  start_time: string
  end_time: string
  reason?: string
}

export interface SlotSettingsPayload {
  slot_duration_minutes: number
  slot_price: number
}

// --- slot settings ---------------------------------------------------------

export function updateSlotSettings(turfId: string, payload: SlotSettingsPayload): Promise<Turf> {
  return api.patch<Turf>(`/owners/me/turfs/${encodeURIComponent(turfId)}/slot-settings`, { body: payload })
}

// --- slot generation and management -----------------------------------------

export async function generateSlots(turfId: string, from: string, to: string): Promise<Slot[]> {
  const body = await api.post<{ slots: Slot[] }>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/slots/generate`,
    { body: { from, to } },
  )
  return body.slots
}

export async function fetchMySlots(turfId: string, from: string, to: string): Promise<Slot[]> {
  const body = await api.get<{ slots: Slot[] }>(`/owners/me/turfs/${encodeURIComponent(turfId)}/slots`, {
    query: { from, to },
  })
  return body.slots
}

export function setSlotStatus(turfId: string, slotId: string, status: SlotStatus): Promise<Slot> {
  return api.patch<Slot>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/slots/${encodeURIComponent(slotId)}`,
    { body: { status } },
  )
}

export function deleteSlot(turfId: string, slotId: string): Promise<void> {
  return api.delete<void>(`/owners/me/turfs/${encodeURIComponent(turfId)}/slots/${encodeURIComponent(slotId)}`)
}

// --- blocked dates -----------------------------------------------------------

export async function fetchBlockedDates(turfId: string): Promise<BlockedDate[]> {
  const body = await api.get<{ blocked_dates: BlockedDate[] }>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/blocked-dates`,
  )
  return body.blocked_dates
}

export function blockDate(turfId: string, date: string, reason: string): Promise<BlockedDate> {
  return api.post<BlockedDate>(`/owners/me/turfs/${encodeURIComponent(turfId)}/blocked-dates`, {
    body: { date, reason },
  })
}

export function unblockDate(turfId: string, blockedDateId: string): Promise<void> {
  return api.delete<void>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/blocked-dates/${encodeURIComponent(blockedDateId)}`,
  )
}

// --- blocked time ranges -----------------------------------------------------

export async function fetchBlockedTimeRanges(turfId: string): Promise<BlockedTimeRange[]> {
  const body = await api.get<{ blocked_time_ranges: BlockedTimeRange[] }>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/blocked-time-ranges`,
  )
  return body.blocked_time_ranges
}

export interface BlockTimeRangePayload {
  date: string
  start_time: string
  end_time: string
  reason: string
}

export function blockTimeRange(turfId: string, payload: BlockTimeRangePayload): Promise<BlockedTimeRange> {
  return api.post<BlockedTimeRange>(`/owners/me/turfs/${encodeURIComponent(turfId)}/blocked-time-ranges`, {
    body: payload,
  })
}

export function unblockTimeRange(turfId: string, blockedTimeRangeId: string): Promise<void> {
  return api.delete<void>(
    `/owners/me/turfs/${encodeURIComponent(turfId)}/blocked-time-ranges/${encodeURIComponent(blockedTimeRangeId)}`,
  )
}

// --- public availability -----------------------------------------------------

export async function fetchPublicAvailability(turfId: string, date: string): Promise<Slot[]> {
  const body = await api.get<{ date: string; slots: Slot[] }>(
    `/turfs/${encodeURIComponent(turfId)}/availability`,
    { query: { date } },
  )
  return body.slots
}
