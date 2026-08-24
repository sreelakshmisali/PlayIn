/**
 * PlayHub mobile design system — the single source of truth for every
 * visual token in the app. A screen or component should never hardcode a
 * color, size, or spacing value; it should import from here.
 *
 * Visual direction: white-first, minimal, calm sports aesthetic. White is
 * the default surface everywhere; `theme.surfaceMuted` is the only
 * "secondary surface" and should be used sparingly (see `cardPresets` in
 * `presets.ts`) — prefer a hairline border over a gray fill. Green
 * (`theme.primary`) is the one accent: primary actions, brand, success.
 * Nothing else should introduce a new hue without extending an existing
 * scale in `colors.ts`.
 *
 * Layout:
 * - colors.ts     → raw palette + semantic `theme` (background/text/status)
 * - typography.ts → fontSizes, fontWeights, lineHeights, named text styles
 * - spacing.ts     → spacing scale, border radius, minimum touch target
 * - shadows.ts     → elevation presets
 * - icons.ts       → icon size scale
 * - layout.ts      → screen padding, header/tab bar heights, content width
 * - animation.ts   → durations + easing curves
 * - presets.ts     → ready-made button/input/card style objects
 */
export { colors, theme } from './colors'
export { spacing, radius, minTouchTarget } from './spacing'
export { typography, fontSizes, fontWeights, lineHeights } from './typography'
export { shadows } from './shadows'
export { iconSizes } from './icons'
export { layout } from './layout'
export { durations, easings } from './animation'
export { buttonPresets, inputPresets, cardPresets } from './presets'
