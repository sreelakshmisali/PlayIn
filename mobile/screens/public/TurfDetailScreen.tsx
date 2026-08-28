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

const HERO_HEIGHT = 200
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
    <Screen>
      {heroImage ? (
        <Image source={{ uri: heroImage.image_url }} style={styles.hero} />
      ) : (
        <View style={[styles.hero, styles.heroPlaceholder]}>
          <Ionicons name="football-outline" size={iconSizes.xl} color={theme.primary} />
        </View>
      )}

      {galleryImages.length > 0 && (
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.gallery}>
          {galleryImages.map((image) => (
            <Image key={image.id} source={{ uri: image.image_url }} style={styles.thumbnail} />
          ))}
        </ScrollView>
      )}

      <Text variant="screenTitle" style={styles.name}>
        {turf.name}
      </Text>

      <View style={styles.locationRow}>
        <Ionicons name="location-outline" size={iconSizes.sm} color={theme.textMuted} style={styles.locationIcon} />
        <Text variant="body" color="secondary" style={styles.locationText}>
          {turf.address}, {turf.city}
        </Text>
      </View>
      <Text variant="metadata" color="muted" style={styles.listedBy}>
        Listed by {turf.owner_display_name}
      </Text>

      {turf.sports.length > 0 && (
        <View style={styles.pillRow}>
          {turf.sports.map((sport) => (
            <View key={sport.id} style={styles.pill}>
              <Text variant="caption" color="secondary">
                {sport.name}
              </Text>
            </View>
          ))}
        </View>
      )}

      {turf.description ? (
        <>
          <Divider />
          <Text variant="body" color="primary">
            {turf.description}
          </Text>
        </>
      ) : null}

      {turf.amenities.length > 0 && (
        <>
          <Divider />
          <Text variant="label" color="primary" style={styles.sectionTitle}>
            Amenities
          </Text>
          <View style={styles.pillRow}>
            {turf.amenities.map((amenity) => (
              <View key={amenity.id} style={[styles.pill, styles.pillAccent]}>
                <Text variant="caption" color="info" style={styles.pillTextAccent}>
                  {amenity.name}
                </Text>
              </View>
            ))}
          </View>
        </>
      )}

      <Divider />

      <Surface variant="muted" style={styles.summary}>
        <View style={styles.summaryRow}>
          <View style={styles.priceBlock}>
            <Text variant="metadata" color="muted">
              Price
            </Text>
            {turf.slot_price !== undefined ? (
              <Text variant="priceEmphasis" color="primary">
                {`₹${turf.slot_price}`}
                <Text variant="caption" color="muted">
                  {turf.slot_duration_minutes ? ` / ${turf.slot_duration_minutes}-min slot` : ' / slot'}
                </Text>
              </Text>
            ) : (
              <Text variant="bodyEmphasized" color="secondary">
                Price on request
              </Text>
            )}
          </View>

          <View style={styles.hoursBlock}>
            <View style={styles.hoursRow}>
              <StatusDot active={openStatus} />
              <Text variant="caption" color={openStatus ? 'success' : 'secondary'} numberOfLines={1}>
                {openStatus === null ? 'Hours' : openStatus ? 'Open now' : 'Closed now'}
              </Text>
            </View>
            <Text variant="metadata" color="muted" numberOfLines={1}>{`${turf.opening_time} – ${turf.closing_time}`}</Text>
            {turf.capacity !== undefined ? (
              <Text variant="metadata" color="muted" numberOfLines={1}>{`${turf.capacity} players`}</Text>
            ) : null}
          </View>
        </View>

        {isPlayer ? (
          <Button
            label="Book this turf"
            onPress={() => navigation.navigate('Booking', { turfId: turf.id })}
            style={styles.cta}
          />
        ) : (
          <>
            <Button label="Book this turf" onPress={() => {}} disabled style={styles.cta} />
            <Text variant="metadata" color="muted" style={styles.ctaNote}>
              Booking is available to players.
            </Text>
          </>
        )}
      </Surface>
    </Screen>
  )
}

const styles = StyleSheet.create({
  hero: { width: '100%', height: HERO_HEIGHT, borderRadius: radius.lg },
  heroPlaceholder: { backgroundColor: theme.primarySurface, alignItems: 'center', justifyContent: 'center' },
  gallery: { gap: spacing.sm, marginTop: spacing.sm },
  thumbnail: { width: THUMBNAIL_SIZE, height: THUMBNAIL_SIZE, borderRadius: radius.sm },
  name: { marginTop: spacing.lg },
  // flex-start (not center) plus a small top nudge on the icon: a long
  // address wraps to two or three lines here, and center-aligning the
  // icon against the whole wrapped block leaves it drifting away from
  // the first line instead of marking it.
  locationRow: { flexDirection: 'row', alignItems: 'flex-start', gap: spacing.xs, marginTop: spacing.xs },
  locationIcon: { marginTop: 2 },
  locationText: { flexShrink: 1 },
  listedBy: { marginTop: spacing.xs / 2 },
  pillRow: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.sm, marginTop: spacing.md },
  pill: { ...cardPresets.pill, backgroundColor: theme.surfaceMuted },
  pillAccent: { backgroundColor: theme.infoSurface },
  pillTextAccent: { color: theme.infoText },
  sectionTitle: { marginBottom: 0 },
  summary: { marginTop: spacing.lg },
  summaryRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-start', gap: spacing.md },
  // Both sides can shrink so a large price or a long hours/capacity line
  // doesn't push this row wider than the card on a narrow screen.
  priceBlock: { flexShrink: 1 },
  hoursBlock: { flexShrink: 1, alignItems: 'flex-end', gap: spacing.xs / 2 },
  hoursRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.xs },
  cta: { marginTop: spacing.lg },
  ctaNote: { textAlign: 'center', marginTop: spacing.sm },
})
