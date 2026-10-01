import { useEffect, useRef, useState } from 'react'
import { Animated, StyleSheet } from 'react-native'
import * as SplashScreenModule from 'expo-splash-screen'

import { useAuth, useReducedMotion } from '../hooks'
import { NotAvailableScreen, SplashScreen } from '../screens'
import { durations, easings } from '../theme'
import { AuthNavigator } from './AuthNavigator'
import { PlayerNavigator } from './PlayerNavigator'
import { OwnerNavigator } from './OwnerNavigator'

/**
 * Picks the whole navigator tree from the signed-in user's role, per the
 * mobile foundation's brief: authentication flow, player flow, owner flow.
 * ADMIN gets an honest "not on mobile yet" rather than being dropped into
 * either flow — the admin web app remains the tool for that role.
 */
export function RootNavigator() {
  const { status, user } = useAuth()

  // Once the session has resolved (first `loading` → `anonymous`/
  // `authenticated` transition), the splash screen crossfades out over the
  // real content instead of being swapped for it instantly — the splash's
  // own "transition smoothly into the app" step.
  const [showSplash, setShowSplash] = useState(true)
  const [minSplashElapsed, setMinSplashElapsed] = useState(false)
  const splashOpacity = useRef(new Animated.Value(1)).current
  const reducedMotion = useReducedMotion()

  // Ensure splash screen remains visible for a minimum duration (3000ms / 3 seconds)
  // so branding is clearly displayed on startup even when auth resolves instantly.
  useEffect(() => {
    const timer = setTimeout(() => {
      setMinSplashElapsed(true)
    }, 3000)
    return () => clearTimeout(timer)
  }, [])

  useEffect(() => {
    if (status === 'loading' || !minSplashElapsed || !showSplash) return

    // Dismiss native splash screen smoothly when session resolution & min timer complete
    SplashScreenModule.hideAsync().catch(() => {
      /* ignore error if already hidden */
    })

    // Under Reduce Motion, drop the splash immediately rather than
    // crossfading it out — same end state, no animated transform/opacity.
    if (reducedMotion) {
      setShowSplash(false)
      return
    }

    Animated.timing(splashOpacity, {
      toValue: 0,
      duration: durations.slow,
      easing: easings.standard,
      useNativeDriver: true,
    }).start(() => setShowSplash(false))
  }, [status, minSplashElapsed, showSplash, splashOpacity, reducedMotion])

  if (status === 'loading' || (!minSplashElapsed && showSplash)) {
    return <SplashScreen />
  }

  let content
  if (status === 'anonymous' || !user) {
    content = <AuthNavigator />
  } else {
    switch (user.role) {
      case 'PLAYER':
        content = <PlayerNavigator />
        break
      case 'OWNER':
        content = <OwnerNavigator />
        break
      default:
        content = <NotAvailableScreen />
    }
  }

  if (!showSplash) {
    return content
  }

  return (
    <>
      {content}
      <Animated.View style={[styles.overlay, { opacity: splashOpacity }]} pointerEvents="none">
        <SplashScreen animateIn={false} />
      </Animated.View>
    </>
  )
}

const styles = StyleSheet.create({
  overlay: { ...StyleSheet.absoluteFill },
})
