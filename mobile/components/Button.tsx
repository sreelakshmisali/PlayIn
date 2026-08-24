import { ActivityIndicator, Animated, Pressable, StyleSheet, Text, type StyleProp, type ViewStyle } from 'react-native'

import { usePressScale } from '../hooks'
import { buttonPresets, theme, typography } from '../theme'

type Variant = 'primary' | 'secondary' | 'danger'

const AnimatedPressable = Animated.createAnimatedComponent(Pressable)

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
 *
 * Press feedback is a small, fast scale-down (see `usePressScale`) layered
 * on top of the existing opacity dip — acknowledges the tap without any
 * bounce, and is skipped under Reduce Motion.
 */
export function Button({ label, onPress, variant = 'primary', pending = false, disabled = false, style }: ButtonProps) {
  const isDisabled = pending || disabled
  const { animatedStyle, onPressIn, onPressOut } = usePressScale()

  return (
    <AnimatedPressable
      onPress={onPress}
      onPressIn={isDisabled ? undefined : onPressIn}
      onPressOut={isDisabled ? undefined : onPressOut}
      disabled={isDisabled}
      accessibilityRole="button"
      accessibilityState={{ disabled: isDisabled, busy: pending }}
      style={({ pressed }: { pressed: boolean }) => [
        styles.base,
        buttonPresets.variant[variant],
        isDisabled && styles.disabled,
        pressed && !isDisabled && styles.pressed,
        animatedStyle,
        style,
      ]}
    >
      {pending ? (
        <ActivityIndicator color={variant === 'secondary' ? theme.primary : theme.textOnPrimary} />
      ) : (
        <Text style={[styles.label, { color: buttonPresets.labelColor[variant] }]}>{label}</Text>
      )}
    </AnimatedPressable>
  )
}

const styles = StyleSheet.create({
  base: buttonPresets.base,
  pressed: { opacity: buttonPresets.pressedOpacity },
  disabled: { opacity: buttonPresets.disabledOpacity },
  label: { ...typography.button },
})
