import { useEffect, useState } from 'react'
import { StyleSheet, Text, View } from 'react-native'

import { EmptyState, ErrorBanner, LoadingView, Screen } from '../../components'
import { fetchPublicTurf } from '../../services/owners'
import { ApiError } from '../../services/api'
import { spacing, theme, typography } from '../../theme'
import type { Turf } from '../../types/owners'

// Mounted from both the player and owner stacks under the same route name and
// param shape ({ turfId }); a shared, minimal prop type avoids pinning this
// screen to either stack's full param list.
interface Props {
  route: { params: { turfId: string } }
}

type State = { kind: 'loading' } | { kind: 'ready'; turf: Turf } | { kind: 'missing' } | { kind: 'failed'; message: string }

/** Public detail view for one turf, reachable only while it is APPROVED. */
export function TurfDetailScreen({ route }: Props) {
  const { turfId } = route.params
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    let cancelled = false
    setState({ kind: 'loading' })

    fetchPublicTurf(turfId)
      .then((turf) => {
        if (!cancelled) setState({ kind: 'ready', turf })
      })
      .catch((error: unknown) => {
        if (cancelled) return
        if (error instanceof ApiError && error.status === 404) {
          setState({ kind: 'missing' })
          return
        }
        setState({ kind: 'failed', message: error instanceof ApiError ? error.message : 'Could not load this turf.' })
      })

    return () => {
      cancelled = true
    }
  }, [turfId])

  if (state.kind === 'loading') {
    return <LoadingView message="Loading turf" />
  }

  if (state.kind === 'missing') {
    return (
      <Screen scroll={false}>
        <EmptyState message="This turf does not exist, or is not open for booking yet." />
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

  const { turf } = state

  return (
    <Screen>
      <Text style={styles.name}>{turf.name}</Text>
      <Text style={styles.address}>
        {turf.address}, {turf.city}
      </Text>
      <Text style={styles.listedBy}>Listed by {turf.owner_display_name}</Text>

      {turf.description ? <Text style={styles.description}>{turf.description}</Text> : null}

      <View style={styles.factsCard}>
        <Row label="Hours" value={`${turf.opening_time} – ${turf.closing_time}`} />
        {turf.capacity !== undefined ? <Row label="Capacity" value={`${turf.capacity} players`} /> : null}
      </View>

      {turf.sports.length > 0 && (
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Sports</Text>
          <View style={styles.pillRow}>
            {turf.sports.map((sport) => (
              <View key={sport.id} style={styles.pill}>
                <Text style={styles.pillText}>{sport.name}</Text>
              </View>
            ))}
          </View>
        </View>
      )}

      {turf.amenities.length > 0 && (
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Amenities</Text>
          <View style={styles.pillRow}>
            {turf.amenities.map((amenity) => (
              <View key={amenity.id} style={[styles.pill, styles.pillAccent]}>
                <Text style={[styles.pillText, styles.pillTextAccent]}>{amenity.name}</Text>
              </View>
            ))}
          </View>
        </View>
      )}
    </Screen>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.row}>
      <Text style={styles.rowLabel}>{label}</Text>
      <Text style={styles.rowValue}>{value}</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  name: { ...typography.title, color: theme.textPrimary },
  address: { ...typography.body, color: theme.textSecondary, marginTop: spacing.xs },
  listedBy: { ...typography.caption, color: theme.textMuted, marginTop: spacing.xs / 2 },
  description: { ...typography.body, color: theme.textPrimary, marginTop: spacing.lg },
  factsCard: {
    marginTop: spacing.lg,
    borderWidth: 1,
    borderColor: theme.border,
    borderRadius: 12,
    overflow: 'hidden',
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.border,
  },
  rowLabel: { ...typography.body, color: theme.textSecondary },
  rowValue: { ...typography.bodyMedium, color: theme.textPrimary },
  section: { marginTop: spacing.xl },
  sectionTitle: { ...typography.label, color: theme.textPrimary, marginBottom: spacing.sm },
  pillRow: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.sm },
  pill: { backgroundColor: theme.surfaceMuted, borderRadius: 999, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  pillAccent: { backgroundColor: theme.primarySurface },
  pillText: { ...typography.caption, color: theme.textSecondary },
  pillTextAccent: { color: theme.primaryText },
})
