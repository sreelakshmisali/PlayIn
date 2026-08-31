import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { useState } from 'react'
import { Pressable, StyleSheet, Text, View } from 'react-native'

import { Button, ErrorBanner, Screen, TextField } from '../../components'
import { useAuth } from '../../hooks'
import { spacing, radius, shadows, theme, typography } from '../../theme'
import { ApiError } from '../../services/api'
import type { AuthStackParamList } from '../../navigation/types'
import type { Role } from '../../types/auth'
import { SELF_ASSIGNABLE_ROLES } from '../../types/auth'

type Props = NativeStackScreenProps<AuthStackParamList, 'Register'>

const ROLE_LABELS: Record<Role, string> = { PLAYER: 'Player', OWNER: 'Turf owner', ADMIN: 'Admin' }

export function RegisterScreen({ navigation }: Props) {
  const { register } = useAuth()

  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('PLAYER')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  async function handleSubmit() {
    setPending(true)
    setError('')
    setFieldErrors({})
    try {
      await register({ full_name: fullName.trim(), email: email.trim().toLowerCase(), password, role })
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
        <Text style={styles.title}>Create your account</Text>
        <Text style={styles.subtitle}>Join as a player or list your turf as an owner.</Text>
      </View>

      <View style={styles.form}>
        {error ? <ErrorBanner message={error} /> : null}

        <Text style={styles.label}>I am a</Text>
        <View style={styles.roleRow}>
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
                <Text style={[styles.roleOptionText, selected && styles.roleOptionTextSelected]}>
                  {ROLE_LABELS[option]}
                </Text>
              </Pressable>
            )
          })}
        </View>

        <TextField
          label="Full name"
          value={fullName}
          onChangeText={setFullName}
          error={fieldErrors.full_name}
          autoComplete="name"
          placeholder="Your name"
        />
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
          autoComplete="password-new"
          placeholder="At least 10 characters"
        />

        <Button label="Create account" onPress={() => void handleSubmit()} pending={pending} />
      </View>

      <Pressable onPress={() => navigation.navigate('Login')} style={styles.footer}>
        <Text style={styles.footerText}>
          Already have an account? <Text style={styles.footerLink}>Sign in</Text>
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
  label: { ...typography.label, color: theme.textPrimary, marginBottom: spacing.sm },
  roleRow: { 
    flexDirection: 'row', 
    backgroundColor: theme.surfaceMuted,
    borderRadius: radius.md,
    padding: spacing.xs,
    marginBottom: spacing.lg 
  },
  roleOption: {
    flex: 1,
    borderRadius: radius.sm,
    paddingVertical: spacing.sm,
    alignItems: 'center',
  },
  roleOptionSelected: { backgroundColor: theme.surface, ...shadows.sm },
  roleOptionText: { ...typography.bodyEmphasized, color: theme.textSecondary },
  roleOptionTextSelected: { color: theme.textPrimary },
  footer: { marginTop: spacing.xl, alignItems: 'center' },
  footerText: { ...typography.body, color: theme.textSecondary },
  footerLink: { color: theme.primaryText, fontWeight: '600' },
})
