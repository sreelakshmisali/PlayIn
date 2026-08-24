import { useEffect, useState } from 'react'
import { AccessibilityInfo } from 'react-native'

/**
 * Mirrors the OS's Reduce Motion accessibility setting (iOS Settings >
 * Accessibility > Motion; Android Settings > Accessibility > Remove
 * animations). Every animated interaction in the app (`usePressScale`,
 * `SplashScreen`, `RootNavigator`'s crossfade, selection pulses) reads this
 * and skips its transform/opacity animation when it's on, changing state
 * instantly instead. Starts `false` and updates once the OS answers, so a
 * user without the setting sees no flash of skipped animation.
 */
export function useReducedMotion(): boolean {
  const [reducedMotion, setReducedMotion] = useState(false)

  useEffect(() => {
    let mounted = true

    AccessibilityInfo.isReduceMotionEnabled().then((value) => {
      if (mounted) setReducedMotion(value)
    })

    const subscription = AccessibilityInfo.addEventListener('reduceMotionChanged', setReducedMotion)
    return () => {
      mounted = false
      subscription.remove()
    }
  }, [])

  return reducedMotion
}
