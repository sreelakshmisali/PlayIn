import { useEffect, useRef } from 'react'
import { ActivityIndicator, Animated, StyleSheet } from 'react-native'

import { useReducedMotion } from '../hooks'
import { durations, easings, spacing, theme } from '../theme'
import { Text } from './Text'

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

/**
 * A centered spinner with an optional message. Uses the same vertical
 * centering and text hierarchy as `EmptyState` and `ErrorBanner` so that
 * loading → loaded and loading → error transitions don't jolt the layout.
 *
 * Fades in over a fast duration rather than appearing instantly — softens
 * the flash when a screen swaps its content for this on every navigation.
 * Skipped under Reduce Motion, where it just renders at full opacity
 * immediately.
 */
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
      {message ? (
        <Text variant="body" color="secondary" style={styles.message}>
          {message}
        </Text>
      ) : null}
    </Animated.View>
  )
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.md,
  },
  inline: {
    flex: 0,
    paddingVertical: spacing.xl,
  },
  message: {
    textAlign: 'center',
  },
})
