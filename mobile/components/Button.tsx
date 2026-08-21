import { ActivityIndicator, Pressable, StyleSheet, Text } from 'react-native'

import { minTouchTarget, radius, spacing, theme, typography } from '../theme'

type Variant = 'primary' | 'secondary' | 'danger'

interface ButtonProps {
  label: string
  onPress: () => void
  variant?: Variant
  pending?: boolean
  disabled?: boolean
}

/**
 * The one button component every screen uses. A fixed minimum height keeps
 * every button a comfortable thumb target regardless of label length, and
 * the pending state disables the press target rather than hiding it, so a
 * double-tap during a submit cannot fire twice.
 */
export function Button({ label, onPress, variant = 'primary', pending = false, disabled = false }: ButtonProps) {
  const isDisabled = pending || disabled

  return (
    <Pressable
      onPress={onPress}
      disabled={isDisabled}
      accessibilityRole="button"
      accessibilityState={{ disabled: isDisabled, busy: pending }}
      style={({ pressed }) => [
        styles.base,
        variantStyles[variant],
        isDisabled && styles.disabled,
        pressed && !isDisabled && styles.pressed,
      ]}
    >
      {pending ? (
        <ActivityIndicator color={variant === 'secondary' ? theme.primary : theme.textOnPrimary} />
      ) : (
        <Text style={[styles.label, variant === 'secondary' && styles.labelSecondary, variant === 'danger' && styles.labelDanger]}>
          {label}
        </Text>
      )}
    </Pressable>
  )
}

const styles = StyleSheet.create({
  base: {
    minHeight: minTouchTarget + 4,
    borderRadius: radius.md,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: spacing.lg,
  },
  pressed: { opacity: 0.85 },
  disabled: { opacity: 0.5 },
  label: { ...typography.button, color: theme.textOnPrimary },
  labelSecondary: { color: theme.primaryText },
  labelDanger: { color: theme.danger },
})

const variantStyles = StyleSheet.create({
  primary: { backgroundColor: theme.primary },
  secondary: { backgroundColor: theme.surface, borderWidth: 1, borderColor: theme.border },
  danger: { backgroundColor: theme.dangerSurface, borderWidth: 1, borderColor: theme.dangerBorder },
})
