import { useEffect, useState } from 'react'
import { View } from 'react-native'

import { Button, ErrorBanner, LoadingView, Screen, TextField } from '../../components'
import { fetchMyPlayerProfile, saveMyPlayerProfile } from '../../services/players'
import { ApiError } from '../../services/api'
import { spacing } from '../../theme'

interface Props {
  navigation: { goBack: () => void }
}

/** Create-or-replace form for the signed-in player's profile. PUT is a full
 * representation, so the existing profile (if any) is loaded first to avoid
 * silently clearing fields the player did not mean to touch. */
export function PlayerProfileEditScreen({ navigation }: Props) {
  const [loading, setLoading] = useState(true)
  const [displayName, setDisplayName] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [bio, setBio] = useState('')
  const [location, setLocation] = useState('')

  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    fetchMyPlayerProfile()
      .then((profile) => {
        setDisplayName(profile.display_name)
        setImageUrl(profile.image_url ?? '')
        setBio(profile.bio ?? '')
        setLocation(profile.location ?? '')
      })
      .catch(() => {
        // No profile yet: the form just starts blank.
      })
      .finally(() => setLoading(false))
  }, [])

  async function handleSubmit() {
    setPending(true)
    setError('')
    setFieldErrors({})
    try {
      await saveMyPlayerProfile({ display_name: displayName, image_url: imageUrl, bio, location })
      navigation.goBack()
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fieldErrors())
        setError(cause.message)
      } else {
        setError('Something went wrong. Try again.')
      }
    } finally {
      setPending(false)
    }
  }

  if (loading) {
    return <LoadingView message="Loading" />
  }

  return (
    <Screen keyboardSafe>
      {error ? <ErrorBanner message={error} /> : null}

      <TextField
        label="Display name"
        value={displayName}
        onChangeText={setDisplayName}
        error={fieldErrors.display_name}
        placeholder="Your name"
      />
      <TextField
        label="Location"
        value={location}
        onChangeText={setLocation}
        error={fieldErrors.location}
        placeholder="e.g. Kochi"
      />
      <TextField
        label="Bio"
        value={bio}
        onChangeText={setBio}
        error={fieldErrors.bio}
        placeholder="A line about how you play"
        multiline
        numberOfLines={4}
      />
      <TextField
        label="Photo URL"
        value={imageUrl}
        onChangeText={setImageUrl}
        error={fieldErrors.image_url}
        placeholder="https://…"
        autoCapitalize="none"
        keyboardType="url"
      />

      <View style={{ marginTop: spacing.sm }}>
        <Button label="Save profile" onPress={() => void handleSubmit()} pending={pending} />
      </View>
    </Screen>
  )
}
