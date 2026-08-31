import { Ionicons } from '@expo/vector-icons'
import { useCallback, useEffect, useState } from 'react'
import { Image, ScrollView, StyleSheet, View } from 'react-native'
import { useNavigation } from '@react-navigation/native'
import type { NativeStackNavigationProp } from '@react-navigation/native-stack'

import { Button, Divider, EmptyState, ErrorBanner, IconContainer, LoadingView, Screen, StatusDot, Surface, Text } from '../../components'
import { fetchPublicTurf } from '../../services/owners'
import { ApiError } from '../../services/api'
import { useAuth } from '../../hooks'
import { cardPresets, iconSizes, radius, spacing, theme } from '../../theme'
import type { Turf } from '../../types/owners'
import type { PlayerStackParamList } from '../../navigation/types'

// Mounted from both the player and owner stacks under the same route name and
// param shape ({ turfId }); a shared, minimal prop type avoids pinning this
// screen to either stack's full param list. Booking (below) is only ever
// reachable from the player stack, so it's cast at the one call site that
// needs it rather than widening this shared Props type.
interface Props {
  route: { params: { turfId: string } }
}

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; turf: Turf }
  | { kind: 'missing' }
  | { kind: 'failed'; message: string; isNetworkError: boolean }

const THUMBNAIL_SIZE = 56

/** "HH:MM" → minutes since midnight, or null if unparseable. Mirrors
 * `components/TurfCard.tsx`'s own small helper — kept local rather than
 * shared, since this screen's scope is deliberately just itself. */
function parseClock(value: string): number | null {
  const match = /^(\d{1,2}):(\d{2})/.exec(value.trim())
  if (!match) return null
  const hours = Number(match[1])
  const minutes = Number(match[2])
  if (hours > 23 || minutes > 59) return null
  return hours * 60 + minutes
}

function isOpenNow(openingTime: string, closingTime: string): boolean | null {
  const open = parseClock(openingTime)
  const close = parseClock(closingTime)
  if (open === null || close === null || open === close) return null
  const minutesNow = new Date().getHours() * 60 + new Date().getMinutes()
  return open < close ? minutesNow >= open && minutesNow < close : minutesNow >= open || minutesNow < close
}

/** Public detail view for one turf, reachable only while it is APPROVED.
 * Organized so the case for booking builds top to bottom: what it looks
 * like, what it is, what it costs, whether it's open now — then the one
 * action every other section is building toward. */
export function TurfDetailScreen({ route }: Props) {
  const { turfId } = route.params
  const [state, setState] = useState<State>({ kind: 'loading' })
  const { user } = useAuth()
  const isPlayer = user?.role === 'PLAYER'
  // Only the player stack registers a "Booking" route; this cast is safe
  // exactly because the CTA below is gated on isPlayer.
  const navigation = useNavigation<NativeStackNavigationProp<PlayerStackParamList>>()

  const load = useCallback(() => {
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
        setState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load this turf.',
          isNetworkError: error instanceof ApiError && error.isNetworkError,
        })
      })

    return () => {
      cancelled = true
    }
  }, [turfId])

  useEffect(() => {
    return load()
  }, [load])

  if (state.kind === 'loading') {
    return <LoadingView message="Loading turf" />
  }

  if (state.kind === 'missing') {
    return (
      <Screen scroll={false}>
        <EmptyState
          title="Turf not found"
          message="This turf does not exist, or is not open for booking yet."
          icon={
            <IconContainer tone="muted" size="lg">
              <Ionicons name="location-outline" size={iconSizes.lg} color={theme.textMuted} />
            </IconContainer>
          }
        />
      </Screen>
    )
  }

  if (state.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner
          message={state.message}
          onRetry={load}
          kind={state.isNetworkError ? 'network' : 'generic'}
        />
      </Screen>
    )
  }

  const { turf } = state
  const heroImage = turf.images[0]
  const galleryImages = turf.images.slice(1, 5)
  const openStatus = isOpenNow(turf.opening_time, turf.closing_time)

  return (
    <Screen scroll={false} contentStyle={styles.noPadding}>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {heroImage ? (
          <Image source={{ uri: heroImage.image_url }} style={styles.hero} />
        ) : (
          <View style={[styles.hero, styles.heroPlaceholder]}>
            <Ionicons name="football-outline" size={iconSizes.xl} color={theme.textMuted} />
          </View>
        )}

        {galleryImages.length > 0 && (
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.gallery}>
            {galleryImages.map((image) => (
              <Image key={image.id} source={{ uri: image.image_url }} style={styles.thumbnail} />
            ))}
          </ScrollView>
        )}

        <View style={styles.body}>
          <Text variant="screenTitle" style={styles.name}>
            {turf.name}
          </Text>

          <View style={styles.locationRow}>
            <Ionicons name="location-outline" size={iconSizes.sm} color={theme.textMuted} style={styles.locationIcon} />
            <Text variant="body" color="secondary" style={styles.locationText}>
              {turf.address}, {turf.city}
            </Text>
          </View>

          {turf.sports.length > 0 && (
            <Text variant="bodyEmphasized" color="secondary" style={styles.sportsText}>
              {turf.sports.map(s => s.name).join(' · ')}
              {turf.amenities.length > 0 && ' · ' + turf.amenities.map(a => a.name).join(' · ')}
            </Text>
          )}

          {turf.description ? (
            <>
              <Divider style={styles.divider} />
              <Text variant="body">
                {turf.description}
              </Text>
            </>
          ) : null}

          <Divider style={styles.divider} />

          <View style={styles.hoursRow}>
            <StatusDot active={openStatus} />
            <Text variant="bodyEmphasized" color={openStatus ? 'success' : 'secondary'} numberOfLines={1}>
              {openStatus === null ? 'Hours' : openStatus ? 'Open now' : 'Closed now'}
            </Text>
            <Text variant="body" color="muted" numberOfLines={1}>{`  ${turf.opening_time} – ${turf.closing_time}`}</Text>
          </View>

          {turf.capacity !== undefined ? (
            <Text variant="body" color="muted" style={styles.capacityText}>{`${turf.capacity} players maximum`}</Text>
          ) : null}
        </View>
      </ScrollView>

      <View style={styles.stickyFooter}>
        <View style={styles.footerPriceBlock}>
          {turf.slot_price !== undefined ? (
            <>
              <Text variant="priceEmphasis" color="primary">
                {`₹${turf.slot_price}`}
              </Text>
              <Text variant="caption" color="muted">
                {turf.slot_duration_minutes ? `per ${turf.slot_duration_minutes}-min slot` : 'per slot'}
              </Text>
            </>
          ) : (
            <Text variant="bodyEmphasized" color="secondary">
              Price on request
            </Text>
          )}
        </View>

        {isPlayer ? (
          <Button
            label="Book this turf"
            onPress={() => navigation.navigate('Booking', { turfId: turf.id })}
            style={styles.ctaButton}
          />
        ) : (
          <Button label="Players only" onPress={() => {}} disabled style={styles.ctaButton} />
        )}
      </View>
    </Screen>
  )
}

const styles = StyleSheet.create({
  noPadding: { padding: 0 },
  scrollContent: { paddingBottom: spacing.xxl * 3 },
  hero: { width: '100%', height: 280 }, // Taller, bleeding image
  heroPlaceholder: { backgroundColor: theme.surfaceMuted, alignItems: 'center', justifyContent: 'center' },
  gallery: { gap: spacing.sm, marginTop: spacing.sm, paddingHorizontal: spacing.lg },
  thumbnail: { width: THUMBNAIL_SIZE, height: THUMBNAIL_SIZE, borderRadius: radius.sm },
  body: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  name: { marginTop: spacing.xs },
  locationRow: { flexDirection: 'row', alignItems: 'flex-start', gap: spacing.xs, marginTop: spacing.xs },
  locationIcon: { marginTop: 2 },
  locationText: { flexShrink: 1 },
  sportsText: { marginTop: spacing.sm },
  divider: { marginVertical: spacing.lg },
  hoursRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.xs },
  capacityText: { marginTop: spacing.xs },
  stickyFooter: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    backgroundColor: theme.surface,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.border,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: spacing.xl,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  footerPriceBlock: { flexShrink: 1, marginRight: spacing.md },
  ctaButton: { paddingHorizontal: spacing.xl },
})
