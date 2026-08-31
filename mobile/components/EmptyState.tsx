import type { ReactNode } from 'react'
import { StyleSheet, View } from 'react-native'

import { spacing } from '../theme'
import { Button } from './Button'
import { Text } from './Text'

interface EmptyStateProps {
  /** A short headline. Optional — when omitted, only the message shows,
   * which is fine for compact empty states inside a tab or section. */
  title?: string
  /** The explanatory line. Always present. */
  message: string
  /** An optional visual above the text — typically an `IconContainer` with
   * a relevant Ionicon. Keep it muted; this is a quiet state, not a
   * marketing moment. */
  icon?: ReactNode
  actionLabel?: string
  onAction?: () => void
}

/**
 * The "nothing here" state. Used when a list is empty, a profile does not
 * exist yet, or a search returned zero results. Visually quiet: no dashed
 * borders, no illustrations, just centered text with optional icon and CTA.
 * Shares the same vertical centering and text hierarchy as `LoadingView`
 * and `ErrorBanner` so transitions between states are smooth.
 */
export function EmptyState({ title, message, icon, actionLabel, onAction }: EmptyStateProps) {
  return (
    <View style={styles.container}>
      {icon ? <View style={styles.icon}>{icon}</View> : null}
      {title ? (
        <Text variant="sectionTitle" color="primary" style={styles.title}>
          {title}
        </Text>
      ) : null}
      <Text variant="body" color="secondary" style={styles.message}>
        {message}
      </Text>
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
    paddingVertical: 64,
    paddingHorizontal: spacing.lg,
    alignItems: 'center',
  },
  icon: {
    marginBottom: spacing.lg,
  },
  title: {
    textAlign: 'center',
    marginBottom: spacing.xs,
  },
  message: {
    textAlign: 'center',
    maxWidth: 280,
  },
  action: {
    marginTop: spacing.xl,
    alignSelf: 'stretch',
  },
})
