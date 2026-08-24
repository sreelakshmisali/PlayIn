import { useRef } from 'react'
import { Animated } from 'react-native'

import { durations, easings } from '../theme'
import { useReducedMotion } from './useReducedMotion'

interface UsePressScaleOptions {
  /** Scale reached on press-in, eased back to 1 on press-out. Deliberately
   * small — this acknowledges a tap, it doesn't perform one. */
  toValue?: number
}

/**
 * Shared press-feedback animation for tappable surfaces — `Button`,
 * `Surface` (and everything built on it: `TurfCard`, booking-flow cards).
 * A fast, subtle scale-down on press-in and a matching ease back on
 * press-out; no bounce, no overshoot. Skips the animation (holds scale at
 * 1) when the OS's Reduce Motion setting is on, so the caller can spread
 * `onPressIn`/`onPressOut` onto its `Pressable` and the animated `style`
 * onto the same node unconditionally.
 */
export function usePressScale({ toValue = 0.98 }: UsePressScaleOptions = {}) {
  const reducedMotion = useReducedMotion()
  const scale = useRef(new Animated.Value(1)).current

  const onPressIn = () => {
    if (reducedMotion) return
    Animated.timing(scale, {
      toValue,
      duration: durations.fast,
      easing: easings.accelerate,
      useNativeDriver: true,
    }).start()
  }

  const onPressOut = () => {
    if (reducedMotion) return
    Animated.timing(scale, {
      toValue: 1,
      duration: durations.fast,
      easing: easings.standard,
      useNativeDriver: true,
    }).start()
  }

  return { animatedStyle: { transform: [{ scale }] }, onPressIn, onPressOut }
}
