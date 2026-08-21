import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { useCallback, useEffect, useState } from 'react'
import { FlatList, RefreshControl, StyleSheet, Text } from 'react-native'

import { EmptyState, ErrorBanner, LoadingView, TurfCard } from '../../components'
import { fetchPublicTurfs } from '../../services/owners'
import { ApiError } from '../../services/api'
import { spacing, theme, typography } from '../../theme'
import type { Turf } from '../../types/owners'

// See HomeScreen for why this is typed loosely: the screen is mounted inside
// two different tab param lists that share this route name but nothing else.
type Props = BottomTabScreenProps<Record<string, object | undefined>>

type State = { kind: 'loading' } | { kind: 'ready'; turfs: Turf[] } | { kind: 'failed'; message: string }

/** Every APPROVED turf, browsable without needing an owner profile or a booking. */
export function TurfListScreen({ navigation }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [refreshing, setRefreshing] = useState(false)

  const load = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    try {
      const turfs = await fetchPublicTurfs()
      setState({ kind: 'ready', turfs })
    } catch (error) {
      setState({
        kind: 'failed',
        message: error instanceof ApiError ? error.message : 'Could not load turfs.',
      })
    } finally {
      if (isRefresh) setRefreshing(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  if (state.kind === 'loading') {
    return <LoadingView message="Loading turfs" />
  }

  if (state.kind === 'failed') {
    return (
      <FlatList
        data={[]}
        renderItem={null}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          <>
            <Text style={styles.title}>Turfs</Text>
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
      ListHeaderComponent={<Text style={styles.title}>Turfs</Text>}
      ItemSeparatorComponent={() => <Text style={styles.separator} />}
      ListEmptyComponent={<EmptyState message="No turfs are listed yet. Check back soon." />}
      renderItem={({ item }) => (
        <TurfCard turf={item} onPress={() => navigation.navigate('TurfDetail', { turfId: item.id })} />
      )}
    />
  )
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, flexGrow: 1 },
  title: { ...typography.title, color: theme.textPrimary, marginBottom: spacing.lg },
  separator: { height: spacing.md },
})
