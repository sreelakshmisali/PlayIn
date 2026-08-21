import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { FormAlert } from '@/components/form/FormAlert'
import { CheckboxList } from '@/components/owners/CheckboxList'
import { TurfDetailsForm, turfToFormValues } from '@/components/owners/TurfDetailsForm'
import { TurfImageManager } from '@/components/owners/TurfImageManager'
import { TurfStatusBadge } from '@/components/owners/TurfStatusBadge'
import { ApiError } from '@/lib/api'
import {
  addTurfAmenity,
  addTurfImage,
  addTurfSport,
  deleteTurf,
  fetchAmenities,
  fetchMyTurf,
  removeTurfAmenity,
  removeTurfImage,
  removeTurfSport,
  submitTurf,
  updateTurf,
  type Amenity,
  type SaveTurfPayload,
  type Turf,
} from '@/lib/owners'
import { fetchSports, type Sport } from '@/lib/players'

type LoadState = { kind: 'loading' } | { kind: 'ready' } | { kind: 'failed'; message: string }

const SUBMITTABLE = new Set(['DRAFT', 'REJECTED'])

/**
 * Edits an existing turf: its own details, the sports and amenities it
 * offers, its photos, and its place in the approval workflow.
 *
 * Details save through PUT as one form. Sports, amenities and images each
 * write on their own the moment they change, matching the shape of the API:
 * none of those three has an endpoint that replaces a whole set.
 */
export function OwnerTurfEditPage() {
  const { turfId = '' } = useParams()
  const navigate = useNavigate()

  const [load, setLoad] = useState<LoadState>({ kind: 'loading' })
  const [turf, setTurf] = useState<Turf | null>(null)
  const [sports, setSports] = useState<Sport[]>([])
  const [amenities, setAmenities] = useState<Amenity[]>([])

  const [savingDetails, setSavingDetails] = useState(false)
  const [detailsMessage, setDetailsMessage] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const [submitting, setSubmitting] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [actionMessage, setActionMessage] = useState('')

  const [pendingSport, setPendingSport] = useState<string | null>(null)
  const [pendingAmenity, setPendingAmenity] = useState<string | null>(null)
  const [pendingImage, setPendingImage] = useState(false)
  const [removingImageId, setRemovingImageId] = useState<string | null>(null)
  const [imageError, setImageError] = useState('')

  useEffect(() => {
    const controller = new AbortController()

    Promise.all([
      fetchMyTurf(turfId, controller.signal),
      fetchSports(controller.signal),
      fetchAmenities(controller.signal),
    ])
      .then(([loadedTurf, sportCatalogue, amenityCatalogue]) => {
        setTurf(loadedTurf)
        setSports(sportCatalogue)
        setAmenities(amenityCatalogue)
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

  async function handleSaveDetails(payload: SaveTurfPayload) {
    setSavingDetails(true)
    setDetailsMessage('')
    setFieldErrors({})

    try {
      setTurf(await updateTurf(turfId, payload))
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fieldErrors())
        setDetailsMessage(error.message)
      } else {
        setDetailsMessage('Something went wrong. Try again.')
      }
    } finally {
      setSavingDetails(false)
    }
  }

  async function handleSubmitForReview() {
    setSubmitting(true)
    setActionMessage('')
    try {
      setTurf(await submitTurf(turfId))
    } catch (error) {
      setActionMessage(error instanceof ApiError ? error.message : 'Could not submit this turf.')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete() {
    setDeleting(true)
    setActionMessage('')
    try {
      await deleteTurf(turfId)
      navigate('/owner/turfs', { replace: true })
    } catch (error) {
      setActionMessage(error instanceof ApiError ? error.message : 'Could not delete this turf.')
      setDeleting(false)
    }
  }

  async function toggleSport(sport: { id: string }, selected: boolean) {
    setPendingSport(sport.id)
    try {
      setTurf(selected ? await addTurfSport(turfId, sport.id) : await removeTurfSport(turfId, sport.id))
    } catch (error) {
      setActionMessage(error instanceof ApiError ? error.message : 'Could not update sports.')
    } finally {
      setPendingSport(null)
    }
  }

  async function toggleAmenity(amenity: { id: string }, selected: boolean) {
    setPendingAmenity(amenity.id)
    try {
      setTurf(selected ? await addTurfAmenity(turfId, amenity.id) : await removeTurfAmenity(turfId, amenity.id))
    } catch (error) {
      setActionMessage(error instanceof ApiError ? error.message : 'Could not update amenities.')
    } finally {
      setPendingAmenity(null)
    }
  }

  async function handleAddImage(url: string) {
    setPendingImage(true)
    setImageError('')
    try {
      setTurf(await addTurfImage(turfId, url))
    } catch (error) {
      setImageError(error instanceof ApiError ? error.message : 'Could not add that image.')
    } finally {
      setPendingImage(false)
    }
  }

  async function handleRemoveImage(imageId: string) {
    setRemovingImageId(imageId)
    setImageError('')
    try {
      setTurf(await removeTurfImage(turfId, imageId))
    } catch (error) {
      setImageError(error instanceof ApiError ? error.message : 'Could not remove that image.')
    } finally {
      setRemovingImageId(null)
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
        <Link to="/owner/turfs" className="mt-4 inline-block text-sm font-medium text-pitch-700 underline">
          Back to your turfs
        </Link>
      </div>
    )
  }

  const selectedSportIds = new Set(turf.sports.map((s) => s.id))
  const selectedAmenityIds = new Set(turf.amenities.map((a) => a.id))

  return (
    <div className="mx-auto w-full max-w-md">
      <p className="text-sm">
        <Link to="/owner/turfs" className="text-pitch-700 underline">
          Your turfs
        </Link>
      </p>

      <header className="mt-1 flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
          {turf.name}
        </h1>
        <TurfStatusBadge status={turf.status} />
      </header>

      <Link
        to={`/owner/turfs/${turf.id}/availability`}
        className="mt-4 flex items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-white p-4 transition-colors hover:bg-neutral-50"
      >
        <div>
          <p className="font-medium text-neutral-900">Availability</p>
          <p className="mt-0.5 text-sm text-neutral-500">Slot duration, pricing, generated slots and blocks.</p>
        </div>
        <span aria-hidden="true" className="text-neutral-400">
          →
        </span>
      </Link>

      {actionMessage && (
        <div className="mt-4">
          <FormAlert message={actionMessage} />
        </div>
      )}

      {SUBMITTABLE.has(turf.status) && (
        <button
          type="button"
          onClick={() => void handleSubmitForReview()}
          disabled={submitting}
          className="mt-4 w-full rounded-lg bg-pitch-600 px-4 py-3 text-base font-medium text-white transition-colors hover:bg-pitch-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
        >
          {submitting ? 'Submitting' : 'Submit for review'}
        </button>
      )}
      {turf.status === 'PENDING_APPROVAL' && (
        <p className="mt-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-800">
          This turf is waiting for admin review. It is not visible to players yet.
        </p>
      )}
      {turf.status === 'APPROVED' && (
        <p className="mt-4 rounded-lg bg-pitch-50 px-4 py-3 text-sm text-pitch-700">
          This turf is live. Editing its details sends it back for review.
        </p>
      )}
      {turf.status === 'SUSPENDED' && (
        <p className="mt-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-800">
          This turf has been suspended and is not visible to players.
        </p>
      )}

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Details</h2>
        <div className="mt-3">
          {detailsMessage && (
            <div className="mb-5">
              <FormAlert message={detailsMessage} />
            </div>
          )}
          <TurfDetailsForm
            initial={turfToFormValues(turf)}
            pending={savingDetails}
            submitLabel="Save details"
            pendingLabel="Saving"
            fieldErrors={fieldErrors}
            onSubmit={handleSaveDetails}
          />
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Sports</h2>
        <div className="mt-3">
          <CheckboxList
            options={sports}
            selectedIds={selectedSportIds}
            pendingId={pendingSport}
            onToggle={(option, selected) => void toggleSport(option, selected)}
          />
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Amenities</h2>
        <div className="mt-3">
          <CheckboxList
            options={amenities}
            selectedIds={selectedAmenityIds}
            pendingId={pendingAmenity}
            onToggle={(option, selected) => void toggleAmenity(option, selected)}
          />
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-neutral-800">Photos</h2>
        <p className="mt-1 text-xs text-neutral-500">Paste a link to an image. Up to 12 photos.</p>
        <div className="mt-3">
          <TurfImageManager
            images={turf.images}
            pending={pendingImage}
            removingId={removingImageId}
            error={imageError}
            onAdd={(url) => void handleAddImage(url)}
            onRemove={(id) => void handleRemoveImage(id)}
          />
        </div>
      </section>

      <button
        type="button"
        onClick={() => void handleDelete()}
        disabled={deleting}
        className="mt-10 w-full rounded-lg border border-red-200 bg-white px-4 py-3 text-base font-medium text-red-700 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:text-neutral-400"
      >
        {deleting ? 'Deleting' : 'Delete turf'}
      </button>
    </div>
  )
}
