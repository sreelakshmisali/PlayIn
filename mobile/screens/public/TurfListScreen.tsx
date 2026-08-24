import { Ionicons } from '@expo/vector-icons'
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from 'react-native'

import { EmptyState, ErrorBanner, IconContainer, LoadingView, Text, TurfCard } from '../../components'
import { fetchPublicTurfs } from '../../services/owners'
import { ApiError } from '../../services/api'
import { iconSizes, radius, spacing, theme } from '../../theme'
import type { SportRef, Turf } from '../../types/owners'

// See HomeScreen for why this is typed loosely: the screen is mounted inside
// two different tab param lists that share this route name but nothing else.
type Props = BottomTabScreenProps<Record<string, object | undefined>>

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; turfs: Turf[] }
  | { kind: 'failed'; message: string; isNetworkError: boolean }

type SortKey = 'name' | 'price'

const SORT_LABEL: Record<SortKey, string> = { name: 'Name', price: 'Price' }

/** Every APPROVED turf, browsable without needing an owner profile or a booking. */
export function TurfListScreen({ navigation }: Props) {
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [refreshing, setRefreshing] = useState(false)
  // Filter/sort are purely client-side over the one list already fetched
  // below — fetchPublicTurfs() takes no query parameters, so this changes
  // how the fetched results are presented, never what's requested from the
  // API.
  const [sportFilter, setSportFilter] = useState<string | null>(null)
  const [sortKey, setSortKey] = useState<SortKey>('name')

  const load = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    try {
      const turfs = await fetchPublicTurfs()
      setState({ kind: 'ready', turfs })
    } catch (error) {
      setState({
        kind: 'failed',
        message: error instanceof ApiError ? error.message : 'Could not load turfs.',
        isNetworkError: error instanceof ApiError && error.isNetworkError,
      })
    } finally {
      if (isRefresh) setRefreshing(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const allTurfs = state.kind === 'ready' ? state.turfs : []

  const sportOptions = useMemo<SportRef[]>(
    () => Array.from(new Map(allTurfs.flatMap((turf) => turf.sports).map((sport) => [sport.id, sport])).values()),
    [allTurfs],
  )

  const visibleTurfs = useMemo(() => {
    const filtered = sportFilter ? allTurfs.filter((turf) => turf.sports.some((sport) => sport.id === sportFilter)) : allTurfs

    return [...filtered].sort((a, b) => {
      if (sortKey === 'name') return a.name.localeCompare(b.name)
      return (a.slot_price ?? Number.POSITIVE_INFINITY) - (b.slot_price ?? Number.POSITIVE_INFINITY)
    })
  }, [allTurfs, sportFilter, sortKey])

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
            <Text variant="screenTitle" style={styles.title}>
              Turfs
            </Text>
            <ErrorBanner
              message={state.message}
              onRetry={() => void load()}
              kind={state.isNetworkError ? 'network' : 'generic'}
            />
          </>
        }
      />
    )
  }

  return (
    <FlatList
      data={visibleTurfs}
      keyExtractor={(turf) => turf.id}
      contentContainerStyle={styles.list}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void load(true)} />}
      ListHeaderComponent={
        <View style={styles.header}>
          <Text variant="screenTitle" style={styles.title}>
            Turfs
          </Text>

          {sportOptions.length > 0 && (
            <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.chipRow}>
              <FilterChip label="All" active={sportFilter === null} onPress={() => setSportFilter(null)} />
              {sportOptions.map((sport) => (
                <FilterChip
                  key={sport.id}
                  label={sport.name}
                  active={sportFilter === sport.id}
                  onPress={() => setSportFilter(sport.id)}
                />
              ))}
            </ScrollView>
          )}

          {allTurfs.length > 0 && (
            <View style={styles.sortRow}>
              <Text variant="caption" color="muted">
                {visibleTurfs.length} {visibleTurfs.length === 1 ? 'turf' : 'turfs'}
              </Text>
              <Pressable
                onPress={() => setSortKey((key) => (key === 'name' ? 'price' : 'name'))}
                accessibilityRole="button"
                style={styles.sortButton}
              >
                <Ionicons name="swap-vertical-outline" size={iconSizes.sm} color={theme.textSecondary} />
                <Text variant="caption" color="secondary">{`Sort: ${SORT_LABEL[sortKey]}`}</Text>
              </Pressable>
            </View>
          )}
        </View>
      }
      ItemSeparatorComponent={() => <View style={styles.separator} />}
      ListEmptyComponent={
        <EmptyState
          icon={
            <IconContainer tone="muted" size="lg">
              <Ionicons name="location-outline" size={iconSizes.lg} color={theme.textMuted} />
            </IconContainer>
          }
          title={allTurfs.length === 0 ? 'No turfs yet' : 'No results'}
          message={
            allTurfs.length === 0
              ? 'No turfs are listed yet. Check back soon.'
              : 'No turfs match this sport. Try a different filter.'
          }
        />
      }
      renderItem={({ item }) => (
        <TurfCard turf={item} onPress={() => navigation.navigate('TurfDetail', { turfId: item.id })} />
      )}
    />
  )
}

function FilterChip({ label, active, onPress }: { label: string; active: boolean; onPress: () => void }) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityState={{ selected: active }}
      style={[styles.chip, active && styles.chipActive]}
    >
      <Text variant="bodyEmphasized" color={active ? 'onPrimary' : 'secondary'}>
        {label}
      </Text>
    </Pressable>
  )
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, flexGrow: 1 },
  header: { marginBottom: spacing.lg },
  title: { color: theme.textPrimary },
  chipRow: { gap: spacing.sm, marginTop: spacing.lg },
  chip: {
    borderRadius: radius.pill,
    borderWidth: 1,
    borderColor: theme.border,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    backgroundColor: theme.surface,
  },
  chipActive: { backgroundColor: theme.primary, borderColor: theme.primary },
  sortRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: spacing.md,
  },
  sortButton: { flexDirection: 'row', alignItems: 'center', gap: spacing.xs },
  separator: { height: spacing.md },
})
