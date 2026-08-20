import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { TurfCard } from '@/components/owners/TurfCard'
import { fetchPendingTurfs } from '@/lib/admin'
import { ApiError } from '@/lib/api'
import type { Turf } from '@/lib/owners'

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; turfs: Turf[] }
  | { kind: 'failed'; message: string }

/** Every turf currently waiting on a moderation decision. */
export function AdminPendingTurfsPage() {
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    const controller = new AbortController()

    fetchPendingTurfs(controller.signal)
      .then((turfs) => setState({ kind: 'ready', turfs }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load pending turfs.',
        })
      })

    return () => controller.abort()
  }, [])

  return (
    <div className="mx-auto w-full max-w-md">
      <p className="text-sm">
        <Link to="/admin" className="text-pitch-700 underline">
          Admin
        </Link>
      </p>

      <h1 className="mt-1 text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
        Pending turfs
      </h1>

      <div className="mt-6">
        {state.kind === 'loading' && <p className="text-neutral-500">Loading pending turfs.</p>}

        {state.kind === 'failed' && (
          <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
            <p className="text-sm text-red-800">{state.message}</p>
          </div>
        )}

        {state.kind === 'ready' && state.turfs.length === 0 && (
          <div className="rounded-xl border border-dashed border-neutral-300 p-6 text-center">
            <p className="text-neutral-600">Nothing is waiting for review.</p>
          </div>
        )}

        {state.kind === 'ready' && state.turfs.length > 0 && (
          <ul className="space-y-3">
            {state.turfs.map((turf) => (
              <li key={turf.id}>
                <TurfCard turf={turf} to={`/admin/turfs/${turf.id}`} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
