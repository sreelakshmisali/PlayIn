import { Ionicons } from '@expo/vector-icons'
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { useCallback, useEffect, useState } from 'react'
import { StyleSheet, Text, View } from 'react-native'

import { Button, EmptyState, ErrorBanner, IconContainer, LoadingView, Screen } from '../../components'
import { fetchMyOwnerProfile } from '../../services/owners'
import { ApiError } from '../../services/api'
import { iconSizes, spacing, theme, typography } from '../../theme'
import type { OwnerProfile } from '../../types/owners'

type Props = BottomTabScreenProps<Record<string, object | undefined>>

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; profile: OwnerProfile }
  | { kind: 'no-profile' }
  | { kind: 'failed'; message: string }

/** The signed-in owner's business profile. */
export function OwnerProfileScreen({ navigation }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const load = useCallback(() => {
    setState({ kind: 'loading' })
    fetchMyOwnerProfile()
      .then((profile) => setState({ kind: 'ready', profile }))
      .catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 404) {
          setState({ kind: 'no-profile' })
          return
        }
        setState({ kind: 'failed', message: error instanceof ApiError ? error.message : 'Could not load your profile.' })
      })
  }, [])

  useEffect(() => {
    const unsubscribe = navigation.addListener('focus', load)
    return unsubscribe
  }, [navigation, load])

  if (state.kind === 'loading') {
    return <LoadingView message="Loading your profile" />
  }

  if (state.kind === 'no-profile') {
    return (
      <Screen scroll={false}>
        <EmptyState
          icon={
            <IconContainer tone="muted" size="lg">
              <Ionicons name="person-outline" size={iconSizes.lg} color={theme.textMuted} />
            </IconContainer>
          }
          title="No profile yet"
          message="Set up your owner profile before listing a turf."
          actionLabel="Create your profile"
          onAction={() => navigation.navigate('OwnerProfileEdit')}
        />
      </Screen>
    )
  }

  if (state.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner message={state.message} onRetry={load} />
      </Screen>
    )
  }

  const { profile } = state

  return (
    <Screen>
      <View style={styles.header}>
        <Text style={styles.name}>{profile.display_name}</Text>
        {profile.phone ? <Text style={styles.meta}>{profile.phone}</Text> : null}
      </View>

      {profile.description ? <Text style={styles.description}>{profile.description}</Text> : null}

      <View style={styles.editAction}>
        <Button label="Edit profile" variant="secondary" onPress={() => navigation.navigate('OwnerProfileEdit')} />
      </View>
    </Screen>
  )
}

const styles = StyleSheet.create({
  header: { alignItems: 'center' },
  name: { ...typography.title, color: theme.textPrimary },
  meta: { ...typography.body, color: theme.textSecondary, marginTop: spacing.xs },
  description: { ...typography.body, color: theme.textPrimary, marginTop: spacing.lg },
  editAction: { marginTop: spacing.xxl },
})
