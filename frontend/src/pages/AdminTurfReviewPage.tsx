import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { ModerationActionForm } from '@/components/admin/ModerationActionForm'
import { FormAlert } from '@/components/form/FormAlert'
import { TurfStatusBadge } from '@/components/owners/TurfStatusBadge'
import {
  approveTurf,
  fetchAdminTurf,
  rejectTurf,
  restoreTurf,
  suspendTurf,
} from '@/lib/admin'
import { ApiError } from '@/lib/api'
import type { Turf } from '@/lib/owners'

type LoadState = { kind: 'loading' } | { kind: 'ready' } | { kind: 'failed'; message: string }

/**
 * One turf's moderation detail. Which actions are offered follows directly
 * from the turf's current status, matching the state machine the API itself
 * enforces: this page never lets a click be sent that the server would
 * refuse anyway.
 */
export function AdminTurfReviewPage() {
  const { turfId = '' } = useParams()

  const [load, setLoad] = useState<LoadState>({ kind: 'loading' })
  const [turf, setTurf] = useState<Turf | null>(null)
  const [pending, setPending] = useState(false)
  const [actionMessage, setActionMessage] = useState('')

  useEffect(() => {
    const controller = new AbortController()

    fetchAdminTurf(turfId, controller.signal)
      .then((loaded) => {
        setTurf(loaded)
        setLoad({ kind: 'ready' })
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setLoad({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load this turf.',
        })
      })

    return () => controller.abort()
  }, [turfId])

  async function run(action: () => Promise<Turf>) {
    setPending(true)
    setActionMessage('')
    try {
      setTurf(await action())
    } catch (error) {
      setActionMessage(error instanceof ApiError ? error.message : 'That action could not be completed.')
    } finally {
      setPending(false)
    }
  }

  if (load.kind === 'loading') {
    return <p className="text-neutral-500">Loading this turf.</p>
  }

  if (load.kind === 'failed' || !turf) {
    return (
      <div className="mx-auto w-full max-w-md">
        <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-sm text-red-800">
            {load.kind === 'failed' ? load.message : 'This turf could not be loaded.'}
          </p>
        </div>
        <Link to="/admin/turfs" className="mt-4 inline-block text-sm font-medium text-pitch-700 underline">
          Back to pending turfs
        </Link>
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-md">
      <p className="text-sm">
        <Link to="/admin/turfs" className="text-pitch-700 underline">
          Pending turfs
        </Link>
      </p>

      <header className="mt-1 flex items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
            {turf.name}
          </h1>
          <p className="mt-1 text-sm text-neutral-500">
            {turf.address}, {turf.city}
          </p>
          <p className="mt-1 text-sm text-neutral-500">Listed by {turf.owner_display_name}</p>
        </div>
        <TurfStatusBadge status={turf.status} />
      </header>

      {turf.description && (
        <p className="mt-4 whitespace-pre-line text-neutral-700">{turf.description}</p>
      )}

      <dl className="mt-6 divide-y divide-neutral-100 overflow-hidden rounded-xl border border-neutral-200 bg-white text-sm">
        <Row label="Hours" value={`${turf.opening_time} – ${turf.closing_time}`} />
        {turf.capacity !== undefined && <Row label="Capacity" value={`${turf.capacity} players`} />}
      </dl>

      {turf.sports.length > 0 && (
        <section className="mt-6">
          <h2 className="text-sm font-medium text-neutral-800">Sports</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {turf.sports.map((sport) => (
              <span
                key={sport.id}
                className="rounded-full bg-neutral-100 px-3 py-1 text-sm text-neutral-700"
              >
                {sport.name}
              </span>
            ))}
          </div>
        </section>
      )}

      {turf.moderation_reason && (
        <div className="mt-6 rounded-lg border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-xs font-medium text-red-700">Reason on file</p>
          <p className="mt-1 text-sm text-red-800">{turf.moderation_reason}</p>
        </div>
      )}

      {actionMessage && (
        <div className="mt-6">
          <FormAlert message={actionMessage} />
        </div>
      )}

      <div className="mt-6 space-y-3">
        {turf.status === 'PENDING_APPROVAL' && (
          <>
            <button
              type="button"
              onClick={() => void run(() => approveTurf(turf.id))}
              disabled={pending}
              className="w-full rounded-lg bg-pitch-600 px-4 py-3 text-base font-medium text-white transition-colors hover:bg-pitch-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
            >
              {pending ? 'Approving' : 'Approve'}
            </button>
            <ModerationActionForm
              label="Reject"
              pendingLabel="Rejecting"
              tone="danger"
              pending={pending}
              onSubmit={(reason) => run(() => rejectTurf(turf.id, reason))}
            />
          </>
        )}

        {turf.status === 'APPROVED' && (
          <ModerationActionForm
            label="Suspend"
            pendingLabel="Suspending"
            tone="warning"
            pending={pending}
            onSubmit={(reason) => run(() => suspendTurf(turf.id, reason))}
          />
        )}

        {turf.status === 'SUSPENDED' && (
          <button
            type="button"
            onClick={() => void run(() => restoreTurf(turf.id))}
            disabled={pending}
            className="w-full rounded-lg bg-pitch-600 px-4 py-3 text-base font-medium text-white transition-colors hover:bg-pitch-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
          >
            {pending ? 'Restoring' : 'Restore'}
          </button>
        )}

        {(turf.status === 'DRAFT' || turf.status === 'REJECTED') && (
          <p className="rounded-lg bg-neutral-100 px-4 py-3 text-sm text-neutral-600">
            This turf is not currently submitted for review. No moderation action applies.
          </p>
        )}
      </div>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-4 px-4 py-3">
      <dt className="text-neutral-500">{label}</dt>
      <dd className="text-right font-medium text-neutral-900">{value}</dd>
    </div>
  )
}
