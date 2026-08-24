import { Pressable, ScrollView, StyleSheet, View } from 'react-native'

import { radius, spacing, theme } from '../theme'
import { Text } from './Text'

/** A date as `YYYY-MM-DD`, matching the format the backend's availability
 * endpoint (`GET /turfs/{id}/availability?date=`) takes and returns. */
export type DateStripDate = string

interface DateStripProps {
  /** The dates to render, in order — the caller decides the range (e.g.
   * the next 14 days). This component has no calendar/range logic of its
   * own; it only presents what it's given. */
  dates: DateStripDate[]
  selectedDate: DateStripDate | null
  onSelectDate: (date: DateStripDate) => void
  /** Dates known to have no bookable slots, from a real per-date
   * availability check — e.g. `GET /turfs/{id}/availability`. Omit (or
   * leave empty) when that hasn't been checked yet: with nothing passed,
   * no date is marked unavailable — this component never guesses. */
  unavailableDates?: DateStripDate[]
}

function todayISO(): DateStripDate {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${month}-${day}`
}

function describe(date: DateStripDate) {
  const parsed = new Date(`${date}T00:00:00`)
  return {
    weekday: parsed.toLocaleDateString('en-US', { weekday: 'short' }),
    day: parsed.getDate(),
  }
}

/**
 * A compact, horizontally-scrolling strip of days for picking a booking
 * date. Deliberately not a full month calendar: the backend only answers
 * "is this one date available" per turf, not a bulk range, so a long grid
 * of guessed states would mean inventing availability. States shown here
 * are only ones this component can back with real information — today
 * (from the device clock), past (unselectable, same reasoning), selected
 * (the one place the primary accent fills a surface), and unavailable
 * (only for dates the caller explicitly passes in `unavailableDates`).
 */
export function DateStrip({ dates, selectedDate, onSelectDate, unavailableDates = [] }: DateStripProps) {
  const today = todayISO()
  const unavailable = new Set(unavailableDates)

  return (
    <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.row}>
      {dates.map((date) => {
        const { weekday, day } = describe(date)
        const isToday = date === today
        const isPast = date < today
        const isSelected = date === selectedDate
        const isDisabled = isPast || unavailable.has(date)

        return (
          <Pressable
            key={date}
            onPress={() => onSelectDate(date)}
            disabled={isDisabled}
            accessibilityRole="button"
            accessibilityState={{ selected: isSelected, disabled: isDisabled }}
            accessibilityLabel={`${weekday} ${day}${isToday ? ', today' : ''}${isDisabled ? ', unavailable' : ''}`}
            style={[
              styles.cell,
              isSelected && styles.cellSelected,
              isToday && !isSelected && styles.cellToday,
              isDisabled && !isSelected && styles.cellDisabled,
            ]}
          >
            <Text
              variant="caption"
              color={isSelected ? 'onPrimary' : isDisabled ? 'disabled' : 'secondary'}
            >
              {weekday}
            </Text>
            <Text variant="bodyEmphasized" color={isSelected ? 'onPrimary' : isDisabled ? 'disabled' : 'primary'}>
              {day}
            </Text>
            {isToday && !isSelected && <View style={styles.todayDot} />}
          </Pressable>
        )
      })}
    </ScrollView>
  )
}

const CELL_WIDTH = 52

const styles = StyleSheet.create({
  row: { gap: spacing.sm },
  cell: {
    width: CELL_WIDTH,
    paddingVertical: spacing.sm,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: theme.border,
    backgroundColor: theme.surface,
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.xs / 2,
  },
  cellSelected: { backgroundColor: theme.primary, borderColor: theme.primary },
  cellToday: { borderColor: theme.primary },
  cellDisabled: { backgroundColor: theme.surfaceMuted, borderColor: theme.border },
  todayDot: { width: 4, height: 4, borderRadius: 2, backgroundColor: theme.primary, marginTop: 2 },
})
