/**
 * Icon sizes, in px — pass straight to `@expo/vector-icons`' `size` prop.
 * Keeps icons consistent instead of every screen picking its own number.
 *
 * `xs` is for an icon inside an already-small badge/pill (e.g. a status
 * pill's leading icon) — `sm` is the floor for every other case, including
 * an inline icon next to caption text (a location pin, a calendar glyph,
 * a trailing chevron).
 */
export const iconSizes = {
  xs: 12,
  sm: 16,
  md: 20,
  lg: 24,
  xl: 32,
} as const
