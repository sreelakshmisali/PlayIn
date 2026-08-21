import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { AvailableSlotsList } from '@/components/availability/AvailableSlotsList'
import { DayStrip } from '@/components/availability/DayStrip'
import { ApiError } from '@/lib/api'
import { fetchPublicAvailability, type Slot } from '@/lib/availability'
import { today } from '@/lib/date'
import { fetchPublicTurf, type Turf } from '@/lib/owners'

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; turf: Turf }
  | { kind: 'missing' }
  | { kind: 'failed'; message: string }

/**
 * Public detail view for one turf. Reachable only for an APPROVED turf: the
 * API answers 404 for any other status, indistinguishable from an id that
 * does not exist at all, so this page never has to know the difference.
 */
export function PublicTurfDetailPage() {
  const { turfId = '' } = useParams()
  const [state, setState] = useState<State>({ kind: 'loading' })

  const [selectedDate, setSelectedDate] = useState(today())
  const [slots, setSlots] = useState<Slot[]>([])
  const [slotsState, setSlotsState] = useState<'loading' | 'ready' | 'failed'>('loading')
  const [slotsErrorMessage, setSlotsErrorMessage] = useState('')

  useEffect(() => {
    const controller = new AbortController()
    setState({ kind: 'loading' })

    fetchPublicTurf(turfId, controller.signal)
      .then((turf) => setState({ kind: 'ready', turf }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return

        if (error instanceof ApiError && error.status === 404) {
          setState({ kind: 'missing' })
          return
        }
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load this turf.',
        })
      })

    return () => controller.abort()
  }, [turfId])

  useEffect(() => {
    if (state.kind !== 'ready') return

    let cancelled = false
    setSlotsState('loading')

    fetchPublicAvailability(turfId, selectedDate)
      .then((fetched) => {
        if (cancelled) return
        setSlots(fetched)
        setSlotsState('ready')
      })
      .catch((error: unknown) => {
        if (cancelled) return
        setSlotsErrorMessage(error instanceof ApiError ? error.message : 'Could not load slots for this date.')
        setSlotsState('failed')
      })

    return () => {
      cancelled = true
    }
  }, [state.kind, turfId, selectedDate])

  if (state.kind === 'loading') {
    return <p className="text-neutral-500">Loading turf.</p>
  }

  if (state.kind === 'missing') {
    return (
      <section className="mx-auto w-full max-w-md text-center">
        <p className="text-sm font-medium text-pitch-600">404</p>
        <h1 className="mt-2 text-xl font-semibold tracking-tight text-neutral-900">
          No turf here
        </h1>
        <p className="mt-3 text-neutral-600">
          This turf does not exist, or is not open for booking yet.
        </p>
        <Link to="/turfs" className="mt-6 inline-block text-sm font-medium text-pitch-700 underline">
          Browse turfs
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

  const { turf } = state

  return (
    <article className="mx-auto w-full max-w-md">
      {turf.images.length > 0 && (
        <div className="-mx-4 mb-5 flex snap-x gap-2 overflow-x-auto px-4 sm:mx-0 sm:px-0">
          {turf.images.map((image) => (
            <img
              key={image.id}
              src={image.image_url}
              alt=""
              className="aspect-video w-64 shrink-0 snap-start rounded-lg border border-neutral-200 object-cover"
            />
          ))}
        </div>
      )}

      <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
        {turf.name}
      </h1>
      <p className="mt-1 text-sm text-neutral-500">
        {turf.address}, {turf.city}
      </p>
      <p className="mt-1 text-sm text-neutral-500">Listed by {turf.owner_display_name}</p>

      {turf.description && (
        <p className="mt-4 whitespace-pre-line text-neutral-700">{turf.description}</p>
      )}

      <dl className="mt-6 divide-y divide-neutral-100 overflow-hidden rounded-xl border border-neutral-200 bg-white text-sm">
        <Row label="Hours" value={`${turf.opening_time} – ${turf.closing_time}`} />
        {turf.capacity !== undefined && <Row label="Capacity" value={`${turf.capacity} players`} />}
        {turf.latitude !== undefined && turf.longitude !== undefined && (
          <Row label="Coordinates" value={`${turf.latitude}, ${turf.longitude}`} />
        )}
      </dl>

      <section className="mt-6">
        <h2 className="text-sm font-medium text-neutral-800">Availability</h2>
        <div className="mt-2">
          <DayStrip value={selectedDate} onChange={setSelectedDate} />
          <AvailableSlotsList state={slotsState} slots={slots} errorMessage={slotsErrorMessage} />
        </div>
      </section>

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

      {turf.amenities.length > 0 && (
        <section className="mt-6">
          <h2 className="text-sm font-medium text-neutral-800">Amenities</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {turf.amenities.map((amenity) => (
              <span
                key={amenity.id}
                className="rounded-full bg-pitch-50 px-3 py-1 text-sm text-pitch-700"
              >
                {amenity.name}
              </span>
            ))}
          </div>
        </section>
      )}
    </article>
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
