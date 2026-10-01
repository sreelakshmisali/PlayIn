import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { useState } from 'react'
import { Pressable, StyleSheet, View } from 'react-native'

import { Button, ErrorBanner, Screen, Text, TextField } from '../../components'
import { useAuth } from '../../hooks'
import { spacing, theme } from '../../theme'
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
    <Screen keyboardSafe contentStyle={styles.container}>
      <View style={styles.content}>
        <View style={styles.header}>
          <Text variant="sectionTitle" color="primary" style={styles.brand}>
            PlayHub<Text variant="sectionTitle" style={styles.brandAccent}>.</Text>
          </Text>
          <Text variant="screenTitle" color="primary">
            Sign in
          </Text>
          <Text variant="body" color="secondary" style={styles.subtitle}>
            Book turfs, join teams and track your game.
          </Text>
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
      </View>

      <Pressable onPress={() => navigation.navigate('Register')} style={styles.footer}>
        <Text variant="body" color="secondary">
          New to PlayHub? <Text variant="bodyEmphasized" color="primary">Create an account</Text>
        </Text>
      </Pressable>
    </Screen>
  )
}

const styles = StyleSheet.create({
  container: { justifyContent: 'space-between' },
  content: { flex: 1 },
  header: { marginBottom: spacing.lg },
  brand: { marginBottom: spacing.xl },
  brandAccent: { color: theme.primary },
  subtitle: { marginTop: spacing.xs },
  form: { marginTop: spacing.sm },
  footer: { marginTop: spacing.xl, paddingVertical: spacing.md, alignItems: 'center' },
})
