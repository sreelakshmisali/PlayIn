import { ActivityIndicator, StyleSheet, Text, View } from 'react-native'

import { spacing, theme, typography } from '../theme'

/** A centred spinner with an optional message, for a screen's initial load. */
export function LoadingView({ message }: { message?: string }) {
  return (
    <View style={styles.container} accessibilityRole="progressbar">
      <ActivityIndicator color={theme.primary} size="large" />
      {message ? <Text style={styles.message}>{message}</Text> : null}
    </View>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: spacing.md },
  message: { ...typography.body, color: theme.textSecondary },
})
