import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import { FormAlert } from '@/components/form/FormAlert'
import { deactivateUser, fetchAdminUser, reactivateUser } from '@/lib/admin'
import { ApiError } from '@/lib/api'
import type { Profile } from '@/lib/auth'

type LoadState = { kind: 'loading' } | { kind: 'ready' } | { kind: 'failed'; message: string }

/**
 * One account: its details and the activate/deactivate control.
 *
 * The control is hidden, not just disabled, when the viewed account is the
 * signed-in admin's own. The API refuses the action either way, but hiding it
 * here means the admin never sees a button that always fails.
 */
export function AdminUserDetailPage() {
  const { userId = '' } = useParams()
  const { user: viewer } = useAuth()

  const [load, setLoad] = useState<LoadState>({ kind: 'loading' })
  const [profile, setProfile] = useState<Profile | null>(null)
  const [pending, setPending] = useState(false)
  const [actionMessage, setActionMessage] = useState('')

  useEffect(() => {
    const controller = new AbortController()

    fetchAdminUser(userId, controller.signal)
      .then((loaded) => {
        setProfile(loaded)
        setLoad({ kind: 'ready' })
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setLoad({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load this user.',
        })
      })

    return () => controller.abort()
  }, [userId])

  async function toggleActive(nextActive: boolean) {
    setPending(true)
    setActionMessage('')
    try {
      setProfile(nextActive ? await reactivateUser(userId) : await deactivateUser(userId))
    } catch (error) {
      setActionMessage(error instanceof ApiError ? error.message : 'That action could not be completed.')
    } finally {
      setPending(false)
    }
  }

  if (load.kind === 'loading') {
    return <p className="text-neutral-500">Loading this user.</p>
  }

  if (load.kind === 'failed' || !profile) {
    return (
      <div className="mx-auto w-full max-w-md">
        <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-sm text-red-800">
            {load.kind === 'failed' ? load.message : 'This user could not be loaded.'}
          </p>
        </div>
        <Link to="/admin/users" className="mt-4 inline-block text-sm font-medium text-pitch-700 underline">
          Back to users
        </Link>
      </div>
    )
  }

  const isSelf = viewer?.id === profile.id

  return (
    <div className="mx-auto w-full max-w-md">
      <p className="text-sm">
        <Link to="/admin/users" className="text-pitch-700 underline">
          Users
        </Link>
      </p>

      <header className="mt-1 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="truncate text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
            {profile.full_name}
          </h1>
          <p className="mt-1 truncate text-sm text-neutral-500">{profile.email}</p>
        </div>
        <span className="shrink-0 rounded-full bg-pitch-50 px-3 py-1 text-xs font-medium text-pitch-700">
          {profile.role}
        </span>
      </header>

      <dl className="mt-6 divide-y divide-neutral-100 overflow-hidden rounded-xl border border-neutral-200 bg-white text-sm">
        <Row label="Account id" value={profile.id} mono />
        <Row label="Status" value={profile.is_active ? 'Active' : 'Deactivated'} />
        <Row label="Member since" value={new Date(profile.created_at).toLocaleDateString()} />
      </dl>

      {actionMessage && (
        <div className="mt-6">
          <FormAlert message={actionMessage} />
        </div>
      )}

      {isSelf ? (
        <p className="mt-6 rounded-lg bg-neutral-100 px-4 py-3 text-sm text-neutral-600">
          This is your own account. It cannot be deactivated from here.
        </p>
      ) : profile.is_active ? (
        <button
          type="button"
          onClick={() => void toggleActive(false)}
          disabled={pending}
          className="mt-6 w-full rounded-lg border border-red-200 bg-white px-4 py-3 text-base font-medium text-red-700 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:text-neutral-400"
        >
          {pending ? 'Deactivating' : 'Deactivate account'}
        </button>
      ) : (
        <button
          type="button"
          onClick={() => void toggleActive(true)}
          disabled={pending}
          className="mt-6 w-full rounded-lg bg-pitch-600 px-4 py-3 text-base font-medium text-white transition-colors hover:bg-pitch-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
        >
          {pending ? 'Reactivating' : 'Reactivate account'}
        </button>
      )}
    </div>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-4 px-4 py-3">
      <dt className="text-neutral-500">{label}</dt>
      <dd className={`text-right font-medium text-neutral-900 ${mono ? 'font-mono text-xs' : ''}`}>
        {value}
      </dd>
    </div>
  )
}
