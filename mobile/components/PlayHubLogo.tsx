import { StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native'

import { authTheme, theme, typography } from '../theme'
import { Text } from './Text'

interface PlayHubLogoProps {
  /** Size variant:
   * - 'sm': compact (34px emblem, 18px text)
   * - 'md': standard auth header (44px emblem, 22px text)
   * - 'lg': splash display (64px emblem, 28px text)
   */
  size?: 'sm' | 'md' | 'lg'
  /** Whether to render the 'PlayHub.' wordmark. Defaults to true. */
  showWordmark?: boolean
  /** Whether to render the tagline beneath the wordmark. Defaults to false. */
  showTagline?: boolean
  /** Tagline text when showTagline is true. Defaults to "Find. Book. Play." */
  tagline?: string
  style?: StyleProp<ViewStyle>
}

/**
 * PlayHub brand logo and geometric sports emblem.
 *
 * Designed with a minimalist, classy sports visual language (reminiscent of Outsport):
 * a crisp pitch court geometry inside a tactile rounded badge, paired with
 * modern restrained typography and the signature brand green accent dot.
 */
export function PlayHubLogo({
  size = 'md',
  showWordmark = true,
  showTagline = false,
  tagline = 'Find. Book. Play.',
  style,
}: PlayHubLogoProps) {
  const badgeDimensions = {
    sm: { box: 34, radius: 9, pitchW: 20, pitchH: 24, circle: 8, stroke: 1.5 },
    md: { box: 44, radius: 12, pitchW: 24, pitchH: 30, circle: 10, stroke: 1.8 },
    lg: { box: 64, radius: 18, pitchW: 36, pitchH: 44, circle: 15, stroke: 2.2 },
  }[size]

  return (
    <View style={[styles.root, style]}>
      {/* Brand Geometric Sports Pitch Emblem */}
      <View
        style={[
          styles.badge,
          {
            width: badgeDimensions.box,
            height: badgeDimensions.box,
            borderRadius: badgeDimensions.radius,
          },
        ]}
        accessibilityRole="image"
        accessibilityLabel="PlayHub Sports Logo"
      >
        <View
          style={[
            styles.pitchBoundary,
            {
              width: badgeDimensions.pitchW,
              height: badgeDimensions.pitchH,
              borderRadius: badgeDimensions.radius * 0.45,
              borderWidth: badgeDimensions.stroke,
            },
          ]}
        >
          {/* Pitch Midline */}
          <View
            style={[
              styles.pitchMidline,
              {
                height: badgeDimensions.stroke,
              },
            ]}
          />
          {/* Pitch Center Circle */}
          <View
            style={[
              styles.pitchCenterCircle,
              {
                width: badgeDimensions.circle,
                height: badgeDimensions.circle,
                borderRadius: badgeDimensions.circle / 2,
                borderWidth: badgeDimensions.stroke,
              },
            ]}
          />
          {/* Center spot dot */}
          <View
            style={[
              styles.pitchDot,
              {
                width: badgeDimensions.stroke * 1.5,
                height: badgeDimensions.stroke * 1.5,
                borderRadius: badgeDimensions.stroke,
              },
            ]}
          />
        </View>
      </View>

      {/* Wordmark */}
      {showWordmark ? (
        <View style={styles.wordmarkContainer}>
          <Text
            style={[
              size === 'sm' && styles.wordmarkSm,
              size === 'md' && styles.wordmarkMd,
              size === 'lg' && styles.wordmarkLg,
            ]}
          >
            PlayHub
            <Text
              style={[
                styles.dotAccent,
                size === 'sm' && styles.wordmarkSm,
                size === 'md' && styles.wordmarkMd,
                size === 'lg' && styles.wordmarkLg,
              ]}
            >
              .
            </Text>
          </Text>

          {showTagline ? (
            <Text
              variant="caption"
              color="secondary"
              style={[styles.tagline, size === 'lg' && styles.taglineLg]}
            >
              {tagline}
            </Text>
          ) : null}
        </View>
      ) : null}
    </View>
  )
}

const styles = StyleSheet.create({
  root: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  badge: {
    backgroundColor: authTheme.colors.surfaceAccent,
    borderWidth: 1,
    borderColor: authTheme.colors.accentBorder,
    alignItems: 'center',
    justifyContent: 'center',
  },
  pitchBoundary: {
    borderColor: theme.primary,
    alignItems: 'center',
    justifyContent: 'center',
    position: 'relative',
  },
  pitchMidline: {
    position: 'absolute',
    left: 0,
    right: 0,
    backgroundColor: theme.primary,
  },
  pitchCenterCircle: {
    borderColor: theme.primary,
    alignItems: 'center',
    justifyContent: 'center',
    position: 'absolute',
  },
  pitchDot: {
    backgroundColor: theme.primary,
    position: 'absolute',
  },
  wordmarkContainer: {
    alignItems: 'center',
    marginTop: authTheme.spacing.sm,
  },
  wordmarkSm: {
    fontSize: 18,
    lineHeight: 22,
    fontWeight: '700',
    letterSpacing: -0.3,
    color: authTheme.colors.textPrimary,
  },
  wordmarkMd: {
    fontSize: 22,
    lineHeight: 26,
    fontWeight: '700',
    letterSpacing: -0.4,
    color: authTheme.colors.textPrimary,
  },
  wordmarkLg: {
    fontSize: 28,
    lineHeight: 34,
    fontWeight: '700',
    letterSpacing: -0.5,
    color: authTheme.colors.textPrimary,
  },
  dotAccent: {
    color: theme.primary,
  },
  tagline: {
    marginTop: authTheme.spacing.xs,
    letterSpacing: 0.5,
  },
  taglineLg: {
    marginTop: authTheme.spacing.sm,
    fontSize: 14,
    letterSpacing: 0.8,
  },
})
