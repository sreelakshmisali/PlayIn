import { StyleSheet, View } from 'react-native'

import { theme } from '../theme'

interface StatusDotProps {
  /** `true` renders success green, `false` renders muted gray, `null`
   * renders nothing — the caller has no signal to show yet (e.g. hours
   * that can't be read as an open/closed hint). */
  active: boolean | null
}

/** A small colored dot for an on/off status hint — e.g. a turf's "open
 * now" indicator next to its posted hours. Shared by `TurfCard` and
 * `TurfDetailScreen` so the same signal looks identical in the list and
 * the detail view. */
export function StatusDot({ active }: StatusDotProps) {
  if (active === null) return null
  return <View style={[styles.dot, { backgroundColor: active ? theme.success : theme.textMuted }]} />
}

const styles = StyleSheet.create({
  dot: { width: 6, height: 6, borderRadius: 3 },
})
