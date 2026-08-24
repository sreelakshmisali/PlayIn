import { Ionicons } from '@expo/vector-icons'
import { Pressable, StyleSheet, View } from 'react-native'

import { iconSizes, radius, spacing, theme } from '../theme'
import { EmptyState } from './EmptyState'
import { IconContainer } from './IconContainer'
import { Text } from './Text'

/**
 * A slot's display state. `'BOOKED'` is included for when it becomes real:
 * the backend's `Slot.status` today is only ever `OPEN`/`BLOCKED` — a
 * `BOOKED` value is documented in `backend/internal/owners/slot_model.go`
 * as a later addition ("Phase 6 adds a BOOKED value here"), not something
 * the API returns yet. Until then, `slotDisplayState` below can only ever
 * produce `'AVAILABLE'`/`'UNAVAILABLE'` — nothing here invents a booked
 * slot that doesn't exist.
 */
export type TimeSlotState = 'AVAILABLE' | 'SELECTED' | 'BOOKED' | 'UNAVAILABLE'

export interface TimeSlotOption {
  id: string
  /** "HH:MM", as the API's `Slot.start_time` already is — rendered as-is,
   * no reformatting. */
  startTime: string
  state: Exclude<TimeSlotState, 'SELECTED'>
}

/**
 * Pure presentation mapping from a real backend slot to a display state —
 * it doesn't decide availability, only how to label what the API already
 * said. Pass the slot's own `status`/`available` fields straight through.
 */
export function slotDisplayState(slot: { status: string; available: boolean }): Exclude<TimeSlotState, 'SELECTED'> {
  if (slot.status === 'BOOKED') return 'BOOKED'
  return slot.available ? 'AVAILABLE' : 'UNAVAILABLE'
}

interface TimeSlotGridProps {
  slots: TimeSlotOption[]
  selectedSlotId: string | null
  onSelectSlot: (id: string) => void
}

/**
 * A compact, wrapping grid of time slots for one date — built for scanning
 * a whole day's slots at once, not a scrolling list. Every cell uses the
 * same shape and label; only fill, border and text communicate state, so
 * a glance across the grid is enough to find what's open.
 */
export function TimeSlotGrid({ slots, selectedSlotId, onSelectSlot }: TimeSlotGridProps) {
  if (slots.length === 0) {
    return (
      <EmptyState
        icon={
          <IconContainer tone="muted" size="lg">
            <Ionicons name="time-outline" size={iconSizes.lg} color={theme.textMuted} />
          </IconContainer>
        }
        title="No slots available"
        message="There are no bookable time slots for this date. Try another day."
      />
    )
  }

  return (
    <View style={styles.grid}>
      {slots.map((slot) => {
        const state: TimeSlotState = slot.id === selectedSlotId ? 'SELECTED' : slot.state
        const isDisabled = state === 'BOOKED' || state === 'UNAVAILABLE'

        return (
          <Pressable
            key={slot.id}
            onPress={() => onSelectSlot(slot.id)}
            disabled={isDisabled}
            accessibilityRole="button"
            accessibilityState={{ selected: state === 'SELECTED', disabled: isDisabled }}
            accessibilityLabel={`${slot.startTime}${state === 'BOOKED' ? ', already booked' : state === 'UNAVAILABLE' ? ', unavailable' : ''}`}
            style={[
              styles.cell,
              state === 'SELECTED' && styles.cellSelected,
              (state === 'BOOKED' || state === 'UNAVAILABLE') && styles.cellDisabled,
            ]}
          >
            <Text
              variant="bodyEmphasized"
              color={state === 'SELECTED' ? 'onPrimary' : isDisabled ? 'disabled' : 'primary'}
              style={[state === 'BOOKED' && styles.bookedLabel]}
            >
              {slot.startTime}
            </Text>
            {state === 'BOOKED' && (
              <Text variant="metadata" color="disabled">
                Booked
              </Text>
            )}
          </Pressable>
        )
      })}
    </View>
  )
}

const styles = StyleSheet.create({
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.sm },
  cell: {
    minWidth: 76,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: theme.border,
    backgroundColor: theme.surface,
    alignItems: 'center',
    justifyContent: 'center',
  },
  cellSelected: { backgroundColor: theme.primary, borderColor: theme.primary },
  cellDisabled: { backgroundColor: theme.surfaceMuted, borderColor: theme.border },
  bookedLabel: { textDecorationLine: 'line-through' },
})
