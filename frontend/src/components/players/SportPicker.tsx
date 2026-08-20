import type { PlayerSport, Sport } from '@/lib/players'

interface SportPickerProps {
  sports: Sport[]
  chosen: PlayerSport[]
  /** The sport id currently being written, so its row can show progress. */
  pending: string | null
  onToggle: (sport: Sport, selected: boolean) => void
  onPositionChange: (sport: Sport, position: string) => void
}

/**
 * Preferred sport selection.
 *
 * Each change is written on its own through the sports sub-resource rather than
 * collected into a save button, because that is the shape of the API: there is
 * no endpoint that replaces the whole list, and inventing a client-side diff to
 * simulate one would only add a way for the two to disagree.
 */
export function SportPicker({
  sports,
  chosen,
  pending,
  onToggle,
  onPositionChange,
}: SportPickerProps) {
  const chosenBySportId = new Map(chosen.map((entry) => [entry.sport.id, entry]))

  return (
    <ul className="space-y-2">
      {sports.map((sport) => {
        const selection = chosenBySportId.get(sport.id)
        const isChosen = selection !== undefined
        const isPending = pending === sport.id

        return (
          <li
            key={sport.id}
            className={[
              'rounded-lg border transition-colors',
              isChosen ? 'border-pitch-600 bg-pitch-50' : 'border-neutral-300 bg-white',
              isPending ? 'opacity-60' : '',
            ].join(' ')}
          >
            <label className="flex cursor-pointer items-center gap-3 px-3.5 py-3">
              <input
                type="checkbox"
                checked={isChosen}
                disabled={isPending}
                onChange={(event) => onToggle(sport, event.target.checked)}
                className="size-4 shrink-0 accent-pitch-600"
              />
              <span className="flex-1 text-sm font-medium text-neutral-900">{sport.name}</span>
              {isPending && <span className="text-xs text-neutral-500">Saving</span>}
            </label>

            {/* A position picker only appears for sports that have positions,
                which the catalogue states rather than the client assuming. */}
            {isChosen && sport.positions.length > 0 && (
              <div className="border-t border-pitch-600/20 px-3.5 py-3">
                <label className="block text-xs font-medium text-neutral-700">
                  Position
                  <select
                    value={selection.position ?? ''}
                    disabled={isPending}
                    onChange={(event) => onPositionChange(sport, event.target.value)}
                    className="mt-1.5 block w-full rounded-lg border border-neutral-300 bg-white px-3 py-2.5 text-base text-neutral-900 focus:outline-2 focus:outline-offset-1 focus:outline-pitch-600"
                  >
                    <option value="">No position</option>
                    {sport.positions.map((position) => (
                      <option key={position} value={position}>
                        {position}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            )}
          </li>
        )
      })}
    </ul>
  )
}
