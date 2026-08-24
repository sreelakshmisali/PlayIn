import type { TextStyle } from 'react-native'

/** The type scale, in px. Every font size in the app should come from here.
 * Deliberately short — a "large display" moment reaches for `display`, the
 * largest step, rather than a one-off number. */
export const fontSizes = {
  xs: 12,
  sm: 13,
  base: 14,
  md: 16,
  lg: 18,
  xl: 20,
  xxl: 24,
  display: 28,
} as const

/** React Native wants font weights as strings, not numbers. */
export const fontWeights = {
  regular: '400',
  medium: '500',
  semibold: '600',
  bold: '700',
} as const satisfies Record<string, TextStyle['fontWeight']>

/**
 * Line heights paired to each font size, roughly `fontSize + 6` (tighter at
 * the very small end, where extra leading just looks loose). Every named
 * style below sets its line height from this table instead of leaving it
 * to the platform default, so vertical rhythm is consistent regardless of
 * which style two adjacent lines of text use.
 */
export const lineHeights = {
  xs: 16,
  sm: 18,
  base: 20,
  md: 22,
  lg: 24,
  xl: 26,
  xxl: 30,
  display: 34,
} as const

/**
 * Named text styles as plain objects, spread into a component's own
 * StyleSheet rather than exported as styled components: this is a small
 * app, and one extra layer of text-component abstraction is not worth it
 * yet. Built from `fontSizes`/`fontWeights`/`lineHeights` above — a screen
 * reaching for a one-off size, weight or line height should add a case
 * here instead, not inline it.
 *
 * This is the full type system for the app. Modern/clean/premium comes
 * from restraint: one typeface (the platform system font), a short size
 * scale, four weights, and tight, deliberate negative tracking only on the
 * two largest styles — never decorative.
 */
export const typography = {
  /** Large display/title — a hero moment: a landing headline, an empty
   * state's lead line. The single largest style in the app; use sparingly,
   * at most once per screen. */
  display: { fontSize: fontSizes.display, lineHeight: lineHeights.display, fontWeight: fontWeights.bold, letterSpacing: -0.4 },

  /** Screen title — the name of the screen itself (a profile's name, a
   * screen header). */
  screenTitle: { fontSize: fontSizes.xxl, lineHeight: lineHeights.xxl, fontWeight: fontWeights.bold, letterSpacing: -0.3 },
  /** @deprecated use `screenTitle` — kept as an alias so existing screens
   * don't need to change; same values. */
  title: { fontSize: fontSizes.xxl, lineHeight: lineHeights.xxl, fontWeight: fontWeights.bold, letterSpacing: -0.3 },

  /** Section title — a heading inside a screen (a card's title, a group of
   * fields). Also usable as a list item's primary line. */
  sectionTitle: { fontSize: fontSizes.lg, lineHeight: lineHeights.lg, fontWeight: fontWeights.semibold },
  /** @deprecated use `sectionTitle` — kept as an alias; same values. */
  heading: { fontSize: fontSizes.lg, lineHeight: lineHeights.lg, fontWeight: fontWeights.semibold },

  /** Body — the default for paragraphs and read-only field values. */
  body: { fontSize: fontSizes.md, lineHeight: lineHeights.md, fontWeight: fontWeights.regular },

  /** Body emphasized — a body line that needs a touch more weight (a value
   * next to a muted label), without escalating to a heading. */
  bodyEmphasized: { fontSize: fontSizes.md, lineHeight: lineHeights.md, fontWeight: fontWeights.medium },
  /** @deprecated use `bodyEmphasized` — kept as an alias; same values. */
  bodyMedium: { fontSize: fontSizes.md, lineHeight: lineHeights.md, fontWeight: fontWeights.medium },

  /** Caption — secondary text under a title, a form field's error message. */
  caption: { fontSize: fontSizes.sm, lineHeight: lineHeights.sm, fontWeight: fontWeights.regular },

  /** Small metadata — the smallest text in the app: a timestamp, a byline,
   * a count. Never the only way to convey something important. */
  metadata: { fontSize: fontSizes.xs, lineHeight: lineHeights.xs, fontWeight: fontWeights.regular },

  /** Button text. */
  button: { fontSize: fontSizes.md, lineHeight: lineHeights.md, fontWeight: fontWeights.semibold },

  /** Input text — what the user types/reads inside a text field. Matches
   * `body`'s size so a field doesn't visually jump against surrounding
   * text, kept as its own named style since inputs are a distinct surface. */
  input: { fontSize: fontSizes.md, lineHeight: lineHeights.md, fontWeight: fontWeights.regular },

  /** A form field's own label — e.g. "Email", above a `TextField`. */
  label: { fontSize: fontSizes.base, lineHeight: lineHeights.base, fontWeight: fontWeights.semibold },

  /** Price/number emphasis — a turf's price, a slot count. Tabular figures
   * keep digits aligned when they change; the slight negative tracking and
   * bold weight is what gives a number its "premium" weight without
   * bumping the font size up. */
  priceEmphasis: {
    fontSize: fontSizes.xl,
    lineHeight: lineHeights.xl,
    fontWeight: fontWeights.bold,
    letterSpacing: -0.2,
    fontVariant: ['tabular-nums'],
  },
} satisfies Record<string, TextStyle>
