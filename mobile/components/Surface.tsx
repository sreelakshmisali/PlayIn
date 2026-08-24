import type { ReactNode } from 'react'
import { Animated, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native'

import { usePressScale } from '../hooks'
import { cardPresets, theme } from '../theme'

type Variant = 'card' | 'muted' | 'elevated'

const variantStyle: Record<Variant, ViewStyle> = {
  card: cardPresets.card,
  muted: cardPresets.surfaceMuted,
  elevated: cardPresets.elevated,
}

const AnimatedPressable = Animated.createAnimatedComponent(Pressable)

interface SurfaceProps {
  children: ReactNode
  /** 'card' (default) — a hairline-bordered white surface; the one to
   * reach for first. 'muted' — a soft tinted section with no border; use
   * sparingly, not every section needs a surface. 'elevated' — a floating
   * surface (a sheet); should be rare. */
  variant?: Variant
  /** Makes the whole surface tappable with a built-in pressed state — a
   * selectable slot card, a tappable summary row. Omit for a static
   * container. */
  onPress?: () => void
  style?: StyleProp<ViewStyle>
  accessibilityLabel?: string
}

/**
 * The one card/surface container every screen uses. See `variant` above
 * for which one to reach for. A tappable surface gets a small, fast scale
 * dip on press (see `usePressScale`) alongside the existing muted-fill
 * pressed state — the same feedback `Button` uses, so a turf card and a
 * primary button feel like the same interactive language. Skipped under
 * Reduce Motion.
 */
export function Surface({ children, variant = 'card', onPress, style, accessibilityLabel }: SurfaceProps) {
  const { animatedStyle, onPressIn, onPressOut } = usePressScale()

  if (onPress) {
    return (
      <AnimatedPressable
        onPress={onPress}
        onPressIn={onPressIn}
        onPressOut={onPressOut}
        accessibilityRole="button"
        accessibilityLabel={accessibilityLabel}
        style={({ pressed }: { pressed: boolean }) => [variantStyle[variant], pressed && styles.pressed, animatedStyle, style]}
      >
        {children}
      </AnimatedPressable>
    )
  }

  return <View style={[variantStyle[variant], style]}>{children}</View>
}

const styles = StyleSheet.create({
  pressed: { backgroundColor: theme.surfaceMuted },
})
