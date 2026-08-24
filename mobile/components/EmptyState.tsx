import type { ReactNode } from 'react'
import { StyleSheet, Text, View } from 'react-native'

import { radius, spacing, theme, typography } from '../theme'
import { Button } from './Button'

interface EmptyStateProps {
  message: string
  /** An optional `IconContainer` (or any element) shown above the message. */
  icon?: ReactNode
  actionLabel?: string
  onAction?: () => void
}

/** A dashed placeholder for "nothing here yet", with an optional icon and
 * next step. */
export function EmptyState({ message, icon, actionLabel, onAction }: EmptyStateProps) {
  return (
    <View style={styles.container}>
      {icon ? <View style={styles.icon}>{icon}</View> : null}
      <Text style={styles.message}>{message}</Text>
      {actionLabel && onAction ? (
        <View style={styles.action}>
          <Button label={actionLabel} onPress={onAction} variant="secondary" />
        </View>
      ) : null}
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    borderWidth: 1,
    borderStyle: 'dashed',
    borderColor: theme.border,
    borderRadius: radius.lg,
    padding: spacing.xl,
    alignItems: 'center',
  },
  icon: { marginBottom: spacing.md },
  message: { ...typography.body, color: theme.textSecondary, textAlign: 'center' },
  action: { marginTop: spacing.lg, alignSelf: 'stretch' },
})
