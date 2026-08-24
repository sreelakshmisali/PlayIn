import { ActivityIndicator, StyleSheet, Text, View } from 'react-native'

import { spacing, theme, typography } from '../theme'

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

/** A centred spinner with an optional message. */
export function LoadingView({ message, size = 'large', inline = false }: LoadingViewProps) {
  return (
    <View style={[styles.container, inline && styles.inline]} accessibilityRole="progressbar">
      <ActivityIndicator color={theme.primary} size={size} />
      {message ? <Text style={styles.message}>{message}</Text> : null}
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: spacing.md },
  inline: { flex: 0, paddingVertical: spacing.lg },
  message: { ...typography.body, color: theme.textSecondary },
})
