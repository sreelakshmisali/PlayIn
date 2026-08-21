import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs'
import { StyleSheet, Text, View } from 'react-native'

import { Button, Screen } from '../../components'
import { useAuth } from '../../hooks'
import { spacing, theme, typography } from '../../theme'

// HomeScreen is mounted inside both PlayerTabParamList and OwnerTabParamList,
// which share the same three sibling route names ('Turfs', 'Profile',
// 'Account') but are otherwise distinct param lists. Typing this against
// either one by name would be misleading, so the navigation prop is accepted
// loosely rather than pinned to one role's tab list.
type Props = BottomTabScreenProps<Record<string, object | undefined>>

/** The landing tab: a greeting and the one thing everyone wants next — turfs. */
export function HomeScreen({ navigation }: Props) {
  const { user } = useAuth()

  return (
    <Screen scroll={false}>
      <View style={styles.body}>
        <Text style={styles.eyebrow}>PlayHub</Text>
        <Text style={styles.title}>
          {user ? `Welcome back, ${user.full_name.split(' ')[0]}` : 'Welcome'}
        </Text>
        <Text style={styles.subtitle}>Find a turf, check its availability, and get playing.</Text>

        <View style={styles.actions}>
          <Button label="Browse turfs" onPress={() => navigation.navigate('Turfs')} />
        </View>
      </View>
    </Screen>
  )
}

const styles = StyleSheet.create({
  body: { flex: 1, justifyContent: 'center' },
  eyebrow: { ...typography.label, color: theme.primaryText },
  title: { ...typography.title, color: theme.textPrimary, marginTop: spacing.xs },
  subtitle: { ...typography.body, color: theme.textSecondary, marginTop: spacing.sm },
  actions: { marginTop: spacing.xl },
})
