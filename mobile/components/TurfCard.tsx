import { Ionicons } from '@expo/vector-icons'
import { Image, StyleSheet, View } from 'react-native'

import { cardPresets, iconSizes, radius, spacing, theme } from '../theme'
import type { Turf } from '../types/owners'
import { StatusBadge } from './StatusBadge'
import { Surface } from './Surface'
import { Text } from './Text'

/** Thumbnail edge, in px. Deliberately small — this is a list row, not a
 * gallery: the image helps identify the venue at a glance, it isn't the
 * point of the card. */
const THUMBNAIL_SIZE = 72
/** How many sport chips to show before collapsing the rest into "+N". */
const MAX_SPORT_CHIPS = 2

/** "HH:MM" (optionally with seconds) → minutes since midnight, or null if
 * unparseable. */
function parseClock(value: string): number | null {
  const match = /^(\d{1,2}):(\d{2})/.exec(value.trim())
  if (!match) return null
  const hours = Number(match[1])
  const minutes = Number(match[2])
  if (hours > 23 || minutes > 59) return null
  return hours * 60 + minutes
}

/**
 * A lightweight "open now" hint from the turf's own posted hours, compared
 * against the device's local clock — not a live slot check (that's
 * `GET /turfs/{id}/availability`, queried per-date on the detail screen,
 * not something a list of cards should fan out N calls for). Null when the
 * hours can't be read as a signal (missing, unparseable, or a 24-hour
 * turf where "open now" says nothing useful).
 */
function isOpenNow(openingTime: string, closingTime: string): boolean | null {
  const open = parseClock(openingTime)
  const close = parseClock(closingTime)
  if (open === null || close === null || open === close) return null

  const now = new Date()
  const minutesNow = now.getHours() * 60 + now.getMinutes()

  return open < close
    ? minutesNow >= open && minutesNow < close
    : minutesNow >= open || minutesNow < close // overnight window, e.g. 18:00–02:00
}

interface TurfCardProps {
  turf: Turf
  onPress: () => void
  /** Off in public browsing, where every result is APPROVED and the badge
   * would be noise; on for an owner's own list, which has every status. */
  showStatus?: boolean
}

/** One turf in a list: a single tappable row, not a grid — the mobile-first
 * shape for a list someone scrolls through with a thumb. */
export function TurfCard({ turf, onPress, showStatus }: TurfCardProps) {
  const image = turf.images[0]
  const extraSports = turf.sports.length - MAX_SPORT_CHIPS
  const openStatus = isOpenNow(turf.opening_time, turf.closing_time)

  return (
    <Surface onPress={onPress} accessibilityLabel={turf.name}>
      <View style={styles.row}>
        {image ? (
          <Image source={{ uri: image.image_url }} style={styles.thumbnail} />
        ) : (
          <View style={[styles.thumbnail, styles.thumbnailPlaceholder]}>
            <Ionicons name="football-outline" size={iconSizes.lg} color={theme.primary} />
          </View>
        )}

        <View style={styles.info}>
          <View style={styles.nameRow}>
            <Text variant="bodyEmphasized" numberOfLines={1} style={styles.name}>
              {turf.name}
            </Text>
            {showStatus && <StatusBadge status={turf.status} />}
          </View>

          <View style={styles.locationRow}>
            <Ionicons name="location-outline" size={iconSizes.sm} color={theme.textMuted} />
            <Text variant="caption" color="secondary" numberOfLines={1} style={styles.locationText}>
              {turf.city}
            </Text>
          </View>

          {turf.sports.length > 0 && (
            <View style={styles.sports}>
              {turf.sports.slice(0, MAX_SPORT_CHIPS).map((sport) => (
                <View key={sport.id} style={styles.sportPill}>
                  <Text variant="caption" color="secondary">
                    {sport.name}
                  </Text>
                </View>
              ))}
              {extraSports > 0 && (
                <View style={styles.sportPill}>
                  <Text variant="caption" color="secondary">{`+${extraSports}`}</Text>
                </View>
              )}
            </View>
          )}
        </View>
      </View>

      <View style={styles.metaRow}>
        {turf.slot_price !== undefined ? (
          <Text variant="priceEmphasis" color="primary" numberOfLines={1}>{`₹${turf.slot_price}`}</Text>
        ) : (
          <Text variant="caption" color="muted">
            Price on request
          </Text>
        )}

        <View style={styles.hoursRow}>
          {openStatus !== null && (
            <View style={[styles.statusDot, { backgroundColor: openStatus ? theme.success : theme.textMuted }]} />
          )}
          <Ionicons name="time-outline" size={iconSizes.sm} color={theme.textMuted} />
          <Text variant="metadata" color="muted" numberOfLines={1} style={styles.hoursText}>{`${turf.opening_time} – ${turf.closing_time}`}</Text>
          <Ionicons name="chevron-forward" size={iconSizes.sm} color={theme.textMuted} />
        </View>
      </View>
    </Surface>
  )
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', gap: spacing.md },
  thumbnail: { width: THUMBNAIL_SIZE, height: THUMBNAIL_SIZE, borderRadius: radius.md },
  thumbnailPlaceholder: { backgroundColor: theme.primarySurface, alignItems: 'center', justifyContent: 'center' },
  info: { flex: 1, justifyContent: 'center' },
  nameRow: { flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', gap: spacing.sm },
  name: { color: theme.textPrimary, flexShrink: 1 },
  locationRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.xs, marginTop: spacing.xs / 2 },
  locationText: { flexShrink: 1 },
  sports: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.xs, marginTop: spacing.xs },
  sportPill: { ...cardPresets.pill, backgroundColor: theme.surfaceMuted },
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: spacing.md,
    paddingTop: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.border,
  },
  // flexShrink so a long "HH:MM – HH:MM" range gives way to the price
  // (the more important figure) instead of pushing the row wider than
  // the card on a narrow screen.
  hoursRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.xs, flexShrink: 1, marginLeft: spacing.sm },
  hoursText: { flexShrink: 1 },
  statusDot: { width: 6, height: 6, borderRadius: 3 },
})
