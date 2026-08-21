import { StyleSheet, Text, View } from 'react-native'

import { radius, spacing, theme, typography } from '../theme'

/**
 * The form- or screen-level failure message: the one a field error cannot
 * express, such as wrong credentials or an unreachable API.
 */
export function ErrorBanner({ message }: { message: string }) {
  return (
    <View style={styles.container} accessibilityRole="alert">
      <Text style={styles.message}>{message}</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: theme.dangerSurface,
    borderWidth: 1,
    borderColor: theme.dangerBorder,
    borderRadius: radius.md,
    padding: spacing.md,
    marginBottom: spacing.lg,
  },
  message: { ...typography.body, color: theme.dangerText },
})
