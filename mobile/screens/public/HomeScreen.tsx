import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { Ionicons } from '@expo/vector-icons'
import { useCallback, useEffect, useState } from 'react'
import { Pressable, ScrollView, StyleSheet, View } from 'react-native'

import {
  Button,
  Divider,
  EmptyState,
  ErrorBanner,
  IconContainer,
  LoadingView,
  Screen,
  Text,
  TurfCard,
} from '../../components'
import { useAuth } from '../../hooks'
import { fetchPublicTurfs } from '../../services/owners'
import { ApiError } from '../../services/api'
import { iconSizes, radius, spacing, theme } from '../../theme'
import type { Turf } from '../../types/owners'

// HomeScreen is mounted inside both PlayerTabParamList and OwnerTabParamList,
// which share the same three sibling route names ('Turfs', 'Profile',
// 'Account') but are otherwise distinct param lists. Typing this against
// either one by name would be misleading, so the navigation prop is accepted
// loosely rather than pinned to one role's tab list. 'TurfDetail' lives one
// level up, on the enclosing stack navigator — omitted from every tab list,
// so navigating to it bubbles up automatically, the same pattern
// TurfListScreen already relies on.
type Props = BottomTabScreenProps<Record<string, object | undefined>>

type TurfsState = { kind: 'loading' } | { kind: 'ready'; turfs: Turf[] } | { kind: 'failed'; message: string }

/** How many turfs to show in the home preview before "See all" takes over. */
const PREVIEW_COUNT = 3
/** How many distinct sports to surface as quick-browse chips. */
const MAX_SPORT_CHIPS = 6

/**
 * The landing tab. Leads with the one thing everyone is here for — finding
 * and booking a turf — then a short real preview of what's available, a
 * couple of quick ways into that same list, and, last, where booking
 * history will live once it exists.
 */
export function HomeScreen({ navigation }: Props) {
  const { user } = useAuth()
  const [state, setState] = useState<TurfsState>({ kind: 'loading' })

  // Same fetchPublicTurfs() call TurfListScreen already makes — no new
  // endpoint, no new business rule. Home just also reads from it, to show a
  // real preview instead of a static promise of one.
  const load = useCallback(async () => {
    try {
      const turfs = await fetchPublicTurfs()
      setState({ kind: 'ready', turfs })
    } catch (error) {
      setState({
        kind: 'failed',
        message: error instanceof ApiError ? error.message : 'Could not load turfs.',
      })
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const goToTurfs = () => navigation.navigate('Turfs')

  const sportChips =
    state.kind === 'ready'
      ? Array.from(new Map(state.turfs.flatMap((turf) => turf.sports).map((sport) => [sport.id, sport])).values()).slice(
          0,
          MAX_SPORT_CHIPS,
        )
      : []

  return (
    <Screen>
      {/* 1. Where/what to book: the greeting and the one primary CTA. */}
      <View style={styles.hero}>
        <View style={styles.eyebrowRow}>
          <IconContainer tone="primary" size="sm">
            <Ionicons name="football-outline" size={iconSizes.sm} color={theme.primary} />
          </IconContainer>
          <Text variant="label" style={styles.eyebrow}>
            PlayHub
          </Text>
        </View>
        <Text variant="screenTitle" style={styles.title}>
          {user ? `Welcome back, ${user.full_name.split(' ')[0]}` : 'Welcome'}
        </Text>
        <Text variant="body" color="secondary" style={styles.subtitle}>
          Find a turf, check its availability, and get playing.
        </Text>
        <Button label="Find a turf" onPress={goToTurfs} style={styles.cta} />
      </View>

      {/* 3. Quick categories — real sports drawn from the fetched turfs, not a fixed list. */}
      {sportChips.length > 0 && (
        <View style={styles.section}>
          <Text variant="sectionTitle" style={styles.sectionTitle}>
            Browse by sport
          </Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.chipRow}>
            {sportChips.map((sport) => (
              <Pressable
                key={sport.id}
                onPress={goToTurfs}
                accessibilityRole="button"
                style={({ pressed }) => [styles.chip, pressed && styles.chipPressed]}
              >
                <Text variant="bodyEmphasized" color="secondary">
                  {sport.name}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>
      )}

      <Divider />

      {/* 2. Available turfs: a short real preview, not every section as its own card. */}
      <View style={styles.section}>
        <View style={styles.sectionHeader}>
          <Text variant="sectionTitle" style={styles.sectionTitle}>
            Available turfs
          </Text>
          {state.kind === 'ready' && state.turfs.length > 0 && (
            <Pressable onPress={goToTurfs} accessibilityRole="button">
              <Text variant="bodyEmphasized" color="primary">
                See all
              </Text>
            </Pressable>
          )}
        </View>

        {state.kind === 'loading' && <LoadingView size="small" inline message="Loading turfs" />}
        {state.kind === 'failed' && <ErrorBanner message={state.message} onRetry={() => void load()} />}
        {state.kind === 'ready' && state.turfs.length === 0 && (
          <EmptyState
            icon={
              <IconContainer tone="muted" size="lg">
                <Ionicons name="location-outline" size={iconSizes.lg} color={theme.textMuted} />
              </IconContainer>
            }
            message="No turfs are listed yet. Check back soon."
          />
        )}
        {state.kind === 'ready' && state.turfs.length > 0 && (
          <View style={styles.turfList}>
            {state.turfs.slice(0, PREVIEW_COUNT).map((turf) => (
              <TurfCard
                key={turf.id}
                turf={turf}
                onPress={() => navigation.navigate('TurfDetail', { turfId: turf.id })}
              />
            ))}
          </View>
        )}
      </View>

      <Divider />

      {/* 4. Quick access to bookings — no booking history exists in this API yet,
          so this stays an honest placeholder rather than invented data or a
          dead-end tap target. */}
      <View style={styles.section}>
        <View style={styles.bookingsRow}>
          <IconContainer tone="muted">
            <Ionicons name="calendar-outline" size={iconSizes.md} color={theme.textMuted} />
          </IconContainer>
          <View style={styles.bookingsText}>
            <Text variant="bodyEmphasized">Your bookings</Text>
            <Text variant="caption" color="muted">
              Booking history will show up here once it's available.
            </Text>
          </View>
        </View>
      </View>

      {/* 5. Supporting information — a quiet, real count, not marketing copy. */}
      {state.kind === 'ready' && (
        <Text variant="metadata" color="muted" style={styles.footer}>
          {state.turfs.length} {state.turfs.length === 1 ? 'turf is' : 'turfs are'} listed on PlayHub right now.
        </Text>
      )}
    </Screen>
  )
}

const styles = StyleSheet.create({
  hero: { marginBottom: spacing.xl },
  eyebrowRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm },
  eyebrow: { color: theme.primaryText },
  title: { marginTop: spacing.md },
  subtitle: { marginTop: spacing.sm },
  cta: { marginTop: spacing.xl, alignSelf: 'stretch' },

  section: { marginVertical: spacing.lg },
  sectionHeader: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  sectionTitle: { marginBottom: spacing.md },

  chipRow: { gap: spacing.sm, paddingRight: spacing.lg },
  chip: {
    borderRadius: radius.pill,
    borderWidth: 1,
    borderColor: theme.border,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    backgroundColor: theme.surface,
  },
  chipPressed: { backgroundColor: theme.surfaceMuted },

  turfList: { gap: spacing.md },

  bookingsRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.md },
  bookingsText: { flex: 1 },

  footer: { textAlign: 'center', marginTop: spacing.sm, marginBottom: spacing.lg },
})
