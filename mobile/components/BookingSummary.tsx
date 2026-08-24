import { Ionicons } from '@expo/vector-icons'
import { useEffect, useRef } from 'react'
import { Animated, StyleSheet, View } from 'react-native'

import { useReducedMotion } from '../hooks'
import { durations, easings, iconSizes, spacing, theme } from '../theme'
import { Divider } from './Divider'
import { IconContainer } from './IconContainer'
import { Surface } from './Surface'
import { Text } from './Text'
import { Button } from './Button'

export interface BookingSummaryDetails {
  turfName: string
  /** e.g. "{address}, {city}" — shown as a subtitle under the turf name. */
  turfLocation: string
  /** "YYYY-MM-DD", formatted for display here. */
  date: string
  /** "HH:MM", the slot's own start time — formatted/combined with
   * `durationMinutes` for display here, never recomputed as booking logic. */
  startTime: string
  /** Omit when the slot's duration isn't known; the Duration row and the
   * end-time half of the Time row are simply left off rather than guessed. */
  durationMinutes?: number
  /** Omit for "price on request", same fallback used elsewhere in the app. */
  price?: number
}

interface BookingSummaryProps {
  details: BookingSummaryDetails
  onConfirm: () => void
  pending?: boolean
  disabled?: boolean
  /** Any other applicable booking information worth a line before
   * confirming (e.g. a cancellation note) — left to the caller; nothing
   * here invents policy copy that doesn't come from real data. */
  note?: string
}

function formatDate(iso: string): string {
  const parsed = new Date(`${iso}T00:00:00`)
  if (Number.isNaN(parsed.getTime())) return iso
  return parsed.toLocaleDateString('en-US', { weekday: 'short', day: 'numeric', month: 'short' })
}

/** "HH:MM" + minutes → "HH:MM", wrapping past midnight. Display formatting
 * only — it doesn't decide whether a slot is bookable. */
function addMinutes(time: string, minutes: number): string | null {
  const match = /^(\d{1,2}):(\d{2})/.exec(time.trim())
  if (!match) return null
  const total = (Number(match[1]) * 60 + Number(match[2]) + minutes + 1440) % 1440
  const hours = String(Math.floor(total / 60)).padStart(2, '0')
  const mins = String(total % 60).padStart(2, '0')
  return `${hours}:${mins}`
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.row}>
      <Text variant="body" color="secondary">
        {label}
      </Text>
      <Text variant="bodyEmphasized" color="primary">
        {value}
      </Text>
    </View>
  )
}

/**
 * A single, calm summary of what's about to be booked, ending in the one
 * action that matters. No step indicators, no marketing copy — just the
 * facts someone would want confirmed before committing, and a CTA that
 * restates the price so nothing about it is a surprise.
 */
export function BookingSummary({ details, onConfirm, pending = false, disabled = false, note }: BookingSummaryProps) {
  const endTime = details.durationMinutes ? addMinutes(details.startTime, details.durationMinutes) : null
  const timeValue = endTime ? `${details.startTime} – ${endTime}` : details.startTime
  const priceLabel = details.price !== undefined ? `₹${details.price}` : 'Price on request'

  const reducedMotion = useReducedMotion()
  const checkOpacity = useRef(new Animated.Value(reducedMotion ? 1 : 0)).current
  const checkScale = useRef(new Animated.Value(reducedMotion ? 1 : 0.8)).current

  useEffect(() => {
    if (reducedMotion) return
    // A quiet arrival for the one icon that signals "this is confirmable" —
    // fires once, on mount, not on every re-render as fields update.
    Animated.parallel([
      Animated.timing(checkOpacity, {
        toValue: 1,
        duration: durations.base,
        easing: easings.standard,
        useNativeDriver: true,
      }),
      Animated.timing(checkScale, {
        toValue: 1,
        duration: durations.base,
        easing: easings.decelerate,
        useNativeDriver: true,
      }),
    ]).start()
    // checkOpacity/checkScale are stable refs — only reducedMotion should retrigger this.
  }, [reducedMotion])

  return (
    <Surface variant="card">
      <View style={styles.header}>
        <Animated.View style={{ opacity: checkOpacity, transform: [{ scale: checkScale }] }}>
          <IconContainer tone="primary" size="md">
            <Ionicons name="checkmark-circle-outline" size={iconSizes.md} color={theme.primary} />
          </IconContainer>
        </Animated.View>
        <View style={styles.headerText}>
          <Text variant="sectionTitle">Review your booking</Text>
          <Text variant="caption" color="secondary" numberOfLines={1}>
            {details.turfName} · {details.turfLocation}
          </Text>
        </View>
      </View>

      <Divider spacing="md" />

      <Row label="Date" value={formatDate(details.date)} />
      <Row label="Time" value={timeValue} />
      {details.durationMinutes ? <Row label="Duration" value={`${details.durationMinutes} min`} /> : null}

      <Divider spacing="md" />

      <View style={styles.priceRow}>
        <Text variant="metadata" color="muted">
          Total
        </Text>
        <Text variant="priceEmphasis" color="primary">
          {priceLabel}
        </Text>
      </View>

      {note ? (
        <Text variant="caption" color="muted" style={styles.note}>
          {note}
        </Text>
      ) : null}

      <Button
        label={details.price !== undefined ? `Confirm booking · ${priceLabel}` : 'Confirm booking'}
        onPress={onConfirm}
        pending={pending}
        disabled={disabled}
        style={styles.cta}
      />
    </Surface>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: spacing.md },
  headerText: { flex: 1 },
  row: { flexDirection: 'row', justifyContent: 'space-between', paddingVertical: spacing.xs },
  priceRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  note: { marginTop: spacing.sm },
  cta: { marginTop: spacing.lg },
})
