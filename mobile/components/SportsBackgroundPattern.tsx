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
  opacity: number
}

/**
 * Scattered sports objects across the entire screen:
 * - Football, basketball, tennis, cricket, badminton, football boots, whistle,
 *   water bottle, cones, goal/net, basketball hoop, running shoe, stopwatch,
 *   jersey, field markings, trophies, etc.
 * - Single monochrome tint derived from PlayHub pitch green.
 * - Calibrated opacity (0.14 - 0.20) for a visible yet calm, classy watermark effect.
 * - Denser toward edges, with clean negative space around the central logo.
 */
const PATTERN_ITEMS: PatternItem[] = [
  // --- TOP REGION (0% - 22%) ---
  { name: 'soccer', size: 24, top: '2%', left: '-1%', rotation: '-18deg', opacity: 0.20 },
  { name: 'shoe-cleat', size: 21, top: '8%', left: '12%', rotation: '25deg', opacity: 0.17 },
  { name: 'whistle-outline', size: 18, top: '3%', left: '26%', rotation: '-12deg', opacity: 0.18 },
  { name: 'badminton', size: 22, top: '2%', left: '46%', rotation: '15deg', opacity: 0.16 },
  { name: 'tennis-ball', size: 16, top: '7%', left: '62%', rotation: '-20deg', opacity: 0.17 },
  { name: 'basketball-hoop-outline', size: 23, top: '3%', right: '16%', rotation: '-10deg', opacity: 0.18 },
  { name: 'cricket', size: 24, top: '4%', right: '-2%', rotation: '30deg', opacity: 0.19 },

  // --- UPPER-MID REGION (14% - 30%) ---
  { name: 'traffic-cone', size: 19, top: '15%', left: '2%', rotation: '-14deg', opacity: 0.18 },
  { name: 'tshirt-crew-outline', size: 22, top: '15%', left: '22%', rotation: '12deg', opacity: 0.16 },
  { name: 'timer-outline', size: 18, top: '23%', left: '8%', rotation: '-22deg', opacity: 0.17 },
  { name: 'soccer-field', size: 25, top: '24%', left: '27%', rotation: '8deg', opacity: 0.15 },
  { name: 'tennis', size: 23, top: '14%', right: '23%', rotation: '-25deg', opacity: 0.17 },
  { name: 'basketball', size: 22, top: '13%', right: '3%', rotation: '18deg', opacity: 0.19 },
  { name: 'water-outline', size: 19, top: '22%', right: '14%', rotation: '15deg', opacity: 0.16 },
  { name: 'shoe-sneaker', size: 20, top: '25%', right: '-1%', rotation: '-30deg', opacity: 0.18 },

  // --- MID REGION (32% - 60% flanks around logo) ---
  { name: 'stadium-variant', size: 24, top: '34%', left: '-2%', rotation: '10deg', opacity: 0.17 },
  { name: 'trophy-outline', size: 20, top: '35%', left: '16%', rotation: '-15deg', opacity: 0.16 },
  { name: 'volleyball', size: 21, top: '44%', left: '3%', rotation: '24deg', opacity: 0.18 },
  { name: 'flag-variant-outline', size: 18, top: '53%', left: '14%', rotation: '-18deg', opacity: 0.16 },
  { name: 'whistle-outline', size: 17, top: '54%', left: '-1%', rotation: '32deg', opacity: 0.17 },

  { name: 'badminton', size: 21, top: '34%', right: '4%', rotation: '-22deg', opacity: 0.17 },
  { name: 'soccer', size: 23, top: '37%', right: '19%', rotation: '16deg', opacity: 0.15 },
  { name: 'cricket', size: 23, top: '45%', right: '2%', rotation: '-12deg', opacity: 0.18 },
  { name: 'tennis-ball', size: 16, top: '53%', right: '18%', rotation: '28deg', opacity: 0.16 },
  { name: 'traffic-cone', size: 19, top: '55%', right: '-1%', rotation: '-15deg', opacity: 0.17 },

  // --- VERY FAINT ACCENTS BEHIND CENTRAL TEXT ---
  { name: 'medal-outline', size: 16, top: '32%', left: '49%', rotation: '12deg', opacity: 0.10 },
  { name: 'timer-outline', size: 15, top: '62%', left: '49%', rotation: '-8deg', opacity: 0.10 },

  // --- LOWER-MID REGION (62% - 78%) ---
  { name: 'basketball', size: 22, top: '63%', left: '10%', rotation: '-20deg', opacity: 0.18 },
  { name: 'tennis', size: 23, top: '65%', left: '26%', rotation: '30deg', opacity: 0.15 },
  { name: 'shoe-cleat', size: 21, top: '72%', left: '3%', rotation: '14deg', opacity: 0.18 },
  { name: 'tshirt-crew-outline', size: 21, top: '73%', left: '20%', rotation: '-16deg', opacity: 0.16 },
  { name: 'stadium-variant', size: 24, top: '63%', right: '24%', rotation: '-10deg', opacity: 0.15 },
  { name: 'shoe-sneaker', size: 21, top: '64%', right: '7%', rotation: '25deg', opacity: 0.18 },
  { name: 'volleyball', size: 21, top: '72%', right: '2%', rotation: '-22deg', opacity: 0.18 },
  { name: 'water-outline', size: 19, top: '73%', right: '18%', rotation: '18deg', opacity: 0.16 },

  // --- BOTTOM REGION (78% - 99%) ---
  { name: 'soccer-field', size: 25, bottom: '16%', left: '9%', rotation: '-12deg', opacity: 0.16 },
  { name: 'basketball-hoop-outline', size: 23, bottom: '15%', right: '14%', rotation: '15deg', opacity: 0.17 },
  { name: 'badminton', size: 21, bottom: '9%', left: '-2%', rotation: '28deg', opacity: 0.18 },
  { name: 'cricket', size: 23, bottom: '8%', left: '17%', rotation: '-24deg', opacity: 0.17 },
  { name: 'whistle-outline', size: 18, bottom: '10%', left: '38%', rotation: '14deg', opacity: 0.17 },
  { name: 'trophy-outline', size: 21, bottom: '8%', right: '36%', rotation: '-16deg', opacity: 0.16 },
  { name: 'tennis', size: 23, bottom: '9%', right: '16%', rotation: '22deg', opacity: 0.18 },
  { name: 'soccer', size: 24, bottom: '8%', right: '-2%', rotation: '-18deg', opacity: 0.19 },
  { name: 'traffic-cone', size: 19, bottom: '2%', left: '8%', rotation: '16deg', opacity: 0.17 },
  { name: 'tennis-ball', size: 16, bottom: '2%', left: '30%', rotation: '-30deg', opacity: 0.17 },
  { name: 'flag-variant-outline', size: 18, bottom: '2%', left: '52%', rotation: '10deg', opacity: 0.16 },
  { name: 'shoe-cleat', size: 21, bottom: '2%', right: '28%', rotation: '-20deg', opacity: 0.18 },
  { name: 'basketball', size: 23, bottom: '1%', right: '4%', rotation: '32deg', opacity: 0.19 },
]

interface SportsBackgroundPatternProps {
  animated?: boolean
  style?: StyleProp<ViewStyle>
}

/**
 * A subtle, elegant sports/play pattern designed for PlayHub.
 *
 * Renders tiny sports objects scattered organically around the entire screen
 * with higher density along the edges and breathing room around the central brand.
 *
 * Uses one light monochrome tint derived from PlayHub's brand green.
 */
export function SportsBackgroundPattern({ animated = true, style }: SportsBackgroundPatternProps) {
  const reducedMotion = useReducedMotion()
  const ambientDrift = useRef(new Animated.Value(0)).current

  useEffect(() => {
    if (!animated || reducedMotion) return

    // Very slow, calm breathing loop that creates a subtle, relaxed atmosphere
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
    outputRange: [0.9, 1.0],
  })

  const patternTranslateY = ambientDrift.interpolate({
    inputRange: [0, 1],
    outputRange: [0, -3],
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
            opacity: item.opacity,
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
    width: '100%',
    height: '100%',
    overflow: 'hidden',
  },
  fill: {
    ...StyleSheet.absoluteFill,
    width: '100%',
    height: '100%',
  },
})
