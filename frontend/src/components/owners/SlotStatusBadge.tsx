import type { Slot } from '@/lib/availability'

/**
 * A slot's status from the owner's point of view: their own OPEN/BLOCKED
 * choice, distinguished from "unavailable because of a date or time-range
 * block" even though both end up unavailable to a player. The distinction
 * matters here because only the first is something this screen's slot list
 * can undo with a single toggle.
 */
export function SlotStatusBadge({ slot }: { slot: Slot }) {
  if (slot.status === 'BLOCKED') {
    return (
      <span className="shrink-0 rounded-full bg-red-50 px-2.5 py-1 text-xs font-medium text-red-700">
        Blocked
      </span>
    )
  }
  if (!slot.available) {
    return (
      <span className="shrink-0 rounded-full bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-700">
        Unavailable
      </span>
    )
  }
  return (
    <span className="shrink-0 rounded-full bg-pitch-50 px-2.5 py-1 text-xs font-medium text-pitch-700">
      Open
    </span>
  )
}
