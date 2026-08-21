import { today as localToday, addDays } from '@/lib/date'

const DAY_COUNT = 14

function nextDates(count: number): string[] {
  const start = localToday()
  const out: string[] = []
  for (let i = 0; i < count; i++) {
    out.push(addDays(start, i))
  }
  return out
}

const WEEKDAY = new Intl.DateTimeFormat(undefined, { weekday: 'short' })
const DAY_NUMBER = new Intl.DateTimeFormat(undefined, { day: 'numeric' })

/**
 * A horizontally scrollable strip of the next two weeks, one tap per day.
 * This is the mobile-first date selector for browsing availability: a native
 * date input works but forces a modal picker before a first result shows;
 * this puts the closest, most likely choices one thumb-reach away instead.
 */
export function DayStrip({ value, onChange }: { value: string; onChange: (date: string) => void }) {
  const dates = nextDates(DAY_COUNT)

  return (
    <div
      role="tablist"
      aria-label="Choose a date"
      className="-mx-4 flex snap-x gap-2 overflow-x-auto px-4 pb-1 sm:mx-0 sm:px-0"
    >
      {dates.map((date) => {
        const selected = date === value
        const d = new Date(date + 'T00:00:00')
        return (
          <button
            key={date}
            type="button"
            role="tab"
            aria-selected={selected}
            onClick={() => onChange(date)}
            className={[
              'flex shrink-0 snap-start flex-col items-center rounded-xl border px-3.5 py-2 text-center transition-colors',
              selected
                ? 'border-pitch-600 bg-pitch-600 text-white'
                : 'border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-50',
            ].join(' ')}
          >
            <span className={`text-xs ${selected ? 'text-pitch-50' : 'text-neutral-400'}`}>
              {WEEKDAY.format(d)}
            </span>
            <span className="text-base font-semibold">{DAY_NUMBER.format(d)}</span>
          </button>
        )
      })}
    </div>
  )
}
