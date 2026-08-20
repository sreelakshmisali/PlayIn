import { useEffect, useState } from 'react'

import { TurfCard } from '@/components/owners/TurfCard'
import { ApiError } from '@/lib/api'
import { fetchPublicTurfs, type Turf } from '@/lib/owners'

type State = { kind: 'loading' } | { kind: 'ready'; turfs: Turf[] } | { kind: 'failed'; message: string }

/**
 * Public turf browsing. Every result here is APPROVED: the API filters to
 * that status before this page ever sees a row, so there is nothing to filter
 * client-side and no status badge worth showing.
 */
export function PublicTurfListPage() {
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    const controller = new AbortController()

    fetchPublicTurfs(controller.signal)
      .then((turfs) => setState({ kind: 'ready', turfs }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load turfs.',
        })
      })

    return () => controller.abort()
  }, [])

  return (
    <div className="mx-auto w-full max-w-md">
      <header>
        <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
          Turfs near you
        </h1>
        <p className="mt-1 text-sm text-neutral-500">Approved turfs, open for booking soon.</p>
      </header>

      <div className="mt-6">
        {state.kind === 'loading' && <p className="text-neutral-500">Loading turfs.</p>}

        {state.kind === 'failed' && (
          <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
            <p className="text-sm text-red-800">{state.message}</p>
          </div>
        )}

        {state.kind === 'ready' && state.turfs.length === 0 && (
          <p className="text-neutral-500">No turfs are listed yet.</p>
        )}

        {state.kind === 'ready' && state.turfs.length > 0 && (
          <ul className="space-y-3">
            {state.turfs.map((turf) => (
              <li key={turf.id}>
                <TurfCard turf={turf} to={`/turfs/${turf.id}`} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
