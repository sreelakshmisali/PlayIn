import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { useState } from 'react'
import { Alert, Pressable, View } from 'react-native'

import { Button, ErrorBanner, PlayHubLogo, Text, TextField } from '../../components'
import { useAuth } from '../../hooks'
import { ApiError } from '../../services/api'
import type { AuthStackParamList } from '../../navigation/types'
import { AuthLayout } from './AuthLayout'
import { authStyles } from './authStyles'

type Props = NativeStackScreenProps<AuthStackParamList, 'Login'>

/**
 * Sign In Screen
 *
 * Minimalist, sporty, and classy aesthetic:
 * - Vertically and horizontally centered open layout
 * - Generous whitespace and restrained brand accent
 * - Tactile compact inputs with secure visibility toggle
 * - Clean visual hierarchy
 */
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

    const trimmedEmail = email.trim().toLowerCase()
    if (!trimmedEmail) {
      setFieldErrors({ email: 'Email is required' })
      setPending(false)
      return
    }

    if (!password) {
      setFieldErrors({ password: 'Password is required' })
      setPending(false)
      return
    }

    try {
      await login({ email: trimmedEmail, password })
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

  function handleForgotPassword() {
    Alert.alert(
      'Forgot Password',
      'Please reach out to support@playhub.com or contact your turf administrator to reset your account credentials.',
      [{ text: 'OK' }],
    )
  }

  return (
    <AuthLayout>
      {/* 1-3. Centered Brand & Welcoming Header */}
      <View style={authStyles.header}>
        <PlayHubLogo size="md" showWordmark={false} style={authStyles.logoMark} />
        <Text style={authStyles.title}>Welcome back</Text>
        <Text style={authStyles.subtitle}>Ready to get back in the game?</Text>
      </View>

      {/* Form Area */}
      <View style={authStyles.form}>
        {error ? <ErrorBanner message={error} /> : null}

        {/* 4. Email input */}
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

        {/* 5. Password input */}
        <TextField
          label="Password"
          value={password}
          onChangeText={setPassword}
          error={fieldErrors.password}
          secureTextEntry
          autoComplete="password"
          placeholder="Your password"
        />

        {/* 6. Primary Sign In button */}
        <Button label="Sign in" onPress={() => void handleSubmit()} pending={pending} />

        {/* 7. Forgot Password action */}
        <Pressable
          onPress={handleForgotPassword}
          accessibilityRole="button"
          accessibilityLabel="Forgot password"
          hitSlop={{ top: 8, bottom: 8, left: 12, right: 12 }}
          style={authStyles.forgotPasswordButton}
        >
          <Text style={authStyles.forgotPasswordText}>Forgot password?</Text>
        </Pressable>

        {/* 8. Divider */}
        <View style={authStyles.dividerRow}>
          <View style={authStyles.dividerLine} />
          <Text style={authStyles.dividerText}>or</Text>
          <View style={authStyles.dividerLine} />
        </View>

        {/* 10. Link to Sign Up */}
        <Pressable
          onPress={() => navigation.navigate('Register')}
          accessibilityRole="button"
          hitSlop={{ top: 8, bottom: 8, left: 12, right: 12 }}
          style={authStyles.footer}
        >
          <Text style={authStyles.footerText}>
            Don't have an account? <Text style={authStyles.footerLink}>Sign up</Text>
          </Text>
        </Pressable>
      </View>
    </AuthLayout>
  )
}
