import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { fetchPendingTurfs } from '@/lib/admin'

/**
 * The admin home. Two entry points, matching the two things this phase
 * covers: turf moderation and basic user management. The pending count is a
 * plain list length, not a stat worth its own endpoint or storage.
 */
export function AdminDashboardPage() {
  const [pendingCount, setPendingCount] = useState<number | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    fetchPendingTurfs(controller.signal)
      .then((turfs) => setPendingCount(turfs.length))
      .catch(() => {
        // The count is a convenience, not the page's purpose: a failed fetch
        // just leaves the tile without a badge rather than showing an error.
      })

    return () => controller.abort()
  }, [])

  return (
    <div className="mx-auto w-full max-w-md">
      <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">Admin</h1>
      <p className="mt-1 text-neutral-600">Turf moderation and account management.</p>

      <div className="mt-6 grid gap-3">
        <Tile
          to="/admin/turfs"
          title="Pending turfs"
          description="Review turfs submitted for approval."
          badge={pendingCount !== null && pendingCount > 0 ? pendingCount : undefined}
        />
        <Tile
          to="/admin/users"
          title="Users"
          description="Look up an account and activate or deactivate it."
        />
      </div>
    </div>
  )
}

function Tile({
  to,
  title,
  description,
  badge,
}: {
  to: string
  title: string
  description: string
  badge?: number | undefined
}) {
  return (
    <Link
      to={to}
      className="flex items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-white p-4 transition-colors hover:bg-neutral-50"
    >
      <div className="min-w-0">
        <h2 className="font-medium text-neutral-900">{title}</h2>
        <p className="mt-0.5 text-sm text-neutral-500">{description}</p>
      </div>
      {badge !== undefined && (
        <span className="shrink-0 rounded-full bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-700">
          {badge}
        </span>
      )}
    </Link>
  )
}
