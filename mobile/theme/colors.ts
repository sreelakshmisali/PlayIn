/**
 * Design tokens, not a component library. Kept close to the web app's own
 * "pitch" palette (Tailwind `pitch`/`neutral` scales) so the two clients read
 * as the same product, without sharing any code between them.
 */
export const colors = {
  pitch50: '#f0fdf4',
  pitch100: '#dcfce7',
  pitch200: '#bbf7d0',
  pitch600: '#16a34a',
  pitch700: '#15803d',

  neutral50: '#fafafa',
  neutral100: '#f5f5f5',
  neutral200: '#e5e5e5',
  neutral300: '#d4d4d4',
  neutral400: '#a3a3a3',
  neutral500: '#737373',
  neutral600: '#525252',
  neutral700: '#404040',
  neutral800: '#262626',
  neutral900: '#171717',

  red50: '#fef2f2',
  red200: '#fecaca',
  red700: '#b91c1c',
  red800: '#991b1b',

  amber50: '#fffbeb',
  amber700: '#b45309',
  amber800: '#92400e',

  white: '#ffffff',
} as const

export const theme = {
  background: colors.white,
  surface: colors.white,
  surfaceMuted: colors.neutral100,
  border: colors.neutral200,

  textPrimary: colors.neutral900,
  textSecondary: colors.neutral600,
  textMuted: colors.neutral400,
  textOnPrimary: colors.white,

  primary: colors.pitch600,
  primaryPressed: colors.pitch700,
  primarySurface: colors.pitch50,
  primaryText: colors.pitch700,

  danger: colors.red700,
  dangerSurface: colors.red50,
  dangerBorder: colors.red200,
  dangerText: colors.red800,

  warningSurface: colors.amber50,
  warningText: colors.amber800,
} as const
