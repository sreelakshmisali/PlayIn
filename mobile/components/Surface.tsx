import type { ReactNode } from 'react'
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native'

import { cardPresets, theme } from '../theme'

type Variant = 'card' | 'muted' | 'elevated'

const variantStyle: Record<Variant, ViewStyle> = {
  card: cardPresets.card,
  muted: cardPresets.surfaceMuted,
  elevated: cardPresets.elevated,
}

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

/** The one card/surface container every screen uses. See `variant` above
 * for which one to reach for. */
export function Surface({ children, variant = 'card', onPress, style, accessibilityLabel }: SurfaceProps) {
  if (onPress) {
    return (
      <Pressable
        onPress={onPress}
        accessibilityRole="button"
        accessibilityLabel={accessibilityLabel}
        style={({ pressed }) => [variantStyle[variant], pressed && styles.pressed, style]}
      >
        {children}
      </Pressable>
    )
  }

  return <View style={[variantStyle[variant], style]}>{children}</View>
}

const styles = StyleSheet.create({
  pressed: { backgroundColor: theme.surfaceMuted },
})
