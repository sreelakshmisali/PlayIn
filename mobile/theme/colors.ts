/**
 * Design tokens, not a component library. Kept close to the web app's own
 * "pitch" palette (Tailwind `pitch`/`neutral` scales) so the two clients read
 * as the same product, without sharing any code between them.
 *
 * Palette philosophy (see mobile/theme/index.ts for the full contract):
 * - White is the primary surface everywhere. `neutral50`/`neutral100` are the
 *   only "secondary surface" tints — reach for a border before reaching for
 *   a gray fill, so the app doesn't turn into a stack of gray cards.
 * - Green (`pitch*`) is the one accent. It marks primary actions, the brand,
 *   and success — never a decorative color. Nothing else in the app should
 *   introduce a new hue; extend an existing scale instead.
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

  blue50: '#eff6ff',
  blue200: '#bfdbfe',
  blue700: '#1d4ed8',
  blue800: '#1e40af',

  white: '#ffffff',
} as const

export const theme = {
  background: colors.white,
  surface: colors.white,
  surfaceMuted: colors.neutral100,
  border: colors.neutral200,
  overlay: 'rgba(23, 23, 23, 0.45)', // neutral900 scrim, for sheets/modal backdrops
  // A light fill for a small badge sitting on top of a primary-filled
  // surface (e.g. an active tab's count badge) — white tinted onto the
  // accent rather than a second, unrelated color.
  overlayOnPrimary: 'rgba(255, 255, 255, 0.25)',

  textPrimary: colors.neutral900,
  textSecondary: colors.neutral600,
  textMuted: colors.neutral400,
  textDisabled: colors.neutral300,
  textOnPrimary: colors.white,

  primary: colors.pitch600,
  primaryPressed: colors.pitch700,
  primarySurface: colors.pitch50,
  primaryText: colors.pitch700,

  // Semantic status colors. `success` deliberately reuses the brand green
  // scale rather than introducing a second green — one accent, used for
  // both meanings, keeps the palette restrained.
  success: colors.pitch600,
  successSurface: colors.pitch50,
  successText: colors.pitch700,

  warning: colors.amber700,
  warningSurface: colors.amber50,
  warningText: colors.amber800,

  danger: colors.red700,
  dangerSurface: colors.red50,
  dangerBorder: colors.red200,
  dangerText: colors.red800,

  info: colors.blue700,
  infoSurface: colors.blue50,
  infoText: colors.blue800,
} as const
