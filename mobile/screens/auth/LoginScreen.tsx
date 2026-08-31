import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { useState } from 'react'
import { Pressable, StyleSheet, Text, View } from 'react-native'

import { Button, ErrorBanner, Screen, TextField } from '../../components'
import { useAuth } from '../../hooks'
import { spacing, theme, typography } from '../../theme'
import { ApiError } from '../../services/api'
import type { AuthStackParamList } from '../../navigation/types'

type Props = NativeStackScreenProps<AuthStackParamList, 'Login'>

export function LoginScreen({ navigation }: Props) {
  const { login } = useAuth()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  async function handleSubmit() {
    setPending(true)
    setError('')
    setFieldErrors({})
    try {
      await login({ email: email.trim().toLowerCase(), password })
      // No manual navigation: RootNavigator swaps to the role-based flow the
      // moment the auth status changes.
    } catch (cause) {
      if (cause instanceof ApiError) {
        setFieldErrors(cause.fieldErrors())
        setError(cause.message)
      } else {
        setError('Something went wrong. Try again.')
      }
    } finally {
      setPending(false)
    }
  }

  return (
    <Screen keyboardSafe>
      <View style={styles.header}>
        <Text style={styles.brand}>
          PlayHub<Text style={styles.brandAccent}>.</Text>
        </Text>
        <Text style={styles.title}>Sign in</Text>
        <Text style={styles.subtitle}>Book turfs, join teams and track your game.</Text>
      </View>

      <View style={styles.form}>
        {error ? <ErrorBanner message={error} /> : null}

        <TextField
          label="Email"
          value={email}
          onChangeText={setEmail}
          error={fieldErrors.email}
          autoCapitalize="none"
          autoComplete="email"
          keyboardType="email-address"
          placeholder="you@example.com"
        />
        <TextField
          label="Password"
          value={password}
          onChangeText={setPassword}
          error={fieldErrors.password}
          secureTextEntry
          autoComplete="password"
          placeholder="Your password"
        />

        <Button label="Sign in" onPress={() => void handleSubmit()} pending={pending} />
      </View>

      <Pressable onPress={() => navigation.navigate('Register')} style={styles.footer}>
        <Text style={styles.footerText}>
          New to PlayHub? <Text style={styles.footerLink}>Create an account</Text>
        </Text>
      </Pressable>
    </Screen>
  )
}

const styles = StyleSheet.create({
  header: { marginBottom: spacing.lg },
  brand: { ...typography.sectionTitle, color: theme.textPrimary, marginBottom: spacing.xl },
  brandAccent: { color: theme.primary },
  title: { ...typography.screenTitle, color: theme.textPrimary },
  subtitle: { ...typography.body, color: theme.textSecondary, marginTop: spacing.xs },
  form: { marginTop: spacing.sm },
  footer: { marginTop: spacing.xl, alignItems: 'center' },
  footerText: { ...typography.body, color: theme.textSecondary },
  footerLink: { color: theme.primaryText, fontWeight: '600' },
})
