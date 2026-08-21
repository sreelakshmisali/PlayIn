import { StyleSheet, Text, View } from 'react-native'

import { colors, radius, spacing, typography } from '../theme'
import type { TurfStatus } from '../types/owners'

const STYLES: Record<TurfStatus, { bg: string; fg: string }> = {
  DRAFT: { bg: colors.neutral100, fg: colors.neutral600 },
  PENDING_APPROVAL: { bg: colors.amber50, fg: colors.amber700 },
  APPROVED: { bg: colors.pitch50, fg: colors.pitch700 },
  REJECTED: { bg: colors.red50, fg: colors.red700 },
  SUSPENDED: { bg: colors.red50, fg: colors.red700 },
}

const LABELS: Record<TurfStatus, string> = {
  DRAFT: 'Draft',
  PENDING_APPROVAL: 'Pending review',
  APPROVED: 'Approved',
  REJECTED: 'Rejected',
  SUSPENDED: 'Suspended',
}

/** A small colour-coded label for a turf's place in the approval workflow. */
export function StatusBadge({ status }: { status: TurfStatus }) {
  const style = STYLES[status]
  return (
    <View style={[styles.badge, { backgroundColor: style.bg }]}>
      <Text style={[styles.label, { color: style.fg }]}>{LABELS[status]}</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  badge: { borderRadius: radius.pill, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  label: { ...typography.caption, fontWeight: '600' },
})
