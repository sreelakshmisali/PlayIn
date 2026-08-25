import { Ionicons } from '@expo/vector-icons'
import { StyleSheet, View } from 'react-native'

import { iconSizes, radius, spacing, theme } from '../theme'
import { Button } from './Button'
import { IconContainer } from './IconContainer'
import { Text } from './Text'

/** Which failure this is, purely for icon/headline — the caller decides
 * this from information it already has (e.g. `ApiError.isNetworkError`);
 * this component never inspects the error itself. */
type ErrorKind = 'generic' | 'network'

const KIND_CONFIG: Record<ErrorKind, { title: string; icon: keyof typeof Ionicons.glyphMap }> = {
  generic: { title: 'Something went wrong', icon: 'alert-circle-outline' },
  network: { title: "Can't connect", icon: 'cloud-offline-outline' },
}

interface ErrorBannerProps {
  message: string
  /** Optional retry action. When provided, a "Try again" button appears
   * below the message. The caller wires this to whatever load function
   * originally failed — the banner does not manage retry state. */
  onRetry?: () => void
  /** Override the retry button label. Defaults to "Try again". */
  retryLabel?: string
  /** 'generic' (default) or 'network' — swaps the icon and headline for a
   * connectivity failure (e.g. when the caught error's `isNetworkError` is
   * true). Only affects the full, `onRetry` layout. */
  kind?: ErrorKind
}

/**
 * An inline error surface for screen-level or section-level failures: a
 * network timeout, a failed fetch, an unexpected server error. Not for
 * field validation (that lives on `TextField`).
 *
 * Two visual modes, chosen automatically by whether `onRetry` is passed:
 * - Without retry: a compact tinted banner, same as before, for errors
 *   shown alongside other content (e.g. a form's top-level error).
 * - With retry: a slightly more spacious layout with icon and CTA, for
 *   errors that replace the screen's content entirely. `kind` further
 *   distinguishes a connectivity failure from any other server/unexpected
 *   error.
 */
export function ErrorBanner({ message, onRetry, retryLabel = 'Try again', kind = 'generic' }: ErrorBannerProps) {
  if (onRetry) {
    const config = KIND_CONFIG[kind]
    return (
      <View style={styles.fullContainer} accessibilityRole="alert">
        <IconContainer tone="danger" size="xl" style={styles.iconCircle}>
          <Ionicons name={config.icon} size={iconSizes.lg} color={theme.danger} />
        </IconContainer>
        <Text variant="sectionTitle" color="primary" style={styles.title}>
          {config.title}
        </Text>
        <Text variant="body" color="secondary" style={styles.message}>
          {message}
        </Text>
        <View style={styles.retryAction}>
          <Button label={retryLabel} onPress={onRetry} variant="secondary" />
        </View>
      </View>
    )
  }

  return (
    <View style={styles.bannerContainer} accessibilityRole="alert">
      <Ionicons name="alert-circle-outline" size={iconSizes.sm} color={theme.danger} />
      <Text variant="body" color="danger" style={styles.bannerText}>
        {message}
      </Text>
    </View>
  )
}

const styles = StyleSheet.create({
  // Compact inline banner (no retry)
  bannerContainer: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: spacing.sm,
    backgroundColor: theme.dangerSurface,
    borderRadius: radius.md,
    padding: spacing.md,
    marginBottom: spacing.lg,
  },
  bannerText: {
    flex: 1,
  },

  // Full error state (with retry)
  fullContainer: {
    paddingVertical: spacing.xxl,
    paddingHorizontal: spacing.lg,
    alignItems: 'center',
  },
  iconCircle: {
    marginBottom: spacing.lg,
  },
  title: {
    textAlign: 'center',
    marginBottom: spacing.xs,
  },
  message: {
    textAlign: 'center',
    maxWidth: 280,
  },
  retryAction: {
    marginTop: spacing.xl,
    alignSelf: 'stretch',
  },
})
