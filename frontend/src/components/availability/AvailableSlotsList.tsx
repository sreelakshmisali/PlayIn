import type { Slot } from '@/lib/availability'

interface AvailableSlotsListProps {
  state: 'loading' | 'ready' | 'failed'
  slots: Slot[]
  errorMessage?: string
}

/**
 * A date's slots for a player or guest: available ones stand out, blocked
 * ones stay visible but visibly inert, so "nothing free today" reads
 * differently from "this turf has no slots configured yet".
 */
export function AvailableSlotsList({ state, slots, errorMessage }: AvailableSlotsListProps) {
  if (state === 'loading') {
    return (
      <p className="mt-3 text-sm text-neutral-500" role="status" aria-live="polite">
        Loading slots.
      </p>
    )
  }

  if (state === 'failed') {
    return (
      <div role="alert" className="mt-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3">
        <p className="text-sm text-red-800">{errorMessage ?? 'Could not load slots for this date.'}</p>
      </div>
    )
  }

  if (slots.length === 0) {
    return (
      <div className="mt-3 rounded-xl border border-dashed border-neutral-300 p-6 text-center">
        <p className="text-neutral-600">No slots are open for this date yet.</p>
      </div>
    )
  }

  return (
    <ul className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
      {slots.map((slot) => (
        <li key={slot.id}>
          <div
            className={[
              'rounded-xl border p-3 text-center',
              slot.available
                ? 'border-pitch-200 bg-pitch-50'
                : 'border-neutral-200 bg-neutral-100 opacity-60',
            ].join(' ')}
          >
            <p className={`text-sm font-medium ${slot.available ? 'text-pitch-800' : 'text-neutral-500'}`}>
              {slot.start_time}
            </p>
            <p className={`mt-0.5 text-xs ${slot.available ? 'text-pitch-700' : 'text-neutral-400'}`}>
              {slot.available ? `₹${slot.price}` : 'Unavailable'}
            </p>
          </div>
        </li>
      ))}
    </ul>
  )
}
