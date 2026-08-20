import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import { ProfileCard } from '@/components/players/ProfileCard'
import { ApiError } from '@/lib/api'
import { fetchPlayerProfile, type PlayerProfile } from '@/lib/players'

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; profile: PlayerProfile }
  | { kind: 'missing' }
  | { kind: 'failed'; message: string }

/**
 * Any player's profile, at /players/:userId.
 *
 * The route is outside the protected group: a profile is public, and requiring
 * a session to read one would defeat the point of having them.
 */
export function PublicPlayerPage() {
  const { userId = '' } = useParams()
  const { user } = useAuth()
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    const controller = new AbortController()
    setState({ kind: 'loading' })

    fetchPlayerProfile(userId, controller.signal)
      .then((profile) => setState({ kind: 'ready', profile }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return

        if (error instanceof ApiError && error.status === 404) {
          setState({ kind: 'missing' })
          return
        }
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load this player.',
        })
      })

    return () => controller.abort()
  }, [userId])

  if (state.kind === 'loading') {
    return <p className="text-neutral-500">Loading player.</p>
  }

  if (state.kind === 'missing') {
    return (
      <section className="mx-auto w-full max-w-md text-center">
        <p className="text-sm font-medium text-pitch-600">404</p>
        <h1 className="mt-2 text-xl font-semibold tracking-tight text-neutral-900">
          No player here
        </h1>
        <p className="mt-3 text-neutral-600">
          This player does not exist, or has not set up a profile yet.
        </p>
        <Link to="/" className="mt-6 inline-block text-sm font-medium text-pitch-700 underline">
          Back to home
        </Link>
      </section>
    )
  }

  if (state.kind === 'failed') {
    return (
      <div className="mx-auto w-full max-w-md">
        <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-sm text-red-800">{state.message}</p>
        </div>
      </div>
    )
  }

  const isOwnProfile = user?.id === state.profile.user_id

  return (
    <div className="mx-auto w-full max-w-md">
      <ProfileCard profile={state.profile} />

      {isOwnProfile && (
        <Link
          to="/profile/edit"
          className="mt-8 block rounded-lg border border-neutral-300 bg-white px-4 py-3 text-center text-base font-medium text-neutral-800 transition-colors hover:bg-neutral-50"
        >
          Edit your profile
        </Link>
      )}
    </div>
  )
}
