import { useEffect, useRef } from 'react'
import { Animated, StyleSheet, View } from 'react-native'

import { PlayHubLogo, SportsBackgroundPattern } from '../components'
import { useReducedMotion } from '../hooks'
import { authTheme, durations, easings } from '../theme'

interface SplashScreenProps {
  /** Off for the brief crossfade hand-off to real content once the app is
   * ready (see `navigation/RootNavigator.tsx`): renders already in its
   * settled end state, so only the container's own opacity animates,
   * rather than replaying the entrance. On (default) for the real,
   * first-launch splash. */
  animateIn?: boolean
}

/**
 * The app's entry screen, shown while session resolves.
 *
 * Visual hierarchy:
 * - A clean, off-white background with a whisper-light monochrome sports pattern
 *   scattered around the screen edges.
 * - Center composition with generous negative space:
 *   1. PlayHub geometric sports pitch mark
 *   2. PlayHub. wordmark
 *   3. Tagline: "Find. Book. Play."
 *
 * Smooth, premium animation: logo scales gently with a relaxed spring,
 * while background sports elements maintain an ultra-slow, ambient breath.
 */
export function SplashScreen({ animateIn = true }: SplashScreenProps) {
  const reducedMotion = useReducedMotion()
  const shouldAnimateIn = animateIn && !reducedMotion

  const logoOpacity = useRef(new Animated.Value(shouldAnimateIn ? 0 : 1)).current
  const logoScale = useRef(new Animated.Value(shouldAnimateIn ? 0.92 : 1)).current
  const taglineOpacity = useRef(new Animated.Value(shouldAnimateIn ? 0 : 1)).current

  useEffect(() => {
    if (!shouldAnimateIn) return

    Animated.sequence([
      Animated.delay(120),
      Animated.parallel([
        Animated.timing(logoOpacity, {
          toValue: 1,
          duration: durations.slow,
          easing: easings.standard,
          useNativeDriver: true,
        }),
        Animated.spring(logoScale, {
          toValue: 1,
          friction: 9,
          tension: 45,
          useNativeDriver: true,
        }),
      ]),
      Animated.timing(taglineOpacity, {
        toValue: 1,
        duration: durations.base,
        easing: easings.standard,
        useNativeDriver: true,
      }),
    ]).start()
  }, [shouldAnimateIn, logoOpacity, logoScale, taglineOpacity])

  return (
    <View style={styles.container}>
      {/* Subtle background sports pattern around edges */}
      <SportsBackgroundPattern animated={!reducedMotion} />

      {/* Centered Brand Presentation */}
      <Animated.View
        style={[
          styles.centerBranding,
          {
            opacity: logoOpacity,
            transform: [{ scale: logoScale }],
          },
        ]}
      >
        <PlayHubLogo
          size="lg"
          showWordmark
          showTagline
          tagline="Find. Book. Play."
        />
      </Animated.View>
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: authTheme.colors.background,
    alignItems: 'center',
    justifyContent: 'center',
  },
  centerBranding: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: authTheme.spacing.xl,
    zIndex: 2,
  },
})
