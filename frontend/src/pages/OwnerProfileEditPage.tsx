import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

import { Field } from '@/components/form/Field'
import { FormAlert } from '@/components/form/FormAlert'
import { SubmitButton } from '@/components/form/SubmitButton'
import { TextArea } from '@/components/form/TextArea'
import { ApiError } from '@/lib/api'
import { fetchMyOwnerProfile, saveMyOwnerProfile } from '@/lib/owners'

const DESCRIPTION_LIMIT = 1000

/** Create and edit the signed-in owner's business profile. */
export function OwnerProfileEditPage() {
  const navigate = useNavigate()

  const [loading, setLoading] = useState(true)
  const [displayName, setDisplayName] = useState('')
  const [phone, setPhone] = useState('')
  const [description, setDescription] = useState('')

  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    const controller = new AbortController()

    fetchMyOwnerProfile(controller.signal)
      .then((profile) => {
        setDisplayName(profile.display_name)
        setPhone(profile.phone ?? '')
        setDescription(profile.description ?? '')
      })
      .catch((error: unknown) => {
        // A 404 here means first-time setup: the form stays blank. Any other
        // failure is not shown as a blocking error, since the form is still
        // usable to create a profile from scratch.
        if (controller.signal.aborted) return
        if (!(error instanceof ApiError) || error.status !== 404) {
          setMessage('Could not load your existing profile. Starting from a blank form.')
        }
      })
      .finally(() => setLoading(false))

    return () => controller.abort()
  }, [])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setMessage('')
    setFieldErrors({})

    try {
      await saveMyOwnerProfile({ display_name: displayName, phone, description })
      navigate('/owner/profile', { replace: true })
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fieldErrors())
        setMessage(error.message)
      } else {
        setMessage('Something went wrong. Try again.')
      }
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-neutral-500">Loading the editor.</p>
  }

  return (
    <div className="mx-auto w-full max-w-md">
      <header>
        <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
          Your business profile
        </h1>
        <p className="mt-1 text-sm text-neutral-500">This is what players see for your turfs.</p>
      </header>

      <form onSubmit={handleSubmit} noValidate className="mt-6 space-y-5">
        {message && <FormAlert message={message} />}

        <Field
          label="Business name"
          type="text"
          value={displayName}
          onChange={(event) => setDisplayName(event.target.value)}
          required
          placeholder="Kochi Sports Arena"
          error={fieldErrors.display_name}
        />

        <Field
          label="Phone"
          type="tel"
          value={phone}
          onChange={(event) => setPhone(event.target.value)}
          inputMode="tel"
          placeholder="+91 98765 43210"
          hint="Optional."
          error={fieldErrors.phone}
        />

        <TextArea
          label="Description"
          rows={4}
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          maxLength={DESCRIPTION_LIMIT}
          placeholder="Tell players about your turfs."
          hint="Optional."
          error={fieldErrors.description}
        />

        <SubmitButton pending={saving} pendingLabel="Saving">
          Save profile
        </SubmitButton>
      </form>
    </div>
  )
}
