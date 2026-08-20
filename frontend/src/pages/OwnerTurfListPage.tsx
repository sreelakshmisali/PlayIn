import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { TurfCard } from '@/components/owners/TurfCard'
import { ApiError } from '@/lib/api'
import { fetchMyTurfs, type Turf } from '@/lib/owners'

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; turfs: Turf[] }
  | { kind: 'no-profile' }
  | { kind: 'failed'; message: string }

/** The signed-in owner's turf list, every status included. */
export function OwnerTurfListPage() {
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    const controller = new AbortController()

    fetchMyTurfs(controller.signal)
      .then((turfs) => setState({ kind: 'ready', turfs }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return

        // A turf list requires an owner profile to anchor to. Route this to
        // its own state with a way forward, not a bare error box.
        if (error instanceof ApiError && error.status === 404) {
          setState({ kind: 'no-profile' })
          return
        }
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load your turfs.',
        })
      })

    return () => controller.abort()
  }, [])

  return (
    <div className="mx-auto w-full max-w-md">
      <header className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
          Your turfs
        </h1>
        <Link
          to="/owner/turfs/new"
          className="shrink-0 rounded-lg bg-pitch-600 px-3.5 py-2 text-sm font-medium text-white transition-colors hover:bg-pitch-700"
        >
          New turf
        </Link>
      </header>

      <div className="mt-6">
        {state.kind === 'loading' && <p className="text-neutral-500">Loading your turfs.</p>}

        {state.kind === 'no-profile' && (
          <div className="rounded-xl border border-dashed border-neutral-300 p-6 text-center">
            <p className="text-neutral-600">Set up your owner profile before listing a turf.</p>
            <Link
              to="/owner/profile/edit"
              className="mt-4 inline-block rounded-lg bg-pitch-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-pitch-700"
            >
              Create your profile
            </Link>
          </div>
        )}

        {state.kind === 'failed' && (
          <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
            <p className="text-sm text-red-800">{state.message}</p>
          </div>
        )}

        {state.kind === 'ready' && state.turfs.length === 0 && (
          <div className="rounded-xl border border-dashed border-neutral-300 p-6 text-center">
            <p className="text-neutral-600">You have not listed a turf yet.</p>
            <Link
              to="/owner/turfs/new"
              className="mt-4 inline-block rounded-lg bg-pitch-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-pitch-700"
            >
              List your first turf
            </Link>
          </div>
        )}

        {state.kind === 'ready' && state.turfs.length > 0 && (
          <ul className="space-y-3">
            {state.turfs.map((turf) => (
              <li key={turf.id}>
                <TurfCard turf={turf} to={`/owner/turfs/${turf.id}/edit`} showStatus />
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
