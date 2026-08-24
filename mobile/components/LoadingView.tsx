import { useEffect, useRef } from 'react'
import { ActivityIndicator, Animated, StyleSheet, Text } from 'react-native'

import { useReducedMotion } from '../hooks'
import { durations, easings, spacing, theme, typography } from '../theme'

interface LoadingViewProps {
  message?: string
  /** 'large' (default) for a screen's initial load; 'small' for a compact
   * inline loader (e.g. inside a card, or a list's "loading more" footer). */
  size?: 'small' | 'large'
  /** Off (default): fills the available height and centers, for a
   * screen's own loading state. On: sits inline within its parent instead
   * of taking over the layout — use alongside `size="small"`. */
  inline?: boolean
}

/** A centred spinner with an optional message. Fades in over a fast
 * duration rather than appearing instantly — softens the flash when a
 * screen swaps its content for this on every navigation. Skipped under
 * Reduce Motion, where it just renders at full opacity immediately. */
export function LoadingView({ message, size = 'large', inline = false }: LoadingViewProps) {
  const reducedMotion = useReducedMotion()
  const opacity = useRef(new Animated.Value(reducedMotion ? 1 : 0)).current

  useEffect(() => {
    if (reducedMotion) return
    Animated.timing(opacity, {
      toValue: 1,
      duration: durations.fast,
      easing: easings.standard,
      useNativeDriver: true,
    }).start()
  }, [reducedMotion, opacity])

  return (
    <Animated.View
      style={[styles.container, inline && styles.inline, { opacity }]}
      accessibilityRole="progressbar"
    >
      <ActivityIndicator color={theme.primary} size={size} />
      {message ? <Text style={styles.message}>{message}</Text> : null}
    </Animated.View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: spacing.md },
  inline: { flex: 0, paddingVertical: spacing.lg },
  message: { ...typography.body, color: theme.textSecondary },
})
