import { Pressable, StyleSheet, Text, View } from 'react-native'

import { radius, spacing, theme, typography } from '../theme'
import type { Turf } from '../types/owners'
import { StatusBadge } from './StatusBadge'

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
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      style={({ pressed }) => [styles.card, pressed && styles.pressed]}
    >
      <View style={styles.header}>
        <Text style={styles.name} numberOfLines={1}>
          {turf.name}
        </Text>
        {showStatus && <StatusBadge status={turf.status} />}
      </View>
      <Text style={styles.city} numberOfLines={1}>
        {turf.city}
      </Text>
      {turf.sports.length > 0 && (
        <View style={styles.sports}>
          {turf.sports.slice(0, 3).map((sport) => (
            <View key={sport.id} style={styles.sportPill}>
              <Text style={styles.sportPillText}>{sport.name}</Text>
            </View>
          ))}
        </View>
      )}
    </Pressable>
  )
}

const styles = StyleSheet.create({
  card: {
    borderWidth: 1,
    borderColor: theme.border,
    borderRadius: radius.lg,
    padding: spacing.lg,
    backgroundColor: theme.surface,
  },
  pressed: { backgroundColor: theme.surfaceMuted },
  header: { flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', gap: spacing.sm },
  name: { ...typography.heading, color: theme.textPrimary, flexShrink: 1 },
  city: { ...typography.caption, color: theme.textSecondary, marginTop: spacing.xs / 2 },
  sports: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.xs, marginTop: spacing.sm },
  sportPill: {
    backgroundColor: theme.surfaceMuted,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 4,
  },
  sportPillText: { ...typography.caption, color: theme.textSecondary },
})
