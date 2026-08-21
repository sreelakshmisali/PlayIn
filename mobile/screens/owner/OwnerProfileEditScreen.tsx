import { useEffect, useState } from 'react'
import { View } from 'react-native'

import { Button, ErrorBanner, LoadingView, Screen, TextField } from '../../components'
import { fetchMyOwnerProfile, saveMyOwnerProfile } from '../../services/owners'
import { ApiError } from '../../services/api'
import { spacing } from '../../theme'

interface Props {
  navigation: { goBack: () => void }
}

/** Create-or-replace form for the signed-in owner's business profile. */
export function OwnerProfileEditScreen({ navigation }: Props) {
  const [loading, setLoading] = useState(true)
  const [displayName, setDisplayName] = useState('')
  const [phone, setPhone] = useState('')
  const [description, setDescription] = useState('')

  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    fetchMyOwnerProfile()
      .then((profile) => {
        setDisplayName(profile.display_name)
        setPhone(profile.phone ?? '')
        setDescription(profile.description ?? '')
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
      await saveMyOwnerProfile({ display_name: displayName, phone, description })
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
        label="Business name"
        value={displayName}
        onChangeText={setDisplayName}
        error={fieldErrors.display_name}
        placeholder="e.g. Kochi Sports Arena"
      />
      <TextField
        label="Phone"
        value={phone}
        onChangeText={setPhone}
        error={fieldErrors.phone}
        placeholder="+91 98765 43210"
        keyboardType="phone-pad"
      />
      <TextField
        label="Description"
        value={description}
        onChangeText={setDescription}
        error={fieldErrors.description}
        placeholder="What makes your turfs worth booking"
        multiline
        numberOfLines={4}
      />

      <View style={{ marginTop: spacing.sm }}>
        <Button label="Save profile" onPress={() => void handleSubmit()} pending={pending} />
      </View>
    </Screen>
  )
}
