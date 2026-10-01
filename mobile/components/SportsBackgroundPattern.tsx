import { useEffect, useRef } from 'react'
import { Animated, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native'
import { MaterialCommunityIcons } from '@expo/vector-icons'

import { useReducedMotion } from '../hooks'
import { authTheme } from '../theme'

type IconName = keyof typeof MaterialCommunityIcons.glyphMap

interface PatternItem {
  name: IconName
  size: number
  top?: `${number}%` | number
  bottom?: `${number}%` | number
  left?: `${number}%` | number
  right?: `${number}%` | number
  rotation: string
  opacity?: number
}

// Scattered around the perimeter edges with generous empty negative space in the center.
// Some items slightly clip off-screen for an authentic organic lifestyle aesthetic.
const PATTERN_ITEMS: PatternItem[] = [
  // Top edge & corners
  { name: 'soccer', size: 22, top: '4%', left: '-2%', rotation: '-18deg', opacity: 0.08 },
  { name: 'whistle-outline', size: 18, top: '6%', left: '28%', rotation: '15deg', opacity: 0.07 },
  { name: 'badminton', size: 20, top: '3%', right: '24%', rotation: '-25deg', opacity: 0.08 },
  { name: 'tennis-ball', size: 16, top: '5%', right: '-1%', rotation: '12deg', opacity: 0.07 },

  // Upper perimeter
  { name: 'shoe-cleat', size: 21, top: '14%', left: '8%', rotation: '28deg', opacity: 0.07 },
  { name: 'traffic-cone', size: 19, top: '15%', right: '8%', rotation: '-14deg', opacity: 0.07 },
  { name: 'soccer-field', size: 24, top: '22%', left: '-3%', rotation: '-12deg', opacity: 0.06 },
  { name: 'basketball', size: 22, top: '23%', right: '-2%', rotation: '32deg', opacity: 0.07 },

  // Mid-upper sides (keeping center 35%-65% clear)
  { name: 'timer-outline', size: 18, top: '32%', left: '5%', rotation: '-15deg', opacity: 0.06 },
  { name: 'cricket', size: 22, top: '31%', right: '6%', rotation: '40deg', opacity: 0.06 },

  // Mid-lower sides
  { name: 'water-outline', size: 19, top: '66%', left: '4%', rotation: '20deg', opacity: 0.07 },
  { name: 'shoe-sneaker', size: 20, top: '68%', right: '5%', rotation: '-22deg', opacity: 0.06 },

  // Lower perimeter
  { name: 'volleyball', size: 21, bottom: '22%', left: '-2%', rotation: '18deg', opacity: 0.07 },
  { name: 'stadium-variant', size: 24, bottom: '21%', right: '-3%', rotation: '-15deg', opacity: 0.06 },
  { name: 'tshirt-crew-outline', size: 20, bottom: '13%', left: '9%', rotation: '-10deg', opacity: 0.08 },
  { name: 'tennis', size: 22, bottom: '14%', right: '10%', rotation: '35deg', opacity: 0.07 },

  // Bottom edge & corners
  { name: 'trophy-outline', size: 20, bottom: '4%', left: '2%', rotation: '-12deg', opacity: 0.07 },
  { name: 'traffic-cone', size: 18, bottom: '5%', left: '32%', rotation: '22deg', opacity: 0.06 },
  { name: 'whistle-outline', size: 17, bottom: '4%', right: '30%', rotation: '-18deg', opacity: 0.07 },
  { name: 'soccer', size: 22, bottom: '3%', right: '-2%', rotation: '25deg', opacity: 0.08 },
]

interface SportsBackgroundPatternProps {
  animated?: boolean
  style?: StyleProp<ViewStyle>
}

/**
 * A subtle, elegant sports/play pattern designed for PlayHub.
 *
 * Renders tiny sports objects (soccer ball, basketball, tennis racket, whistle,
 * cleat, cone, stopwatch, shuttlecock, turf markings) scattered organically
 * around the perimeter of the screen.
 *
 * All icons use one extremely light monochrome tint derived from PlayHub's
 * pitch green (contrast is very low and restrained so it never overwhelms).
 */
export function SportsBackgroundPattern({ animated = true, style }: SportsBackgroundPatternProps) {
  const reducedMotion = useReducedMotion()
  const ambientDrift = useRef(new Animated.Value(0)).current

  useEffect(() => {
    if (!animated || reducedMotion) return

    // Very slow, calm breathing loop (6s duration) that creates a subtle, relaxed atmosphere
    const animation = Animated.loop(
      Animated.sequence([
        Animated.timing(ambientDrift, {
          toValue: 1,
          duration: 3500,
          useNativeDriver: true,
        }),
        Animated.timing(ambientDrift, {
          toValue: 0,
          duration: 3500,
          useNativeDriver: true,
        }),
      ]),
    )

    animation.start()
    return () => animation.stop()
  }, [animated, reducedMotion, ambientDrift])

  const patternOpacity = ambientDrift.interpolate({
    inputRange: [0, 1],
    outputRange: [0.85, 1.0],
  })

  const patternTranslateY = ambientDrift.interpolate({
    inputRange: [0, 1],
    outputRange: [0, -2],
  })

  return (
    <View style={[styles.container, style]} pointerEvents="none">
      <Animated.View
        style={[
          styles.fill,
          {
            opacity: animated && !reducedMotion ? patternOpacity : 1,
            transform: animated && !reducedMotion ? [{ translateY: patternTranslateY }] : [],
          },
        ]}
      >
        {PATTERN_ITEMS.map((item, index) => {
          const itemStyle = {
            position: 'absolute' as const,
            top: item.top,
            bottom: item.bottom,
            left: item.left,
            right: item.right,
            transform: [{ rotate: item.rotation }],
            opacity: item.opacity ?? 0.07,
          }

          return (
            <View key={`${item.name}-${index}`} style={itemStyle}>
              <MaterialCommunityIcons
                name={item.name}
                size={item.size}
                color={authTheme.colors.accent}
              />
            </View>
          )
        })}
      </Animated.View>
    </View>
  )
}

const styles = StyleSheet.create({
  container: {
    ...StyleSheet.absoluteFill,
    overflow: 'hidden',
  },
  fill: {
    flex: 1,
  },
})
