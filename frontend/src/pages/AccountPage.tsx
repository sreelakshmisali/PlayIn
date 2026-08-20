import { useEffect, useState } from 'react'

import { useAuth } from '@/auth/useAuth'
import { ApiError } from '@/lib/api'
import { me, type Profile } from '@/lib/auth'

/**
 * The signed-in view. It re-reads GET /auth/me on mount rather than rendering
 * the cached profile alone, which is what makes this page the end-to-end proof
 * that the stored access token authenticates a real request.
 */
export function AccountPage() {
  const { user, logout } = useAuth()
  const [fresh, setFresh] = useState<Profile | null>(null)
  const [error, setError] = useState('')
  const [signingOut, setSigningOut] = useState(false)

  useEffect(() => {
    const controller = new AbortController()

    me(controller.signal)
      .then(setFresh)
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return
        setError(cause instanceof ApiError ? cause.message : 'Could not load your account.')
      })

    return () => controller.abort()
  }, [])

  const profile = fresh ?? user
  if (!profile) return null

  return (
    <section className="mx-auto w-full max-w-md">
      <header className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-neutral-900">
            {profile.full_name}
          </h1>
          <p className="mt-1 text-sm text-neutral-500">{profile.email}</p>
        </div>
        <RoleBadge role={profile.role} />
      </header>

      {error && (
        <p role="alert" className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </p>
      )}

      <dl className="mt-6 divide-y divide-neutral-100 overflow-hidden rounded-xl border border-neutral-200 bg-white text-sm">
        <Row label="Account id" value={profile.id} mono />
        <Row label="Status" value={profile.is_active ? 'Active' : 'Deactivated'} />
        <Row label="Member since" value={new Date(profile.created_at).toLocaleDateString()} />
        <Row label="Verified by" value={fresh ? 'GET /api/v1/auth/me' : 'Cached session'} />
      </dl>

      <button
        type="button"
        onClick={() => {
          setSigningOut(true)
          void logout()
        }}
        disabled={signingOut}
        className="mt-6 w-full rounded-lg border border-neutral-300 bg-white px-4 py-3 text-base font-medium text-neutral-800 transition-colors hover:bg-neutral-50 disabled:cursor-not-allowed disabled:text-neutral-400"
      >
        {signingOut ? 'Signing out' : 'Sign out'}
      </button>
    </section>
  )
}

function RoleBadge({ role }: { role: Profile['role'] }) {
  return (
    <span className="shrink-0 rounded-full bg-pitch-50 px-3 py-1 text-xs font-medium text-pitch-700">
      {role}
    </span>
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
