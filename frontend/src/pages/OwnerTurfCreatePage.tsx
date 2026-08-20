import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { FormAlert } from '@/components/form/FormAlert'
import { emptyTurfFormValues, TurfDetailsForm } from '@/components/owners/TurfDetailsForm'
import { ApiError } from '@/lib/api'
import { createTurf, type SaveTurfPayload } from '@/lib/owners'

/**
 * Creates a turf, then hands off to the edit screen where sports, amenities
 * and images are managed: those sub-resources need a turf id to attach to,
 * which does not exist until this step completes.
 */
export function OwnerTurfCreatePage() {
  const navigate = useNavigate()

  const [pending, setPending] = useState(false)
  const [message, setMessage] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  async function handleSubmit(payload: SaveTurfPayload) {
    setPending(true)
    setMessage('')
    setFieldErrors({})

    try {
      const turf = await createTurf(payload)
      navigate(`/owner/turfs/${turf.id}/edit`, { replace: true })
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fieldErrors())
        setMessage(error.message)
      } else {
        setMessage('Something went wrong. Try again.')
      }
      setPending(false)
    }
  }

  return (
    <div className="mx-auto w-full max-w-md">
      <header>
        <p className="text-sm">
          <Link to="/owner/turfs" className="text-pitch-700 underline">
            Your turfs
          </Link>
        </p>
        <h1 className="mt-1 text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
          List a new turf
        </h1>
        <p className="mt-1 text-sm text-neutral-500">
          It starts as a draft. Add sports, amenities and photos next, then submit it for review.
        </p>
      </header>

      <div className="mt-6">
        {message && <div className="mb-5"><FormAlert message={message} /></div>}

        <TurfDetailsForm
          initial={emptyTurfFormValues}
          pending={pending}
          submitLabel="Create turf"
          pendingLabel="Creating"
          fieldErrors={fieldErrors}
          onSubmit={handleSubmit}
        />
      </div>
    </div>
  )
}
