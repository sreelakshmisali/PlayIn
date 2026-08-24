import { Easing } from 'react-native'

/**
 * Animation timing. This app doesn't run Reanimated — these are plain
 * durations/curves for the built-in `Animated` API (`LayoutAnimation`,
 * `Animated.timing`, etc.). Kept short: motion here is meant to acknowledge
 * a state change, not perform.
 */
export const durations = {
  fast: 120,
  base: 200,
  slow: 320,
} as const

export const easings = {
  standard: Easing.out(Easing.cubic),
  decelerate: Easing.out(Easing.quad),
  accelerate: Easing.in(Easing.quad),
} as const
