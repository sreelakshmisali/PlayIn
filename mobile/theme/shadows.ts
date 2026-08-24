import { Platform } from 'react-native'

import { colors } from './colors'

/**
 * Elevation presets, cross-platform (iOS `shadow*`, Android `elevation`).
 * Kept deliberately subtle — soft lift for a floating element, not a heavy
 * drop shadow. Most surfaces should use a hairline border (`theme.border`)
 * instead of a shadow; reach for these only for things that are genuinely
 * elevated above the page: a sheet, a floating action, an active/dragged
 * card.
 */
const shadow = (opacity: number, radius: number, offsetY: number, elevation: number) =>
  Platform.select({
    ios: {
      shadowColor: colors.neutral900,
      shadowOpacity: opacity,
      shadowRadius: radius,
      shadowOffset: { width: 0, height: offsetY },
    },
    android: { elevation },
    default: {},
  })!

export const shadows = {
  none: {},
  sm: shadow(0.05, 4, 1, 1),
  md: shadow(0.08, 8, 2, 3),
  lg: shadow(0.1, 16, 4, 6),
} as const
