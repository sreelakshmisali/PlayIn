import type { ReactNode } from 'react'
import { StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native'

import { iconSizes, theme } from '../theme'

type Size = keyof typeof iconSizes
type Tone = 'default' | 'primary' | 'muted'

/** Container edge per size step — comfortably larger than the icon glyph
 * itself (from `iconSizes`) so the tinted badge has breathing room. */
const boxSize: Record<Size, number> = { sm: 28, md: 36, lg: 44, xl: 56 }

const toneStyle: Record<Tone, ViewStyle> = {
  default: { backgroundColor: theme.surfaceMuted },
  primary: { backgroundColor: theme.primarySurface },
  muted: { backgroundColor: 'transparent' },
}

interface IconContainerProps {
  /** The icon element itself (e.g. an `Ionicons`), sized separately via
   * `iconSizes` by the caller — this component only provides the badge. */
  children: ReactNode
  size?: Size
  tone?: Tone
  style?: StyleProp<ViewStyle>
}

/**
 * A sized, tinted circular badge for an icon — a list row's leading icon,
 * an empty state's illustration, a booking step's status marker.
 */
export function IconContainer({ children, size = 'md', tone = 'default', style }: IconContainerProps) {
  const dimension = boxSize[size]
  return (
    <View
      style={[
        styles.base,
        toneStyle[tone],
        { width: dimension, height: dimension, borderRadius: dimension / 2 },
        style,
      ]}
    >
      {children}
    </View>
  )
}

const styles = StyleSheet.create({
  base: { alignItems: 'center', justifyContent: 'center' },
})
