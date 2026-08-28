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
 * session. White background, the "PlayHub" wordmark centered, and two very
 * low-contrast tinted shapes in the corners for a hint of atmosphere —
 * never competing with the wordmark for attention. No logo asset exists
 * yet; the wordmark stands in for it and should be swapped for the real
 * mark in place, without touching the animation around it.
 */
export function SplashScreen({ animateIn = true }: SplashScreenProps) {
  const reducedMotion = useReducedMotion()
  // Under Reduce Motion, skip the entrance choreography entirely and start
  // already in the settled end state — same outcome as `animateIn={false}`.
  const shouldAnimateIn = animateIn && !reducedMotion

  const atmosphere = useRef(new Animated.Value(shouldAnimateIn ? 0 : 1)).current
  const wordmarkOpacity = useRef(new Animated.Value(shouldAnimateIn ? 0 : 1)).current
  const wordmarkScale = useRef(new Animated.Value(shouldAnimateIn ? 0.92 : 1)).current

  useEffect(() => {
    if (!shouldAnimateIn) return

    // 2. A very subtle fade for the background atmosphere, starting immediately.
    Animated.timing(atmosphere, {
      toValue: 1,
      duration: durations.slow,
      easing: easings.standard,
      useNativeDriver: true,
    }).start()

    // 3. The wordmark fades and scales into place, slightly staggered after
    // the background so it reads as the focal point arriving second.
    // 4. A tiny secondary settle — the spring's own small overshoot — gives
    // it a touch of life without turning into a flashy animation.
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
  }, [shouldAnimateIn, atmosphere, wordmarkOpacity, wordmarkScale])

  return (
    <View style={styles.container}>
      <Animated.View
        style={[styles.blobTopRight, { opacity: Animated.multiply(atmosphere, 0.6) }]}
        pointerEvents="none"
      />
      <Animated.View
        style={[styles.blobBottomLeft, { opacity: Animated.multiply(atmosphere, 0.5) }]}
        pointerEvents="none"
      />
      <Animated.View style={{ opacity: wordmarkOpacity, transform: [{ scale: wordmarkScale }] }}>
        <Animated.Text style={styles.wordmark}>PlayIn</Animated.Text>
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
    overflow: 'hidden',
  },
  // Two soft, low-contrast circles bleeding off opposite corners — the
  // "turf atmosphere" hint. Plain tinted shapes, not turf/ball artwork:
  // anything more literal would compete with the wordmark.
  blobTopRight: {
    position: 'absolute',
    top: -120,
    right: -100,
    width: 280,
    height: 280,
    borderRadius: 140,
    backgroundColor: theme.primarySurface,
  },
  blobBottomLeft: {
    position: 'absolute',
    bottom: -140,
    left: -110,
    width: 320,
    height: 320,
    borderRadius: 160,
    backgroundColor: theme.primarySurface,
  },
  wordmark: { ...typography.display, color: theme.textPrimary },
})
