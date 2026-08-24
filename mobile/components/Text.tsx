import { Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from 'react-native'

import { theme, typography } from '../theme'

type Variant = keyof typeof typography

type Color = 'primary' | 'secondary' | 'muted' | 'disabled' | 'onPrimary' | 'danger' | 'success' | 'warning' | 'info'

const colorMap: Record<Color, string> = {
  primary: theme.textPrimary,
  secondary: theme.textSecondary,
  muted: theme.textMuted,
  disabled: theme.textDisabled,
  onPrimary: theme.textOnPrimary,
  danger: theme.dangerText,
  success: theme.successText,
  warning: theme.warningText,
  info: theme.infoText,
}

interface TextProps extends RNTextProps {
  /** Which named style from `theme/typography.ts` to render as. */
  variant?: Variant
  /** A semantic color keyword, resolved against `theme.*` — never pass a
   * raw color. Omit to inherit color from `style` (e.g. a variant's own
   * color, or a one-off case a screen genuinely needs). */
  color?: Color
  style?: StyleProp<TextStyle>
}

/**
 * The one text component every screen uses. Wraps React Native's `Text`
 * with a `variant` (from the typography scale) and a semantic `color` — so
 * a screen never hardcodes a font size, weight, line height, or color.
 *
 * `variant` defaults to `body`, `color` defaults to `primary`; both can be
 * omitted for the common case of a plain paragraph.
 */
export function Text({ variant = 'body', color = 'primary', style, ...rest }: TextProps) {
  return <RNText style={[typography[variant], { color: colorMap[color] }, style]} {...rest} />
}
