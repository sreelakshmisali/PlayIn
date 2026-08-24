import { ActivityIndicator, Pressable, StyleSheet, Text, type StyleProp, type ViewStyle } from 'react-native'

import { buttonPresets, theme, typography } from '../theme'

type Variant = 'primary' | 'secondary' | 'danger'

interface ButtonProps {
  label: string
  onPress: () => void
  /** 'secondary' is the outline/neutral button — the one to reach for
   * next to a 'primary' action (e.g. "Cancel" beside "Confirm booking"). */
  variant?: Variant
  pending?: boolean
  disabled?: boolean
  /** Layout-only overrides (margin, width) for placing this button inside
   * a specific screen — never use this to change color/spacing/radius,
   * those come from the design system. */
  style?: StyleProp<ViewStyle>
}

/**
 * The one button component every screen uses, for every variant. A fixed
 * minimum height keeps every button a comfortable thumb target regardless
 * of label length, and the pending state disables the press target rather
 * than hiding it, so a double-tap during a submit cannot fire twice.
 */
export function Button({ label, onPress, variant = 'primary', pending = false, disabled = false, style }: ButtonProps) {
  const isDisabled = pending || disabled

  return (
    <Pressable
      onPress={onPress}
      disabled={isDisabled}
      accessibilityRole="button"
      accessibilityState={{ disabled: isDisabled, busy: pending }}
      style={({ pressed }) => [
        styles.base,
        buttonPresets.variant[variant],
        isDisabled && styles.disabled,
        pressed && !isDisabled && styles.pressed,
        style,
      ]}
    >
      {pending ? (
        <ActivityIndicator color={variant === 'secondary' ? theme.primary : theme.textOnPrimary} />
      ) : (
        <Text style={[styles.label, { color: buttonPresets.labelColor[variant] }]}>{label}</Text>
      )}
    </Pressable>
  )
}

const styles = StyleSheet.create({
  base: buttonPresets.base,
  pressed: { opacity: buttonPresets.pressedOpacity },
  disabled: { opacity: buttonPresets.disabledOpacity },
  label: { ...typography.button },
})
