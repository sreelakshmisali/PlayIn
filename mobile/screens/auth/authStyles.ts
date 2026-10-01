import { StyleSheet } from 'react-native'

import { authTheme } from '../../theme'

/**
 * Shared layout and typography styles for PlayHub Authentication screens.
 * Ensures consistent alignment, whitespace, typography, and visual hierarchy.
 */
export const authStyles = StyleSheet.create({
  header: {
    alignItems: 'center',
    marginBottom: authTheme.spacing.xl,
  },
  logoMark: {
    marginBottom: authTheme.spacing.md,
  },
  title: {
    fontSize: 24,
    lineHeight: 30,
    fontWeight: '700',
    letterSpacing: -0.4,
    color: authTheme.colors.textPrimary,
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 14,
    lineHeight: 20,
    color: authTheme.colors.textSecondary,
    textAlign: 'center',
    marginTop: authTheme.spacing.xs,
  },
  form: {
    width: '100%',
  },
  forgotPasswordButton: {
    alignSelf: 'center',
    marginTop: authTheme.spacing.md,
    paddingVertical: authTheme.spacing.xs,
  },
  forgotPasswordText: {
    fontSize: 13,
    lineHeight: 18,
    color: authTheme.colors.textSecondary,
    fontWeight: '500',
  },
  dividerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginVertical: authTheme.spacing.lg,
  },
  dividerLine: {
    flex: 1,
    height: StyleSheet.hairlineWidth,
    backgroundColor: authTheme.colors.divider,
  },
  dividerText: {
    paddingHorizontal: authTheme.spacing.md,
    fontSize: 13,
    color: authTheme.colors.textMuted,
    textTransform: 'lowercase',
  },
  footer: {
    alignItems: 'center',
    paddingVertical: authTheme.spacing.xs,
  },
  footerText: {
    fontSize: 14,
    lineHeight: 20,
    color: authTheme.colors.textSecondary,
  },
  footerLink: {
    fontWeight: '600',
    color: authTheme.colors.accent,
  },
})
