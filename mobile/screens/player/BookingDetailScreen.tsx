import { Ionicons } from '@expo/vector-icons'
import { useCallback, useEffect, useState } from 'react'
import { StyleSheet, View } from 'react-native'

import { Button, Divider, ErrorBanner, IconContainer, LoadingView, Screen, Surface, Text } from '../../components'
import { cancelBooking, fetchBooking } from '../../services/bookings'
import { ApiError } from '../../services/api'
import { fontWeights, iconSizes, radius, spacing, theme } from '../../theme'
import type { Booking, BookingStatus } from '../../types/bookings'

/** Mirrors `MyBookingsScreen`'s own status pill so a booking's status reads
 * identically wherever it appears — kept local to each screen rather than
 * shared, the same reasoning `TurfDetailScreen`'s clock helpers use. */
const STATUS_CONFIG: Record<BookingStatus, { label: string; bg: string; fg: string; icon: keyof typeof Ionicons.glyphMap }> = {
  CONFIRMED: { label: 'Confirmed', bg: theme.successSurface, fg: theme.successText, icon: 'checkmark-circle-outline' },
  CANCELLED: { label: 'Cancelled', bg: theme.dangerSurface, fg: theme.dangerText, icon: 'close-circle-outline' },
}

function StatusPill({ status }: { status: BookingStatus }) {
  const config = STATUS_CONFIG[status]
  return (
    <View style={[pillStyles.pill, { backgroundColor: config.bg }]}>
      <Ionicons name={config.icon} size={iconSizes.xs} color={config.fg} />
      <Text variant="metadata" style={{ color: config.fg, fontWeight: fontWeights.semibold }}>
        {config.label}
      </Text>
    </View>
  )
}

const pillStyles = StyleSheet.create({
  pill: { flexDirection: 'row', alignItems: 'center', gap: spacing.xs, borderRadius: radius.pill, paddingHorizontal: spacing.sm, paddingVertical: spacing.xs },
})

interface Props {
  route: { params: { bookingId: string } }
}

function formatDate(iso: string): string {
  const parsed = new Date(`${iso}T00:00:00`)
  if (Number.isNaN(parsed.getTime())) return iso
  return parsed.toLocaleDateString('en-US', { weekday: 'long', day: 'numeric', month: 'long' })
}

function todayISO(): string {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${month}-${day}`
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
 * A single booking's full detail, with cancellation for one still eligible
 * — CONFIRMED and not yet in the past. Cancelling calls the real endpoint
 * and waits for its response before updating anything on screen; nothing
 * here marks a booking cancelled locally ahead of the server confirming it.
 */
export function BookingDetailScreen({ route }: Props) {
  const { bookingId } = route.params
  const [state, setState] = useState<State>({ kind: 'loading' })
  const [confirmingCancel, setConfirmingCancel] = useState(false)
  const [cancelling, setCancelling] = useState(false)
  const [cancelError, setCancelError] = useState<string | null>(null)

  const load = useCallback(() => {
    setState({ kind: 'loading' })
    fetchBooking(bookingId)
      .then((booking) => setState({ kind: 'ready', booking }))
      .catch((error: unknown) => {
        setState({ kind: 'failed', message: error instanceof ApiError ? error.message : 'Could not load this booking.' })
      })
  }, [bookingId])

  useEffect(() => {
    load()
  }, [load])

  const handleCancel = () => {
    setCancelling(true)
    setCancelError(null)
    cancelBooking(bookingId)
      .then((booking) => {
        setState({ kind: 'ready', booking })
        setConfirmingCancel(false)
      })
      .catch((error: unknown) => {
        setCancelError(error instanceof ApiError ? error.message : 'Could not cancel this booking.')
      })
      .finally(() => setCancelling(false))
  }

  if (state.kind === 'loading') {
    return <LoadingView message="Loading booking" />
  }

  if (state.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner message={state.message} onRetry={load} />
      </Screen>
    )
  }

  const { booking } = state
  const isCancellable = booking.status === 'CONFIRMED' && booking.date >= todayISO()

  return (
    <Screen>
      <View style={styles.header}>
        <IconContainer tone={booking.status === 'CANCELLED' ? 'danger' : 'primary'} size="lg">
          <Ionicons
            name={booking.status === 'CANCELLED' ? 'close-circle-outline' : 'football-outline'}
            size={iconSizes.lg}
            color={booking.status === 'CANCELLED' ? theme.danger : theme.primary}
          />
        </IconContainer>
        <View style={styles.headerText}>
          <Text variant="screenTitle">{booking.turf.name}</Text>
          <Text variant="caption" color="secondary">
            {booking.turf.address}, {booking.turf.city}
          </Text>
        </View>
        <StatusPill status={booking.status} />
      </View>

      <Surface variant="card" style={styles.card}>
        <Row label="Date" value={formatDate(booking.date)} />
        <Row label="Time" value={`${booking.start_time} – ${booking.end_time}`} />
        <Divider spacing="md" />
        <View style={styles.priceRow}>
          <Text variant="metadata" color="muted">
            Price
          </Text>
          <Text variant="priceEmphasis" color="primary">
            {`₹${booking.price}`}
          </Text>
        </View>
        {booking.cancelled_at ? (
          <>
            <Divider spacing="md" />
            <Text variant="caption" color="muted">
              Cancelled on {formatDate(booking.cancelled_at.slice(0, 10))}
            </Text>
          </>
        ) : null}
      </Surface>

      {isCancellable && (
        <View style={styles.cancelSection}>
          {cancelError && <ErrorBanner message={cancelError} />}
          {confirmingCancel ? (
            <>
              <Text variant="body" color="secondary" style={styles.confirmText}>
                Cancel this booking? This can't be undone.
              </Text>
              <View style={styles.confirmActions}>
                <Button
                  label="Keep booking"
                  variant="secondary"
                  onPress={() => setConfirmingCancel(false)}
                  disabled={cancelling}
                  style={styles.confirmButton}
                />
                <Button
                  label="Cancel booking"
                  variant="danger"
                  onPress={handleCancel}
                  pending={cancelling}
                  style={styles.confirmButton}
                />
              </View>
            </>
          ) : (
            <Button label="Cancel booking" variant="danger" onPress={() => setConfirmingCancel(true)} />
          )}
        </View>
      )}
    </Screen>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: spacing.md, marginBottom: spacing.lg },
  headerText: { flex: 1, gap: spacing.xs / 2 },
  card: { gap: 0 },
  row: { flexDirection: 'row', justifyContent: 'space-between', gap: spacing.md, paddingVertical: spacing.xs },
  rowValue: { flexShrink: 1, textAlign: 'right' },
  priceRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  cancelSection: { marginTop: spacing.xl, gap: spacing.md },
  confirmText: { textAlign: 'center' },
  confirmActions: { flexDirection: 'row', gap: spacing.md },
  confirmButton: { flex: 1 },
})
