import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { useState } from 'react'
import { Pressable, StyleSheet, View } from 'react-native'

import { Button, ErrorBanner, PlayHubLogo, Text, TextField } from '../../components'
import { useAuth } from '../../hooks'
import { ApiError } from '../../services/api'
import { authTheme, shadows } from '../../theme'
import type { AuthStackParamList } from '../../navigation/types'
import type { Role } from '../../types/auth'
import { SELF_ASSIGNABLE_ROLES } from '../../types/auth'
import { AuthLayout } from './AuthLayout'
import { authStyles } from './authStyles'

type Props = NativeStackScreenProps<AuthStackParamList, 'Register'>

const ROLE_LABELS: Record<Role, string> = {
  PLAYER: 'Player',
  OWNER: 'Turf owner',
  ADMIN: 'Admin',
}

/**
 * Sign Up Screen
 *
 * Visual language perfectly matched to Sign In:
 * - Minimalist, classy, relaxed, and sporty
 * - Vertically and horizontally centered open layout
 * - Compact form structure that fits gracefully across phone sizes
 * - Smooth scroll & keyboard avoidance
 */
export function RegisterScreen({ navigation }: Props) {
  const { register } = useAuth()

  const [role, setRole] = useState<Role>('PLAYER')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  async function handleSubmit() {
    setPending(true)
    setError('')
    setFieldErrors({})

    const trimmedName = fullName.trim()
    const trimmedEmail = email.trim().toLowerCase()
    const errors: Record<string, string> = {}

    if (!trimmedName) {
      errors.full_name = 'Full name is required'
    }

    if (!trimmedEmail) {
      errors.email = 'Email is required'
    }

    if (!password) {
      errors.password = 'Password is required'
    } else if (password.length < 10) {
      errors.password = 'Password must be at least 10 characters'
    }

    if (!confirmPassword) {
      errors.confirm_password = 'Confirm password is required'
    } else if (password !== confirmPassword) {
      errors.confirm_password = 'Passwords do not match'
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors)
      setPending(false)
      return
    }

    try {
      await register({
        full_name: trimmedName,
        email: trimmedEmail,
        password,
        role,
      })
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
    <AuthLayout>
      {/* 1-3. Centered Brand & Welcoming Header */}
      <View style={authStyles.header}>
        <PlayHubLogo size="md" showWordmark={false} style={authStyles.logoMark} />
        <Text style={authStyles.title}>Create your account</Text>
        <Text style={authStyles.subtitle}>Find a game. Book a turf. Play.</Text>
      </View>

      {/* Form Area */}
      <View style={authStyles.form}>
        {error ? <ErrorBanner message={error} /> : null}

        {/* Role Selector Segmented Control */}
        <View style={styles.roleContainer}>
          <View style={styles.roleSegment}>
            {SELF_ASSIGNABLE_ROLES.map((option) => {
              const selected = option === role
              return (
                <Pressable
                  key={option}
                  onPress={() => setRole(option)}
                  accessibilityRole="radio"
                  accessibilityState={{ selected }}
                  style={[styles.roleOption, selected && styles.roleOptionSelected]}
                >
                  <Text
                    style={[
                      styles.roleText,
                      selected && styles.roleTextSelected,
                    ]}
                  >
                    {ROLE_LABELS[option]}
                  </Text>
                </Pressable>
              )
            })}
          </View>
        </View>

        {/* Full Name */}
        <TextField
          label="Full name"
          value={fullName}
          onChangeText={setFullName}
          error={fieldErrors.full_name}
          autoComplete="name"
          placeholder="Your name"
        />

        {/* Email */}
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

        {/* Password */}
        <TextField
          label="Password"
          value={password}
          onChangeText={setPassword}
          error={fieldErrors.password}
          secureTextEntry
          autoComplete="password-new"
          placeholder="At least 10 characters"
        />

        {/* Confirm Password */}
        <TextField
          label="Confirm password"
          value={confirmPassword}
          onChangeText={setConfirmPassword}
          error={fieldErrors.confirm_password}
          secureTextEntry
          autoComplete="password-new"
          placeholder="Re-enter your password"
        />

        {/* Create Account Primary Button */}
        <Button
          label="Create account"
          onPress={() => void handleSubmit()}
          pending={pending}
        />

        {/* Divider */}
        <View style={authStyles.dividerRow}>
          <View style={authStyles.dividerLine} />
          <Text style={authStyles.dividerText}>or</Text>
          <View style={authStyles.dividerLine} />
        </View>

        {/* Switch to Sign In */}
        <Pressable
          onPress={() => navigation.navigate('Login')}
          accessibilityRole="button"
          hitSlop={{ top: 8, bottom: 8, left: 12, right: 12 }}
          style={authStyles.footer}
        >
          <Text style={authStyles.footerText}>
            Already have an account? <Text style={authStyles.footerLink}>Sign in</Text>
          </Text>
        </Pressable>
      </View>
    </AuthLayout>
  )
}

const styles = StyleSheet.create({
  roleContainer: {
    marginBottom: authTheme.spacing.md,
  },
  roleSegment: {
    flexDirection: 'row',
    backgroundColor: authTheme.colors.surfaceMuted,
    borderWidth: 1,
    borderColor: authTheme.colors.border,
    borderRadius: authTheme.layout.borderRadius,
    padding: 3,
  },
  roleOption: {
    flex: 1,
    borderRadius: authTheme.layout.borderRadiusSmall,
    paddingVertical: authTheme.spacing.sm,
    alignItems: 'center',
    justifyContent: 'center',
  },
  roleOptionSelected: {
    backgroundColor: authTheme.colors.surface,
    ...shadows.sm,
  },
  roleText: {
    fontSize: 13,
    lineHeight: 18,
    fontWeight: '500',
    color: authTheme.colors.textSecondary,
  },
  roleTextSelected: {
    fontWeight: '600',
    color: authTheme.colors.textPrimary,
  },
})
