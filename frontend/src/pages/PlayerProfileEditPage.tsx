import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { useAuth } from '@/auth/useAuth'
import { Field } from '@/components/form/Field'
import { FormAlert } from '@/components/form/FormAlert'
import { SubmitButton } from '@/components/form/SubmitButton'
import { TextArea } from '@/components/form/TextArea'
import { SportPicker } from '@/components/players/SportPicker'
import { ApiError } from '@/lib/api'
import {
  fetchMyProfile,
  fetchSports,
  removeMySport,
  saveMyProfile,
  setMySport,
  type PlayerProfile,
  type PlayerSport,
  type Sport,
} from '@/lib/players'

const BIO_LIMIT = 500

/**
 * Create and edit the signed-in player's profile.
 *
 * The details form and the sports list save through different endpoints, and
 * the screen reflects that rather than hiding it: the form is submitted, while
 * each sport is written the moment it is toggled. Batching the sports behind
 * the same button would mean holding a client-side diff that the API has no
 * endpoint to accept.
 */
export function PlayerProfileEditPage() {
  const { user } = useAuth()
  const navigate = useNavigate()

  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [displayName, setDisplayName] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [bio, setBio] = useState('')
  const [location, setLocation] = useState('')

  const [sports, setSports] = useState<Sport[]>([])
  const [chosen, setChosen] = useState<PlayerSport[]>([])
  const [hasProfile, setHasProfile] = useState(false)

  const [pendingSport, setPendingSport] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  // The catalogue and the current profile load together. A 404 on the profile
  // means this is a first-time setup, which is a state, not a failure.
  useEffect(() => {
    const controller = new AbortController()

    Promise.all([
      fetchSports(controller.signal),
      fetchMyProfile(controller.signal).catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 404) return null
        throw error
      }),
    ])
      .then(([catalogue, profile]) => {
        setSports(catalogue)

        if (profile) {
          applyProfile(profile)
          setHasProfile(true)
        } else if (user) {
          // A sensible starting point rather than an empty box.
          setDisplayName(user.full_name)
        }
        setLoading(false)
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        setLoadError(error instanceof ApiError ? error.message : 'Could not load the editor.')
        setLoading(false)
      })

    return () => controller.abort()

    function applyProfile(profile: PlayerProfile) {
      setDisplayName(profile.display_name)
      setImageUrl(profile.image_url ?? '')
      setBio(profile.bio ?? '')
      setLocation(profile.location ?? '')
      setChosen(profile.sports)
    }
  }, [user])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setMessage('')
    setFieldErrors({})

    try {
      await saveMyProfile({
        display_name: displayName,
        image_url: imageUrl,
        bio,
        location,
      })
      navigate('/profile', { replace: true })
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

  // Sports can only be written once a profile row exists, because they hang off
  // it. Saying so is better than a 404 the player cannot interpret.
  async function writeSport(sportId: string, action: () => Promise<PlayerProfile>) {
    if (!hasProfile) {
      setMessage('Save your details first, then choose your sports.')
      return
    }

    setPendingSport(sportId)
    setMessage('')

    try {
      setChosen((await action()).sports)
    } catch (error) {
      setMessage(
        error instanceof ApiError ? error.message : 'Could not update your sports. Try again.',
      )
    } finally {
      setPendingSport(null)
    }
  }

  if (loading) {
    return <p className="text-neutral-500">Loading the editor.</p>
  }

  if (loadError) {
    return (
      <div className="mx-auto w-full max-w-md">
        <FormAlert message={loadError} />
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-md">
      <header>
        <h1 className="text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
          {hasProfile ? 'Edit your profile' : 'Create your profile'}
        </h1>
        <p className="mt-1 text-sm text-neutral-500">
          This is what other players see.
        </p>
      </header>

      <form onSubmit={handleSubmit} noValidate className="mt-6 space-y-5">
        {message && <FormAlert message={message} />}

        <Field
          label="Display name"
          type="text"
          name="display_name"
          value={displayName}
          onChange={(event) => setDisplayName(event.target.value)}
          required
          placeholder="How you want to be known"
          error={fieldErrors.display_name}
        />

        <Field
          label="Location"
          type="text"
          name="location"
          value={location}
          onChange={(event) => setLocation(event.target.value)}
          placeholder="City or area"
          hint="Optional."
          error={fieldErrors.location}
        />

        <Field
          label="Profile image URL"
          type="url"
          name="image_url"
          value={imageUrl}
          onChange={(event) => setImageUrl(event.target.value)}
          inputMode="url"
          autoCapitalize="none"
          spellCheck={false}
          placeholder="https://example.com/photo.jpg"
          hint="Optional. Must be a full http or https address."
          error={fieldErrors.image_url}
        />

        <TextArea
          label="Bio"
          name="bio"
          rows={4}
          value={bio}
          onChange={(event) => setBio(event.target.value)}
          maxLength={BIO_LIMIT}
          placeholder="How you play, when you play, who you play with."
          hint="Optional."
          error={fieldErrors.bio}
        />

        <SubmitButton pending={saving} pendingLabel="Saving">
          {hasProfile ? 'Save details' : 'Create profile'}
        </SubmitButton>
      </form>

      <section className="mt-10">
        <h2 className="text-sm font-medium text-neutral-800">Preferred sports</h2>
        <p className="mt-1 text-xs text-neutral-500">
          {hasProfile
            ? 'Each change saves on its own.'
            : 'Available once your profile is created.'}
        </p>

        <div className={`mt-3 ${hasProfile ? '' : 'pointer-events-none opacity-50'}`}>
          <SportPicker
            sports={sports}
            chosen={chosen}
            pending={pendingSport}
            onToggle={(sport, selected) =>
              void writeSport(sport.id, () =>
                selected ? setMySport(sport.id) : removeMySport(sport.id),
              )
            }
            onPositionChange={(sport, position) =>
              void writeSport(sport.id, () => setMySport(sport.id, position))
            }
          />
        </div>
      </section>

      <Link
        to="/profile"
        className="mt-8 block rounded-lg border border-neutral-300 bg-white px-4 py-3 text-center text-base font-medium text-neutral-800 transition-colors hover:bg-neutral-50"
      >
        Done
      </Link>
    </div>
  )
}
