interface Option {
  id: string
  name: string
}

interface CheckboxListProps {
  options: Option[]
  selectedIds: ReadonlySet<string>
  /** The id currently being written, so its row can show progress. */
  pendingId: string | null
  onToggle: (option: Option, selected: boolean) => void
}

/**
 * A generic multi-select checkbox list, used for both a turf's sports and its
 * amenities: neither carries a per-row attribute the way a player's preferred
 * sport carries a position, so one component serves both.
 *
 * Each change is written on its own the moment it is toggled, matching the
 * shape of the API: there is no endpoint that replaces the whole set.
 */
export function CheckboxList({ options, selectedIds, pendingId, onToggle }: CheckboxListProps) {
  return (
    <ul className="space-y-2">
      {options.map((option) => {
        const isSelected = selectedIds.has(option.id)
        const isPending = pendingId === option.id

        return (
          <li key={option.id}>
            <label
              className={[
                'flex cursor-pointer items-center gap-3 rounded-lg border px-3.5 py-3 transition-colors',
                isSelected ? 'border-pitch-600 bg-pitch-50' : 'border-neutral-300 bg-white hover:bg-neutral-50',
                isPending ? 'opacity-60' : '',
              ].join(' ')}
            >
              <input
                type="checkbox"
                checked={isSelected}
                disabled={isPending}
                onChange={(event) => onToggle(option, event.target.checked)}
                className="size-4 shrink-0 accent-pitch-600"
              />
              <span className="flex-1 text-sm font-medium text-neutral-900">{option.name}</span>
              {isPending && <span className="text-xs text-neutral-500">Saving</span>}
            </label>
          </li>
        )
      })}
    </ul>
  )
}
