import { Ionicons } from '@expo/vector-icons'
import { useCallback, useEffect, useState } from 'react'
import { Pressable, StyleSheet, View } from 'react-native'
import { useNavigation } from '@react-navigation/native'
import type { NativeStackNavigationProp } from '@react-navigation/native-stack'

import { Divider, EmptyState, IconContainer, LoadingView, Screen, Surface, Text } from '../../components'
import { ErrorBanner } from '../../components/ErrorBanner'
import { fetchMyBookings } from '../../services/bookings'
import { ApiError } from '../../services/api'
import { cardPresets, fontWeights, iconSizes, minTouchTarget, radius, spacing, theme, typography } from '../../theme'
import type { Booking, BookingStatus } from '../../types/bookings'
import type { PlayerStackParamList } from '../../navigation/types'

// ---------------------------------------------------------------------------
// Tab filter
// ---------------------------------------------------------------------------

type Tab = 'upcoming' | 'past' | 'cancelled'

const TABS: { key: Tab; label: string }[] = [
  { key: 'upcoming', label: 'Upcoming' },
  { key: 'past', label: 'Past' },
  { key: 'cancelled', label: 'Cancelled' },
]

function todayISO(): string {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${month}-${day}`
}

function bucketFor(booking: Booking): Tab {
  if (booking.status === 'CANCELLED') return 'cancelled'
  // CONFIRMED: upcoming if date >= today, past otherwise. The backend has
  // no third "completed" status — a CONFIRMED booking whose date has
  // passed is still, honestly, CONFIRMED; this bucket is where that reads
  // as "past" without inventing a status the API never sends.
  return booking.date >= todayISO() ? 'upcoming' : 'past'
}

// ---------------------------------------------------------------------------
// Status presentation
// ---------------------------------------------------------------------------

const STATUS_CONFIG: Record<BookingStatus, { label: string; bg: string; fg: string; icon: keyof typeof Ionicons.glyphMap }> = {
  CONFIRMED: { label: 'Confirmed', bg: theme.successSurface, fg: theme.successText, icon: 'checkmark-circle-outline' },
  CANCELLED: { label: 'Cancelled', bg: theme.dangerSurface, fg: theme.dangerText, icon: 'close-circle-outline' },
}

function StatusPill({ status }: { status: BookingStatus }) {
  const config = STATUS_CONFIG[status]
  return (
    <View style={[styles.statusPill, { backgroundColor: config.bg }]}>
      <Ionicons name={config.icon} size={iconSizes.xs} color={config.fg} />
      <Text variant="metadata" style={{ color: config.fg, fontWeight: fontWeights.semibold }}>
        {config.label}
      </Text>
    </View>
  )
}

// ---------------------------------------------------------------------------
// Date formatting
// ---------------------------------------------------------------------------

function formatDate(iso: string): string {
  const parsed = new Date(`${iso}T00:00:00`)
  if (Number.isNaN(parsed.getTime())) return iso
  return parsed.toLocaleDateString('en-US', { weekday: 'short', day: 'numeric', month: 'short' })
}

function formatTimeRange(start: string, end: string): string {
  return `${start} – ${end}`
}

// ---------------------------------------------------------------------------
// Booking card
// ---------------------------------------------------------------------------

function BookingCard({ booking, onPress }: { booking: Booking; onPress: () => void }) {
  return (
    <Pressable
      style={({ pressed }) => [styles.card, pressed && { backgroundColor: theme.surfaceMuted }]}
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Booking at ${booking.turf.name}`}
    >
      <View style={styles.cardHeader}>
        <View style={styles.sportRow}>
          <IconContainer tone="primary" size="sm">
            <Ionicons name="football-outline" size={iconSizes.sm} color={theme.primary} />
          </IconContainer>
          <Text variant="bodyEmphasized" color="primary" numberOfLines={1} style={styles.turfNameHeader}>
            {booking.turf.name}
          </Text>
        </View>
        <StatusPill status={booking.status} />
      </View>

      <View style={styles.infoRow}>
        <Ionicons name="location-outline" size={iconSizes.sm} color={theme.textMuted} />
        <Text variant="caption" color="secondary" numberOfLines={1} style={styles.infoText}>
          {booking.turf.address}, {booking.turf.city}
        </Text>
      </View>

      <View style={styles.detailsRow}>
        <View style={styles.detailItem}>
          <Ionicons name="calendar-outline" size={iconSizes.sm} color={theme.textMuted} />
          <Text variant="caption" color="primary" numberOfLines={1}>{formatDate(booking.date)}</Text>
        </View>
        <View style={styles.detailItem}>
          <Ionicons name="time-outline" size={iconSizes.sm} color={theme.textMuted} />
          <Text variant="caption" color="primary" numberOfLines={1}>{formatTimeRange(booking.start_time, booking.end_time)}</Text>
        </View>
        <Text variant="bodyEmphasized" color="primary" numberOfLines={1} style={styles.price}>
          {`₹${booking.price}`}
        </Text>
      </View>
      <View style={styles.actionRow}>
        <Text variant="bodyEmphasized" color="primary">View details</Text>
        <Ionicons name="arrow-forward" size={iconSizes.sm} color={theme.textPrimary} />
      </View>
    </Pressable>
  )
}

// ---------------------------------------------------------------------------
// Tab bar
// ---------------------------------------------------------------------------

function TabBar({ active, counts, onSelect }: { active: Tab; counts: Record<Tab, number>; onSelect: (tab: Tab) => void }) {
  return (
    <View style={styles.tabBar}>
      {TABS.map(({ key, label }) => {
        const isActive = key === active
        return (
          <Pressable
            key={key}
            onPress={() => onSelect(key)}
            accessibilityRole="tab"
            accessibilityState={{ selected: isActive }}
            style={[styles.tab, isActive && styles.tabActive]}
          >
            <Text
              variant="caption"
              style={[styles.tabLabel, isActive && styles.tabLabelActive]}
            >
              {label}
            </Text>
            {counts[key] > 0 && (
              <View style={[styles.countBadge, isActive && styles.countBadgeActive]}>
                <Text variant="metadata" style={[styles.countText, isActive && styles.countTextActive]}>
                  {counts[key]}
                </Text>
              </View>
            )}
          </Pressable>
        )
      })}
    </View>
  )
}

// ---------------------------------------------------------------------------
// Empty states per tab
// ---------------------------------------------------------------------------

const EMPTY_MESSAGES: Record<Tab, { title: string; message: string; icon: keyof typeof Ionicons.glyphMap }> = {
  upcoming: {
    title: 'No upcoming bookings',
    message: 'Browse turfs to find your next game.',
    icon: 'calendar-outline',
  },
  past: {
    title: 'No past bookings',
    message: 'Your completed sessions will show here.',
    icon: 'time-outline',
  },
  cancelled: {
    title: 'No cancelled bookings',
    message: 'Cancelled sessions will show here.',
    icon: 'close-circle-outline',
  },
}

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; bookings: Booking[] }
  | { kind: 'failed'; message: string; isNetworkError: boolean }

// A generous single page rather than paging UI: comfortably covers a
// player's real booking history without the added complexity of
// infinite-scroll/"load more" for what is, in practice, a short list.
const PAGE_SIZE = 100

export function MyBookingsScreen() {
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [activeTab, setActiveTab] = useState<Tab>('upcoming')
  const navigation = useNavigation<NativeStackNavigationProp<PlayerStackParamList>>()

  const load = useCallback(() => {
    setState({ kind: 'loading' })
    fetchMyBookings(PAGE_SIZE, 0)
      .then((page) => setState({ kind: 'ready', bookings: page.bookings }))
      .catch((error: unknown) => {
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load your bookings.',
          isNetworkError: error instanceof ApiError && error.isNetworkError,
        })
      })
  }, [])

  useEffect(() => { load() }, [load])

  if (state.kind === 'loading') {
    return <LoadingView message="Loading bookings" />
  }

  if (state.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner message={state.message} onRetry={load} kind={state.isNetworkError ? 'network' : 'generic'} />
      </Screen>
    )
  }

  const { bookings } = state

  // Bucket bookings into tabs
  const buckets: Record<Tab, Booking[]> = { upcoming: [], past: [], cancelled: [] }
  for (const booking of bookings) {
    buckets[bucketFor(booking)].push(booking)
  }

  // Sort: upcoming by date ascending, past/cancelled by date descending
  buckets.upcoming.sort((a, b) => a.date.localeCompare(b.date) || a.start_time.localeCompare(b.start_time))
  buckets.past.sort((a, b) => b.date.localeCompare(a.date) || b.start_time.localeCompare(a.start_time))
  buckets.cancelled.sort((a, b) => b.date.localeCompare(a.date) || b.start_time.localeCompare(a.start_time))

  const counts: Record<Tab, number> = {
    upcoming: buckets.upcoming.length,
    past: buckets.past.length,
    cancelled: buckets.cancelled.length,
  }

  const filtered = buckets[activeTab]
  const empty = EMPTY_MESSAGES[activeTab]

  return (
    <Screen>
      <TabBar active={activeTab} counts={counts} onSelect={setActiveTab} />

      {filtered.length === 0 ? (
        <View style={styles.emptyContainer}>
          <EmptyState
            title={empty.title}
            message={empty.message}
            icon={
              <IconContainer tone="muted" size="lg">
                <Ionicons name={empty.icon} size={iconSizes.lg} color={theme.textMuted} />
              </IconContainer>
            }
          />
        </View>
      ) : (
        // A plain mapped list, not a nested FlatList: this screen's only
        // scroll container is the Screen itself, and a non-scrolling
        // FlatList inside another ScrollView only renders its initial
        // batch — everything past it stays invisible since nothing ever
        // drives its own virtualization forward.
        <View style={styles.list}>
          {filtered.map((item) => (
            <BookingCard key={item.id} booking={item} onPress={() => navigation.navigate('BookingDetail', { bookingId: item.id })} />
          ))}
        </View>
      )}
    </Screen>
  )
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const styles = StyleSheet.create({
  // Tab bar
  tabBar: {
    flexDirection: 'row',
    gap: spacing.sm,
    marginBottom: spacing.lg,
  },
  tab: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: minTouchTarget,
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    borderBottomWidth: 2,
    borderBottomColor: 'transparent',
  },
  tabActive: {
    borderBottomColor: theme.primary,
  },
  tabLabel: {
    ...typography.bodyEmphasized,
    color: theme.textSecondary,
  },
  tabLabelActive: {
    color: theme.primary,
  },
  countBadge: {
    minWidth: 18,
    height: 18,
    borderRadius: 9,
    backgroundColor: theme.surfaceMuted,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: spacing.xs,
  },
  countBadgeActive: {
    backgroundColor: theme.primary,
  },
  countText: {
    ...typography.metadata,
    fontWeight: fontWeights.semibold,
    color: theme.textSecondary,
  },
  countTextActive: {
    color: theme.textOnPrimary,
  },

  // List
  list: {
    gap: spacing.md,
  },
  emptyContainer: {
    marginTop: spacing.xxl,
  },

  // Card
  card: {
    gap: spacing.xs,
    paddingVertical: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.border,
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: spacing.xs,
  },
  sportRow: {
    flexDirection: 'row',
    alignItems: 'center',
    flexShrink: 1,
    gap: spacing.xs,
  },
  turfNameHeader: {
    flexShrink: 1,
  },
  infoRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
    marginTop: spacing.xs,
  },
  infoText: {
    flexShrink: 1,
  },
  detailsRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    alignItems: 'center',
    rowGap: spacing.xs,
    columnGap: spacing.md,
  },
  detailItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
  },
  price: {
    marginLeft: 'auto',
  },

  // Status pill
  statusPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
  },

  // Action row
  actionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: minTouchTarget,
    paddingTop: spacing.xs,
  },
  actionText: {
    color: theme.primary,
    fontWeight: fontWeights.medium,
  },
})
