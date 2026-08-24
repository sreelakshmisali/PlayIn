import { Ionicons } from '@expo/vector-icons'
import { useCallback, useEffect, useState } from 'react'
import { FlatList, Pressable, StyleSheet, View } from 'react-native'

import { Divider, EmptyState, IconContainer, LoadingView, Screen, Surface, Text } from '../../components'
import { ErrorBanner } from '../../components/ErrorBanner'
import { fetchMyBookings } from '../../services/bookings'
import { ApiError } from '../../services/api'
import { cardPresets, iconSizes, radius, spacing, theme, typography } from '../../theme'
import type { Booking, BookingStatus } from '../../types/bookings'

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
  if (booking.status === 'COMPLETED') return 'past'
  // CONFIRMED: upcoming if date >= today, past otherwise
  return booking.date >= todayISO() ? 'upcoming' : 'past'
}

// ---------------------------------------------------------------------------
// Status presentation
// ---------------------------------------------------------------------------

const STATUS_CONFIG: Record<BookingStatus, { label: string; bg: string; fg: string; icon: keyof typeof Ionicons.glyphMap }> = {
  CONFIRMED: { label: 'Confirmed', bg: theme.successSurface, fg: theme.successText, icon: 'checkmark-circle-outline' },
  COMPLETED: { label: 'Completed', bg: theme.surfaceMuted, fg: theme.textSecondary, icon: 'checkmark-done-outline' },
  CANCELLED: { label: 'Cancelled', bg: theme.dangerSurface, fg: theme.dangerText, icon: 'close-circle-outline' },
}

function StatusPill({ status }: { status: BookingStatus }) {
  const config = STATUS_CONFIG[status]
  return (
    <View style={[styles.statusPill, { backgroundColor: config.bg }]}>
      <Ionicons name={config.icon} size={12} color={config.fg} />
      <Text variant="metadata" style={{ color: config.fg, fontWeight: '600' }}>
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

function BookingCard({ booking }: { booking: Booking }) {
  const config = STATUS_CONFIG[booking.status]

  return (
    <Surface variant="card" style={styles.card}>
      {/* Top row: sport + status */}
      <View style={styles.cardHeader}>
        <View style={styles.sportRow}>
          <IconContainer tone="primary" size="sm">
            <Ionicons name="football-outline" size={iconSizes.sm} color={theme.primary} />
          </IconContainer>
          <Text variant="caption" color="secondary">{booking.sport_name}</Text>
        </View>
        <StatusPill status={booking.status} />
      </View>

      {/* Turf name */}
      <Text variant="bodyEmphasized" color="primary" style={styles.turfName}>
        {booking.turf.name}
      </Text>

      {/* Location */}
      <View style={styles.infoRow}>
        <Ionicons name="location-outline" size={iconSizes.sm} color={theme.textMuted} />
        <Text variant="caption" color="secondary" numberOfLines={1} style={styles.infoText}>
          {booking.turf.address}, {booking.turf.city}
        </Text>
      </View>

      <Divider spacing="md" />

      {/* Date, time, price row */}
      <View style={styles.detailsRow}>
        <View style={styles.detailItem}>
          <Ionicons name="calendar-outline" size={14} color={theme.textMuted} />
          <Text variant="caption" color="primary">{formatDate(booking.date)}</Text>
        </View>
        <View style={styles.detailItem}>
          <Ionicons name="time-outline" size={14} color={theme.textMuted} />
          <Text variant="caption" color="primary">{formatTimeRange(booking.start_time, booking.end_time)}</Text>
        </View>
        <Text variant="bodyEmphasized" color="primary" style={styles.price}>
          {`₹${booking.price}`}
        </Text>
      </View>

      {/* Action row for upcoming bookings */}
      {booking.status === 'CONFIRMED' && booking.date >= todayISO() && (
        <>
          <Divider spacing="md" />
          <Pressable
            style={styles.actionRow}
            accessibilityRole="button"
            accessibilityLabel="View booking details"
            onPress={() => {
              // Navigation to booking detail will be wired when the route exists
            }}
          >
            <Text variant="caption" color="primary" style={styles.actionText}>
              View details
            </Text>
            <Ionicons name="chevron-forward" size={14} color={theme.primary} />
          </Pressable>
        </>
      )}
    </Surface>
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

const EMPTY_MESSAGES: Record<Tab, { message: string; icon: keyof typeof Ionicons.glyphMap }> = {
  upcoming: { message: 'No upcoming bookings. Browse turfs to find your next game.', icon: 'calendar-outline' },
  past: { message: 'No past bookings yet. Your completed sessions will show here.', icon: 'time-outline' },
  cancelled: { message: 'No cancelled bookings.', icon: 'close-circle-outline' },
}

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; bookings: Booking[] }
  | { kind: 'failed'; message: string }

export function MyBookingsScreen() {
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [activeTab, setActiveTab] = useState<Tab>('upcoming')

  const load = useCallback(() => {
    setState({ kind: 'loading' })
    fetchMyBookings()
      .then((bookings) => setState({ kind: 'ready', bookings }))
      .catch((error: unknown) => {
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load your bookings.',
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
        <ErrorBanner message={state.message} />
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
            message={empty.message}
            icon={
              <IconContainer tone="muted" size="lg">
                <Ionicons name={empty.icon} size={iconSizes.lg} color={theme.textMuted} />
              </IconContainer>
            }
          />
        </View>
      ) : (
        <FlatList
          data={filtered}
          keyExtractor={(item) => item.id}
          renderItem={({ item }) => <BookingCard booking={item} />}
          contentContainerStyle={styles.list}
          showsVerticalScrollIndicator={false}
          scrollEnabled={false}
        />
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
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radius.pill,
    backgroundColor: theme.surfaceMuted,
  },
  tabActive: {
    backgroundColor: theme.primary,
  },
  tabLabel: {
    ...typography.caption,
    fontWeight: '500',
    color: theme.textSecondary,
  },
  tabLabelActive: {
    color: theme.textOnPrimary,
  },
  countBadge: {
    minWidth: 18,
    height: 18,
    borderRadius: 9,
    backgroundColor: theme.border,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 4,
  },
  countBadgeActive: {
    backgroundColor: 'rgba(255, 255, 255, 0.25)',
  },
  countText: {
    ...typography.metadata,
    fontWeight: '600',
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
    gap: 0,
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  sportRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
  },
  turfName: {
    marginTop: spacing.sm,
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
    alignItems: 'center',
    gap: spacing.md,
  },
  detailItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  price: {
    marginLeft: 'auto',
  },

  // Status pill
  statusPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 3,
  },

  // Action row
  actionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.xs,
  },
  actionText: {
    color: theme.primary,
    fontWeight: '500',
  },
})
