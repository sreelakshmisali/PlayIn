import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { useCallback, useEffect, useState } from 'react'
import { FlatList, RefreshControl, StyleSheet, Text, View } from 'react-native'

import { Button, EmptyState, ErrorBanner, LoadingView, TurfCard } from '../../components'
import { fetchMyTurfs } from '../../services/owners'
import { ApiError } from '../../services/api'
import { spacing, theme, typography } from '../../theme'
import type { Turf } from '../../types/owners'

type Props = BottomTabScreenProps<Record<string, object | undefined>>

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; turfs: Turf[] }
  | { kind: 'no-profile' }
  | { kind: 'failed'; message: string }

/** The signed-in owner's own turfs, every status included. */
export function OwnerTurfListScreen({ navigation }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [refreshing, setRefreshing] = useState(false)

  const load = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    else setState({ kind: 'loading' })

    try {
      const turfs = await fetchMyTurfs()
      setState({ kind: 'ready', turfs })
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        setState({ kind: 'no-profile' })
        return
      }
      setState({ kind: 'failed', message: error instanceof ApiError ? error.message : 'Could not load your turfs.' })
    } finally {
      if (isRefresh) setRefreshing(false)
    }
  }, [])

  useEffect(() => {
    const unsubscribe = navigation.addListener('focus', () => void load())
    return unsubscribe
  }, [navigation, load])

  const header = (
    <View style={styles.header}>
      <Text style={styles.title}>Your turfs</Text>
      <Button label="New turf" onPress={() => navigation.navigate('OwnerTurfEdit', {})} />
    </View>
  )

  if (state.kind === 'loading') {
    return <LoadingView message="Loading your turfs" />
  }

  if (state.kind === 'no-profile') {
    return (
      <FlatList
        data={[]}
        renderItem={null}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          <>
            <Text style={styles.title}>Your turfs</Text>
            <EmptyState
              message="Set up your owner profile before listing a turf."
              actionLabel="Create your profile"
              onAction={() => navigation.navigate('Profile')}
            />
          </>
        }
      />
    )
  }

  if (state.kind === 'failed') {
    return (
      <FlatList
        data={[]}
        renderItem={null}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          <>
            <Text style={styles.title}>Your turfs</Text>
            <ErrorBanner message={state.message} />
          </>
        }
      />
    )
  }

  return (
    <FlatList
      data={state.turfs}
      keyExtractor={(turf) => turf.id}
      contentContainerStyle={styles.list}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void load(true)} />}
      ListHeaderComponent={header}
      ItemSeparatorComponent={() => <View style={styles.separator} />}
      ListEmptyComponent={
        <EmptyState
          message="You have not listed a turf yet."
          actionLabel="List your first turf"
          onAction={() => navigation.navigate('OwnerTurfEdit', {})}
        />
      }
      renderItem={({ item }) => (
        <TurfCard
          turf={item}
          showStatus
          onPress={() => navigation.navigate('OwnerTurfEdit', { turfId: item.id })}
        />
      )}
    />
  )
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, flexGrow: 1 },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: spacing.lg, gap: spacing.md },
  title: { ...typography.title, color: theme.textPrimary },
  separator: { height: spacing.md },
})
