import { colors, theme } from './colors'
import { radius, spacing } from './spacing'

/**
 * Reusable design tokens and layout constants for PlayHub Authentication
 * and Splash screens. Ensures strict visual consistency between Sign In,
 * Sign Up, and Splash screens, and provides a clear foundation for future UI screens.
 *
 * Aesthetic: Minimalist, classy, relaxed, and sporty — inspired by modern
 * outdoor/sports lifestyle products (e.g. Outsport).
 */
export const authTheme = {
  colors: {
    background: colors.white,
    surface: colors.white,
    surfaceMuted: colors.neutral50,
    surfaceAccent: colors.pitch50,
    accent: theme.primary,
    accentPressed: theme.primaryPressed,
    accentBorder: colors.pitch200,
    textPrimary: theme.textPrimary,
    textSecondary: theme.textSecondary,
    textMuted: theme.textMuted,
    border: theme.border,
    borderFocused: theme.primary,
    divider: theme.border,
    watermark: 'rgba(22, 163, 74, 0.08)',
  },
  layout: {
    containerMaxWidth: 400,
    inputHeight: 48,
    buttonHeight: 48,
    borderRadius: radius.md,
    borderRadiusSmall: radius.sm,
    logoAuthSize: 44,
    logoSplashSize: 64,
  },
  spacing: {
    xs: spacing.xs,
    sm: spacing.sm,
    md: spacing.md,
    lg: spacing.lg,
    xl: spacing.xl,
    xxl: spacing.xxl,
  },
} as const
