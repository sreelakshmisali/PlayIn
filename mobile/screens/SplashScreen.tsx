import { useEffect, useRef } from 'react'
import { Animated, StyleSheet, View } from 'react-native'

import { useReducedMotion } from '../hooks'
import { durations, easings, theme, typography } from '../theme'

interface SplashScreenProps {
  /** Off for the brief crossfade hand-off to real content once the app is
   * ready (see `navigation/RootNavigator.tsx`): renders already in its
   * settled end state, so only the container's own opacity animates,
   * rather than replaying the entrance. On (default) for the real,
   * first-launch splash. */
  animateIn?: boolean
}

/**
 * The app's entry screen, shown while `AuthProvider` resolves the stored
 * session. White background, the "PlayHub" wordmark centered.
 * No logo asset exists yet; the wordmark stands in for it.
 */
export function SplashScreen({ animateIn = true }: SplashScreenProps) {
  const reducedMotion = useReducedMotion()
  const shouldAnimateIn = animateIn && !reducedMotion

  const wordmarkOpacity = useRef(new Animated.Value(shouldAnimateIn ? 0 : 1)).current
  const wordmarkScale = useRef(new Animated.Value(shouldAnimateIn ? 0.92 : 1)).current

  useEffect(() => {
    if (!shouldAnimateIn) return

    Animated.sequence([
      Animated.delay(150),
      Animated.parallel([
        Animated.timing(wordmarkOpacity, {
          toValue: 1,
          duration: durations.slow,
          easing: easings.standard,
          useNativeDriver: true,
        }),
        Animated.spring(wordmarkScale, {
          toValue: 1,
          friction: 8,
          tension: 50,
          useNativeDriver: true,
        }),
      ]),
    ]).start()
  }, [shouldAnimateIn, wordmarkOpacity, wordmarkScale])

  return (
    <View style={styles.container}>
      <Animated.View style={{ opacity: wordmarkOpacity, transform: [{ scale: wordmarkScale }], flexDirection: 'row', alignItems: 'baseline' }}>
        <Animated.Text style={styles.wordmark}>
          PlayHub<Animated.Text style={{ color: theme.primary }}>.</Animated.Text>
        </Animated.Text>
      </Animated.View>
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: theme.background,
    alignItems: 'center',
    justifyContent: 'center',
  },
  wordmark: { ...typography.display, color: theme.textPrimary },
})
