import { StyleSheet, Text, View } from 'react-native'

import { Button, Screen } from '../components'
import { useAuth } from '../hooks'
import { spacing, theme, typography } from '../theme'

/**
 * Shown to an ADMIN account that signs into the mobile app. Admin stays a web
 * experience for now (see CLAUDE.md); this is an honest "not here yet" rather
 * than silently defaulting an admin into the player or owner flow.
 */
export function NotAvailableScreen() {
  const { logout } = useAuth()

  return (
    <Screen scroll={false}>
      <View style={styles.body}>
        <Text style={styles.title}>Admin isn't on mobile yet</Text>
        <Text style={styles.subtitle}>
          Turf moderation and user management are still a web-only experience. Use the admin web app for now.
        </Text>
        <View style={styles.action}>
          <Button label="Sign out" variant="secondary" onPress={() => void logout()} />
        </View>
      </View>
    </Screen>
  )
}

const styles = StyleSheet.create({
  body: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  title: { ...typography.heading, color: theme.textPrimary, textAlign: 'center' },
  subtitle: { ...typography.body, color: theme.textSecondary, textAlign: 'center', marginTop: spacing.sm },
  action: { marginTop: spacing.xl, alignSelf: 'stretch' },
})
