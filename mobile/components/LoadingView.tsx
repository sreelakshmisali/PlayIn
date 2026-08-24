import { ActivityIndicator, StyleSheet, View } from 'react-native'

import { spacing, theme } from '../theme'
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
 */
export function LoadingView({ message, size = 'large', inline = false }: LoadingViewProps) {
  return (
    <View style={[styles.container, inline && styles.inline]} accessibilityRole="progressbar">
      <ActivityIndicator color={theme.primary} size={size} />
      {message ? (
        <Text variant="body" color="secondary" style={styles.message}>
          {message}
        </Text>
      ) : null}
    </View>
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
