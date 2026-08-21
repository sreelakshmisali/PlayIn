import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { useCallback, useEffect, useState } from 'react'
import { StyleSheet, Text, View } from 'react-native'

import { Button, EmptyState, ErrorBanner, LoadingView, Screen } from '../../components'
import { fetchMyPlayerProfile } from '../../services/players'
import { ApiError } from '../../services/api'
import { spacing, theme, typography } from '../../theme'
import type { PlayerProfile } from '../../types/players'

type Props = BottomTabScreenProps<Record<string, object | undefined>>

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; profile: PlayerProfile }
  | { kind: 'no-profile' }
  | { kind: 'failed'; message: string }

/** The signed-in player's own sports profile. */
export function PlayerProfileScreen({ navigation }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })

  const load = useCallback(() => {
    setState({ kind: 'loading' })
    fetchMyPlayerProfile()
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
          message="You haven't set up your player profile yet."
          actionLabel="Create your profile"
          onAction={() => navigation.navigate('PlayerProfileEdit')}
        />
      </Screen>
    )
  }

  if (state.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner message={state.message} />
      </Screen>
    )
  }

  const { profile } = state

  return (
    <Screen>
      <View style={styles.header}>
        <Text style={styles.name}>{profile.display_name}</Text>
        {profile.location ? <Text style={styles.location}>{profile.location}</Text> : null}
      </View>

      {profile.bio ? <Text style={styles.bio}>{profile.bio}</Text> : null}

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Sports</Text>
        {profile.sports.length === 0 ? (
          <Text style={styles.mutedText}>No preferred sports yet.</Text>
        ) : (
          <View style={styles.pillRow}>
            {profile.sports.map((ps) => (
              <View key={ps.sport.id} style={styles.pill}>
                <Text style={styles.pillText}>
                  {ps.sport.name}
                  {ps.position ? ` · ${ps.position}` : ''}
                </Text>
              </View>
            ))}
          </View>
        )}
      </View>

      <View style={styles.editAction}>
        <Button label="Edit profile" variant="secondary" onPress={() => navigation.navigate('PlayerProfileEdit')} />
      </View>
    </Screen>
  )
}

const styles = StyleSheet.create({
  header: { alignItems: 'center' },
  name: { ...typography.title, color: theme.textPrimary },
  location: { ...typography.body, color: theme.textSecondary, marginTop: spacing.xs },
  bio: { ...typography.body, color: theme.textPrimary, marginTop: spacing.lg },
  section: { marginTop: spacing.xl },
  sectionTitle: { ...typography.label, color: theme.textPrimary, marginBottom: spacing.sm },
  mutedText: { ...typography.body, color: theme.textMuted },
  pillRow: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.sm },
  pill: { backgroundColor: theme.primarySurface, borderRadius: 999, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  pillText: { ...typography.caption, color: theme.primaryText },
  editAction: { marginTop: spacing.xxl },
})
