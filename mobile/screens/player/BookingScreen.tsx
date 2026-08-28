import { useCallback, useEffect, useMemo, useState } from 'react'
import { StyleSheet, View } from 'react-native'
import type { NativeStackNavigationProp } from '@react-navigation/native-stack'
import { useNavigation } from '@react-navigation/native'

import { BookingSummary, DateStrip, ErrorBanner, LoadingView, Screen, Text, TimeSlotGrid, slotDisplayState } from '../../components'
import type { TimeSlotOption } from '../../components'
import { createBooking } from '../../services/bookings'
import { fetchPublicTurf, fetchTurfAvailability } from '../../services/owners'
import { ApiError } from '../../services/api'
import { spacing } from '../../theme'
import type { Turf, Slot } from '../../types/owners'
import type { PlayerStackParamList } from '../../navigation/types'

interface Props {
  route: { params: { turfId: string } }
}

/** How many upcoming dates the strip offers — enough to plan a few days
 * out without turning into a full calendar picker (see `DateStrip`'s own
 * reasoning: the backend only answers "is this one date available", not a
 * bulk range). */
const DATE_RANGE_DAYS = 14

function upcomingDates(count: number): string[] {
  const out: string[] = []
  const start = new Date()
  for (let i = 0; i < count; i++) {
    const d = new Date(start)
    d.setDate(start.getDate() + i)
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    out.push(`${d.getFullYear()}-${month}-${day}`)
  }
  return out
}

type TurfState = { kind: 'loading' } | { kind: 'ready'; turf: Turf } | { kind: 'failed'; message: string; isNetworkError: boolean }
type SlotsState = { kind: 'idle' } | { kind: 'loading' } | { kind: 'ready'; slots: Slot[] } | { kind: 'failed'; message: string; isNetworkError: boolean }

/**
 * The booking flow's one screen: pick a date, pick an open slot, review and
 * confirm — the middle stretch between "Book this turf" on `TurfDetailScreen`
 * and a created `Booking`. Every date/slot shown comes from a real API call;
 * nothing here invents availability.
 */
export function BookingScreen({ route }: Props) {
  const { turfId } = route.params
  const navigation = useNavigation<NativeStackNavigationProp<PlayerStackParamList>>()

  const dates = useMemo(() => upcomingDates(DATE_RANGE_DAYS), [])
  const [turfState, setTurfState] = useState<TurfState>({ kind: 'loading' })
  const [selectedDate, setSelectedDate] = useState<string | null>(dates[0] ?? null)
  const [slotsState, setSlotsState] = useState<SlotsState>({ kind: 'idle' })
  const [selectedSlotId, setSelectedSlotId] = useState<string | null>(null)
  const [confirming, setConfirming] = useState(false)
  const [confirmError, setConfirmError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    fetchPublicTurf(turfId)
      .then((turf) => {
        if (!cancelled) setTurfState({ kind: 'ready', turf })
      })
      .catch((error: unknown) => {
        if (cancelled) return
        setTurfState({
          kind: 'failed',
          message: error instanceof ApiError ? error.message : 'Could not load this turf.',
          isNetworkError: error instanceof ApiError && error.isNetworkError,
        })
      })
    return () => {
      cancelled = true
    }
  }, [turfId])

  const loadSlots = useCallback(
    (date: string) => {
      setSlotsState({ kind: 'loading' })
      setSelectedSlotId(null)
      fetchTurfAvailability(turfId, date)
        .then((slots) => setSlotsState({ kind: 'ready', slots }))
        .catch((error: unknown) => {
          setSlotsState({
            kind: 'failed',
            message: error instanceof ApiError ? error.message : 'Could not load availability for this date.',
            isNetworkError: error instanceof ApiError && error.isNetworkError,
          })
        })
    },
    [turfId],
  )

  useEffect(() => {
    if (selectedDate) loadSlots(selectedDate)
  }, [selectedDate, loadSlots])

  const handleSelectDate = (date: string) => {
    setSelectedDate(date)
    setConfirmError(null)
  }

  const handleSelectSlot = (slotId: string) => {
    setSelectedSlotId((current) => (current === slotId ? null : slotId))
    setConfirmError(null)
  }

  const handleConfirm = () => {
    if (!selectedSlotId) return
    setConfirming(true)
    setConfirmError(null)
    createBooking(selectedSlotId)
      .then((booking) => {
        navigation.replace('BookingConfirmation', { bookingId: booking.id })
      })
      .catch((error: unknown) => {
        setConfirming(false)
        if (error instanceof ApiError && error.status === 409) {
          // Slot was taken between load and confirm — refresh so the grid
          // reflects reality instead of leaving a stale "open" cell tappable.
          setConfirmError(error.message || 'This slot is no longer open for booking.')
          if (selectedDate) loadSlots(selectedDate)
          return
        }
        setConfirmError(error instanceof ApiError ? error.message : 'Could not confirm this booking.')
      })
  }

  if (turfState.kind === 'loading') {
    return <LoadingView message="Loading turf" />
  }

  if (turfState.kind === 'failed') {
    return (
      <Screen scroll={false}>
        <ErrorBanner
          message={turfState.message}
          onRetry={() => setTurfState({ kind: 'loading' })}
          kind={turfState.isNetworkError ? 'network' : 'generic'}
        />
      </Screen>
    )
  }

  const { turf } = turfState
  const slots: TimeSlotOption[] =
    slotsState.kind === 'ready'
      ? slotsState.slots.map((slot) => ({ id: slot.id, startTime: slot.start_time, state: slotDisplayState(slot) }))
      : []
  const selectedSlot = slotsState.kind === 'ready' ? slotsState.slots.find((s) => s.id === selectedSlotId) ?? null : null

  return (
    <Screen>
      <Text variant="screenTitle" style={styles.title}>
        {turf.name}
      </Text>
      <Text variant="body" color="secondary" style={styles.subtitle}>
        {turf.address}, {turf.city}
      </Text>

      <Text variant="label" color="primary" style={styles.sectionLabel}>
        Date
      </Text>
      <DateStrip dates={dates} selectedDate={selectedDate} onSelectDate={handleSelectDate} />

      <Text variant="label" color="primary" style={styles.sectionLabel}>
        Time
      </Text>
      {slotsState.kind === 'loading' && <LoadingView size="small" inline message="Loading slots" />}
      {slotsState.kind === 'failed' && (
        <ErrorBanner message={slotsState.message} onRetry={() => selectedDate && loadSlots(selectedDate)} kind={slotsState.isNetworkError ? 'network' : 'generic'} />
      )}
      {slotsState.kind === 'ready' && (
        <TimeSlotGrid slots={slots} selectedSlotId={selectedSlotId} onSelectSlot={handleSelectSlot} />
      )}

      {selectedSlot && (
        <View style={styles.summary}>
          {confirmError && <ErrorBanner message={confirmError} />}
          <BookingSummary
            details={{
              turfName: turf.name,
              turfLocation: `${turf.address}, ${turf.city}`,
              date: selectedSlot.date,
              startTime: selectedSlot.start_time,
              durationMinutes: turf.slot_duration_minutes,
              price: selectedSlot.price,
            }}
            onConfirm={handleConfirm}
            pending={confirming}
            disabled={confirming}
          />
        </View>
      )}
    </Screen>
  )
}

const styles = StyleSheet.create({
  title: { marginTop: spacing.xs },
  subtitle: { marginTop: spacing.xs / 2, marginBottom: spacing.lg },
  sectionLabel: { marginTop: spacing.lg, marginBottom: spacing.sm },
  summary: { marginTop: spacing.xl, gap: spacing.md },
})
