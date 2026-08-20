import { Link } from 'react-router-dom'

import { TurfStatusBadge } from '@/components/owners/TurfStatusBadge'
import type { Turf } from '@/lib/owners'

interface TurfCardProps {
  turf: Turf
  /** Where the card links to: the owner's edit screen or the public detail page. */
  to: string
  /** Show the status badge. On, for an owner's own list; off, in public browsing
   * where every result is APPROVED and the badge would be noise. */
  showStatus?: boolean
}

/** One turf in a list, mobile-first: a single tappable row, not a grid. */
export function TurfCard({ turf, to, showStatus }: TurfCardProps) {
  return (
    <Link
      to={to}
      className="flex items-center gap-3 rounded-xl border border-neutral-200 bg-white p-4 transition-colors hover:bg-neutral-50"
    >
      <Thumbnail url={turf.images[0]?.image_url} />

      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-2">
          <h3 className="truncate font-medium text-neutral-900">{turf.name}</h3>
          {showStatus && <TurfStatusBadge status={turf.status} />}
        </div>
        <p className="mt-0.5 truncate text-sm text-neutral-500">{turf.city}</p>

        {turf.sports.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {turf.sports.slice(0, 3).map((sport) => (
              <span
                key={sport.id}
                className="rounded-full bg-neutral-100 px-2 py-0.5 text-xs text-neutral-600"
              >
                {sport.name}
              </span>
            ))}
            {turf.sports.length > 3 && (
              <span className="px-1 text-xs text-neutral-400">+{turf.sports.length - 3}</span>
            )}
          </div>
        )}
      </div>
    </Link>
  )
}

function Thumbnail({ url }: { url: string | undefined }) {
  if (!url) {
    return (
      <span
        aria-hidden="true"
        className="grid size-14 shrink-0 place-items-center rounded-lg bg-neutral-100 text-xl"
      >
        🏟️
      </span>
    )
  }
  return (
    <img
      src={url}
      alt=""
      className="size-14 shrink-0 rounded-lg border border-neutral-200 object-cover"
    />
  )
}
