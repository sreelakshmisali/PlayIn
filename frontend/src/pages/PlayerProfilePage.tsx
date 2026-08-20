import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import { ProfileCard } from '@/components/players/ProfileCard'
import { ApiError } from '@/lib/api'
import { fetchMyProfile, type PlayerProfile } from '@/lib/players'

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; profile: PlayerProfile }
  | { kind: 'missing' }
  | { kind: 'failed'; message: string }

/**
 * The signed-in player's own profile.
 *
 * A player who has not set one up yet is not an error state: the API answers
 * 404 until PUT creates one, and that is the signal to show the invitation to
 * build it rather than a failure.
 */
export function PlayerProfilePage() {
  const { user } = useAuth()
  const [state, setState] = useState<State>({ kind: 'loading' })

  const load = useCallback((signal?: AbortSignal) => {
    setState({ kind: 'loading' })

    fetchMyProfile(signal)
      .then((profile) => setState({ kind: 'ready', profile }))
      .catch((error: unknown) => {
        if (signal?.aborted) return

        if (error instanceof ApiError && error.status === 404) {
          setState({ kind: 'missing' })
          return
        }
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load your profile.',
        })
      })
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    load(controller.signal)
    return () => controller.abort()
  }, [load])

  if (state.kind === 'loading') {
    return <p className="text-neutral-500">Loading your profile.</p>
  }

  if (state.kind === 'failed') {
    return (
      <div className="mx-auto w-full max-w-md">
        <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-sm text-red-800">{state.message}</p>
        </div>
        <button
          type="button"
          onClick={() => load()}
          className="mt-4 rounded-lg border border-neutral-300 bg-white px-4 py-2.5 text-sm font-medium text-neutral-800 hover:bg-neutral-50"
        >
          Try again
        </button>
      </div>
    )
  }

  if (state.kind === 'missing') {
    return <EmptyProfile />
  }

  return (
    <div className="mx-auto w-full max-w-md">
      <ProfileCard profile={state.profile} />

      <div className="mt-8 flex flex-col gap-3">
        <Link
          to="/profile/edit"
          className="rounded-lg bg-pitch-600 px-4 py-3 text-center text-base font-medium text-white transition-colors hover:bg-pitch-700"
        >
          Edit profile
        </Link>
        {user && (
          <Link
            to={`/players/${user.id}`}
            className="rounded-lg border border-neutral-300 bg-white px-4 py-3 text-center text-base font-medium text-neutral-800 transition-colors hover:bg-neutral-50"
          >
            View as others see it
          </Link>
        )}
      </div>
    </div>
  )
}

function EmptyProfile() {
  return (
    <section className="mx-auto w-full max-w-md text-center">
      <span
        aria-hidden="true"
        className="mx-auto grid size-16 place-items-center rounded-full bg-neutral-100 text-2xl"
      >
        🏅
      </span>
      <h1 className="mt-5 text-xl font-semibold tracking-tight text-neutral-900">
        Set up your player profile
      </h1>
      <p className="mt-3 text-neutral-600">
        Add a display name and the sports you play, so other players can find you.
      </p>
      <Link
        to="/profile/edit"
        className="mt-6 inline-block w-full rounded-lg bg-pitch-600 px-4 py-3 text-base font-medium text-white transition-colors hover:bg-pitch-700"
      >
        Create profile
      </Link>
    </section>
  )
}
