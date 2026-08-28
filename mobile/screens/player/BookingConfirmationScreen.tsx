import { Ionicons } from '@expo/vector-icons'
import { useEffect, useState } from 'react'
import { StyleSheet, View } from 'react-native'
import { CommonActions, useNavigation } from '@react-navigation/native'
import type { NativeStackNavigationProp } from '@react-navigation/native-stack'

import { Button, Divider, ErrorBanner, IconContainer, LoadingView, Screen, Surface, Text } from '../../components'
import { fetchBooking } from '../../services/bookings'
import { ApiError } from '../../services/api'
import { iconSizes, spacing, theme } from '../../theme'
import type { Booking } from '../../types/bookings'
import type { PlayerStackParamList } from '../../navigation/types'

interface Props {
  route: { params: { bookingId: string } }
}

function formatDate(iso: string): string {
  const parsed = new Date(`${iso}T00:00:00`)
  if (Number.isNaN(parsed.getTime())) return iso
  return parsed.toLocaleDateString('en-US', { weekday: 'long', day: 'numeric', month: 'long' })
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.row}>
      <Text variant="body" color="secondary">
        {label}
      </Text>
      <Text variant="bodyEmphasized" color="primary" numberOfLines={1} style={styles.rowValue}>
        {value}
      </Text>
    </View>
  )
}

type State = { kind: 'loading' } | { kind: 'ready'; booking: Booking } | { kind: 'failed'; message: string }

/**
 * The calm landing after a booking is created — no confetti, just the facts
 * of what was reserved and a way forward. Reads the booking back from the
 * API rather than trusting whatever `BookingScreen` had in memory, so this
 * screen always shows what the server actually recorded.
 */
export function BookingConfirmationScreen({ route }: Props) {
  const { bookingId } = route.params
  const navigation = useNavigation<NativeStackNavigationProp<PlayerStackParamList>>()
  const [state, setState] = useState<State>({ kind: 'loading' })

  useEffect(() => {
    let cancelled = false
    fetchBooking(bookingId)
      .then((booking) => {
        if (!cancelled) setState({ kind: 'ready', booking })
      })
      .catch((error: unknown) => {
        if (cancelled) return
        setState({ kind: 'failed', message: error instanceof ApiError ? error.message : 'Could not load this booking.' })
      })
    return () => {
      cancelled = true
    }
  }, [bookingId])

  const goToMyBookings = () => {
    navigation.dispatch(
      CommonActions.reset({
        index: 0,
        routes: [{ name: 'PlayerTabs', state: { routes: [{ name: 'Bookings' }] } }],
      }),
    )
  }

  if (state.kind === 'loading') {
    return <LoadingView message="Confirming booking" />
  }

  if (state.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner message={state.message} />
      </Screen>
    )
  }

  const { booking } = state

  return (
    <Screen background="muted">
      <View style={styles.header}>
        <IconContainer tone="primary" size="xl">
          <Ionicons name="checkmark-circle-outline" size={iconSizes.xl} color={theme.primary} />
        </IconContainer>
        <Text variant="screenTitle" color="primary" style={styles.headline}>
          Booking confirmed
        </Text>
        <Text variant="body" color="secondary" style={styles.subheadline}>
          You're set for {booking.turf.name}.
        </Text>
      </View>

      <Surface variant="card" style={styles.card}>
        <Row label="Turf" value={booking.turf.name} />
        <Row label="Address" value={`${booking.turf.address}, ${booking.turf.city}`} />
        <Divider spacing="md" />
        <Row label="Date" value={formatDate(booking.date)} />
        <Row label="Time" value={`${booking.start_time} – ${booking.end_time}`} />
        <Divider spacing="md" />
        <View style={styles.priceRow}>
          <Text variant="metadata" color="muted">
            Total paid
          </Text>
          <Text variant="priceEmphasis" color="primary">
            {`₹${booking.price}`}
          </Text>
        </View>
      </Surface>

      <Button label="View my bookings" onPress={goToMyBookings} style={styles.cta} />
    </Screen>
  )
}

const styles = StyleSheet.create({
  header: { alignItems: 'center', paddingTop: spacing.xl, paddingBottom: spacing.xl, gap: spacing.sm },
  headline: { textAlign: 'center' },
  subheadline: { textAlign: 'center' },
  card: { gap: 0 },
  row: { flexDirection: 'row', justifyContent: 'space-between', gap: spacing.md, paddingVertical: spacing.xs },
  rowValue: { flexShrink: 1, textAlign: 'right' },
  priceRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  cta: { marginTop: spacing.xl },
})
