import { useState } from 'react'
import { StyleSheet, Text, View } from 'react-native'

import { Button, Screen } from '../components'
import { useAuth } from '../hooks'
import { spacing, theme, typography } from '../theme'

/** The signed-in account view: who you are, and the one way out. Shared
 * between the player and owner flows, since signing out is not role-specific. */
export function AccountScreen() {
  const { user, logout } = useAuth()
  const [signingOut, setSigningOut] = useState(false)

  if (!user) return null

  return (
    <Screen scroll={false}>
      <View style={styles.header}>
        <Text style={styles.name}>{user.full_name}</Text>
        <Text style={styles.email}>{user.email}</Text>
        <View style={styles.roleBadge}>
          <Text style={styles.roleBadgeText}>{user.role}</Text>
        </View>
      </View>

      <View style={styles.card}>
        <Row label="Status" value={user.is_active ? 'Active' : 'Deactivated'} />
        <Row label="Member since" value={new Date(user.created_at).toLocaleDateString()} last />
      </View>

      <View style={styles.signOut}>
        <Button
          label={signingOut ? 'Signing out' : 'Sign out'}
          variant="secondary"
          pending={signingOut}
          onPress={() => {
            setSigningOut(true)
            void logout()
          }}
        />
      </View>
    </Screen>
  )
}

function Row({ label, value, last }: { label: string; value: string; last?: boolean }) {
  return (
    <View style={[styles.row, last && styles.rowLast]}>
      <Text style={styles.rowLabel}>{label}</Text>
      <Text style={styles.rowValue}>{value}</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  header: { marginBottom: spacing.xl },
  name: { ...typography.screenTitle, color: theme.textPrimary },
  email: { ...typography.body, color: theme.textSecondary, marginTop: spacing.xs },
  roleBadge: {
    marginTop: spacing.sm,
    backgroundColor: theme.primarySurface,
    borderRadius: 999,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
  },
  roleBadgeText: { ...typography.caption, color: theme.primaryText, fontWeight: '600' },
  card: {
    marginTop: spacing.xl,
    borderWidth: 1,
    borderColor: theme.border,
    borderRadius: 12,
    overflow: 'hidden',
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.border,
  },
  rowLast: { borderBottomWidth: 0 },
  rowLabel: { ...typography.caption, color: theme.textSecondary },
  rowValue: { ...typography.bodyEmphasized, color: theme.textPrimary },
  signOut: { marginTop: spacing.xl },
})
