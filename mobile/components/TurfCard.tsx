import { Ionicons } from '@expo/vector-icons'
import { Image, StyleSheet, View } from 'react-native'

import { cardPresets, iconSizes, radius, spacing, theme } from '../theme'
import type { Turf } from '../types/owners'
import { StatusBadge } from './StatusBadge'
import { StatusDot } from './StatusDot'
import { Surface } from './Surface'
import { Text } from './Text'

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
    <Surface onPress={onPress} accessibilityLabel={turf.name} style={styles.surface}>
      <View style={styles.imageContainer}>
        {image ? (
          <Image source={{ uri: image.image_url }} style={styles.image} />
        ) : (
          <View style={[styles.image, styles.imagePlaceholder]}>
            <Ionicons name="football-outline" size={iconSizes.xl} color={theme.textMuted} />
          </View>
        )}
        {showStatus && (
          <View style={styles.statusOverlay}>
            <StatusBadge status={turf.status} />
          </View>
        )}
      </View>

      <View style={styles.content}>
        <View style={styles.titleRow}>
          <Text variant="sectionTitle" numberOfLines={1} style={styles.name}>
            {turf.name}
          </Text>
          {turf.slot_price !== undefined ? (
            <Text variant="priceEmphasis" color="primary" style={styles.price}>
              ₹{turf.slot_price}
            </Text>
          ) : (
            <Text variant="caption" color="muted">
              Pricing on request
            </Text>
          )}
        </View>

        <View style={styles.metadataRow}>
          <Text variant="body" color="secondary" numberOfLines={1} style={styles.metadataText}>
            {turf.city}
            {turf.sports.length > 0 && ` · ${turf.sports.slice(0, MAX_SPORT_CHIPS).map(s => s.name).join(', ')}`}
            {extraSports > 0 && ` +${extraSports}`}
          </Text>
        </View>

        <View style={styles.actionRow}>
          <View style={styles.hoursGroup}>
            <StatusDot active={openStatus} />
            <Text variant="caption" color="secondary" numberOfLines={1}>
              {turf.opening_time} – {turf.closing_time}
            </Text>
          </View>
          <Ionicons name="arrow-forward-outline" size={iconSizes.sm} color={theme.textMuted} />
        </View>
      </View>
    </Surface>
  )
}

const styles = StyleSheet.create({
  surface: { padding: 0, overflow: 'hidden' }, // Override default card padding to bleed the image
  imageContainer: {
    height: 160,
    backgroundColor: theme.surfaceMuted,
  },
  image: {
    width: '100%',
    height: '100%',
  },
  imagePlaceholder: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  statusOverlay: {
    position: 'absolute',
    top: spacing.md,
    right: spacing.md,
  },
  content: {
    padding: spacing.lg,
  },
  titleRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: spacing.md,
  },
  name: {
    flex: 1,
  },
  price: {
    flexShrink: 0,
  },
  metadataRow: {
    marginTop: spacing.xs,
    flexDirection: 'row',
    alignItems: 'center',
  },
  metadataText: {
    flex: 1,
  },
  actionRow: {
    marginTop: spacing.md,
    paddingTop: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.border,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  hoursGroup: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
  },
})
