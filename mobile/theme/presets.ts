import type { TextStyle, ViewStyle } from 'react-native'

import { theme } from './colors'
import { typography } from './typography'
import { minTouchTarget, radius, spacing } from './spacing'
import { shadows } from './shadows'

/**
 * Ready-made style objects for the app's three recurring surface types —
 * buttons, inputs, cards. Built once here from the atomic tokens
 * (colors/spacing/radius/shadows) so every screen's buttons, inputs and
 * cards look identical without hand-rolling the same StyleSheet each time.
 *
 * `components/Button.tsx`, `TextField.tsx` and `TurfCard.tsx` already
 * consume these — a new screen should reach for those components first,
 * and fall back to these presets directly only for a one-off surface that
 * doesn't warrant its own component.
 */

type Variant = 'primary' | 'secondary' | 'danger'

export const buttonPresets = {
  base: {
    minHeight: minTouchTarget + 4,
    borderRadius: radius.md,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: spacing.lg,
  } satisfies ViewStyle,

  variant: {
    primary: { backgroundColor: theme.primary } satisfies ViewStyle,
    secondary: { backgroundColor: theme.surface, borderWidth: 1, borderColor: theme.border } satisfies ViewStyle,
    danger: { backgroundColor: theme.dangerSurface, borderWidth: 1, borderColor: theme.dangerBorder } satisfies ViewStyle,
  } satisfies Record<Variant, ViewStyle>,

  labelColor: {
    primary: theme.textOnPrimary,
    secondary: theme.primaryText,
    danger: theme.danger,
  } satisfies Record<Variant, string>,

  pressedOpacity: 0.85,
  disabledOpacity: 0.5,
} as const

export const inputPresets = {
  field: {
    borderWidth: 1,
    borderColor: theme.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    ...typography.input,
    color: theme.textPrimary,
    backgroundColor: theme.surface,
  } satisfies ViewStyle & TextStyle,
  /** Applied on top of `field` when the input has a validation error. */
  error: { borderColor: theme.danger } satisfies ViewStyle,
  /** Applied on top of `field` for a focused input. Not yet wired into
   * `TextField` (it has no focus state today) — available for a screen
   * that adds one. */
  focused: { borderColor: theme.primary } satisfies ViewStyle,
  /** Applied on top of `field` for a non-editable input. */
  disabled: { backgroundColor: theme.surfaceMuted, borderColor: theme.border } satisfies ViewStyle,
} as const

export const cardPresets = {
  /** The default card: a hairline border on a white surface. No shadow —
   * this is the one every list/detail screen should reach for first. */
  card: {
    backgroundColor: theme.surface,
    borderWidth: 1,
    borderColor: theme.border,
    borderRadius: radius.lg,
    padding: spacing.lg,
  } satisfies ViewStyle,
  /** A soft, borderless tinted surface for grouping content without the
   * weight of a bordered card — e.g. a summary block. Use sparingly: not
   * every section needs a surface. */
  surfaceMuted: {
    backgroundColor: theme.surfaceMuted,
    borderRadius: radius.lg,
    padding: spacing.lg,
  } satisfies ViewStyle,
  /** A genuinely elevated surface (sheet, floating panel). Rare — most of
   * the app should never need this. */
  elevated: {
    backgroundColor: theme.surface,
    borderRadius: radius.lg,
    padding: spacing.lg,
    ...shadows.md,
  } satisfies ViewStyle,
  /** A small rounded chip/pill, e.g. a sport tag or status badge. */
  pill: {
    borderRadius: radius.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
  } satisfies ViewStyle,
} as const
